package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/user/go-expo-sample/backend/models"
)

type scanner interface{ Scan(...any) error }
type eventQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

const eventColumns = `id::text, pin, name, description, language, is_active, created_at, screensaver_config, screen_state, slides, participant_policy, scoring_policy`
const moduleColumns = `id::text, type, is_active, name, data, inputs, outputs, activation_rules`

func scanEvent(row scanner) (*models.Event, error) {
	var e models.Event
	var screensaver, screen, slides, participant, scoring []byte
	if err := row.Scan(&e.ID, &e.Pin, &e.Name, &e.Description, &e.Language, &e.IsActive, &e.CreatedAt, &screensaver, &screen, &slides, &participant, &scoring); err != nil {
		return nil, translate(err)
	}
	for _, f := range []struct {
		raw []byte
		dst any
	}{{screensaver, &e.ScreensaverConfig}, {screen, &e.ScreenState}, {slides, &e.Slides}, {participant, &e.ParticipantPolicy}, {scoring, &e.ScoringPolicy}} {
		if err := unmarshal(f.raw, f.dst); err != nil {
			return nil, fmt.Errorf("decode event %s: %w", e.ID, err)
		}
	}
	return &e, nil
}

func scanModule(row scanner) (models.Module, error) {
	var m models.Module
	var data, inputs, outputs, rules []byte
	if err := row.Scan(&m.ID, &m.Type, &m.IsActive, &m.Name, &data, &inputs, &outputs, &rules); err != nil {
		return m, translate(err)
	}
	for _, f := range []struct {
		raw []byte
		dst any
	}{{data, &m.Data}, {inputs, &m.Inputs}, {outputs, &m.Outputs}, {rules, &m.ActivationRules}} {
		if err := unmarshal(f.raw, f.dst); err != nil {
			return m, fmt.Errorf("decode module %s: %w", m.ID, err)
		}
	}
	return m, nil
}

func loadModules(ctx context.Context, q eventQuery, e *models.Event) error {
	rows, err := q.Query(ctx, `SELECT `+moduleColumns+` FROM modules WHERE event_id=$1 ORDER BY position`, e.ID)
	if err != nil {
		return translate(err)
	}
	defer rows.Close()
	e.Modules = make([]models.Module, 0)
	for rows.Next() {
		m, err := scanModule(rows)
		if err != nil {
			return err
		}
		e.Modules = append(e.Modules, m)
	}
	return rows.Err()
}

func insertModule(ctx context.Context, q eventQuery, eventID string, position int64, m models.Module) error {
	if m.Data == nil {
		m.Data = map[string]any{}
	}
	data, err := marshal(m.Data)
	if err != nil {
		return err
	}
	inputs, err := marshal(m.Inputs)
	if err != nil {
		return err
	}
	outputs, err := marshal(m.Outputs)
	if err != nil {
		return err
	}
	rules, err := marshal(m.ActivationRules)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `INSERT INTO modules (id,event_id,position,type,name,is_active,data,inputs,outputs,activation_rules) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, m.ID, eventID, position, m.Type, m.Name, m.IsActive, data, inputs, outputs, rules)
	return translate(err)
}

func (s *Store) CreateEvent(ctx context.Context, e *models.Event) error {
	screensaver, err := marshal(e.ScreensaverConfig)
	if err != nil {
		return err
	}
	screen, err := marshal(e.ScreenState)
	if err != nil {
		return err
	}
	slides, err := marshal(e.Slides)
	if err != nil {
		return err
	}
	participant, err := marshal(e.ParticipantPolicy)
	if err != nil {
		return err
	}
	scoring, err := marshal(e.ScoringPolicy)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO events (id,pin,name,description,language,is_active,created_at,screensaver_config,screen_state,slides,participant_policy,scoring_policy) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, e.ID, e.Pin, e.Name, e.Description, e.Language, e.IsActive, e.CreatedAt, screensaver, screen, slides, participant, scoring)
	if err != nil {
		return translate(err)
	}
	for i, m := range e.Modules {
		if err := insertModule(ctx, tx, e.ID, int64(i), m); err != nil {
			return err
		}
	}
	return translate(tx.Commit(ctx))
}

func (s *Store) PINExists(ctx context.Context, pin string) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM events WHERE pin=$1)`, pin).Scan(&exists)
	return exists, translate(err)
}

func (s *Store) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	e, err := scanEvent(s.Pool.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE id=$1`, id))
	if err != nil {
		return nil, err
	}
	if err = loadModules(ctx, s.Pool, e); err != nil {
		return nil, err
	}
	return e, nil
}
func (s *Store) GetEventByPIN(ctx context.Context, pin string) (*models.Event, error) {
	e, err := scanEvent(s.Pool.QueryRow(ctx, `SELECT `+eventColumns+` FROM events WHERE pin=$1`, pin))
	if err != nil {
		return nil, err
	}
	if err = loadModules(ctx, s.Pool, e); err != nil {
		return nil, err
	}
	return e, nil
}
func (s *Store) GetEventByModule(ctx context.Context, id string) (*models.Event, error) {
	var eventID string
	if err := s.Pool.QueryRow(ctx, `SELECT event_id::text FROM modules WHERE id=$1`, id).Scan(&eventID); err != nil {
		return nil, translate(err)
	}
	return s.GetEvent(ctx, eventID)
}
func (s *Store) ListEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+eventColumns+` FROM events ORDER BY created_at DESC,id`)
	if err != nil {
		return nil, translate(err)
	}
	events := make([]models.Event, 0)
	for rows.Next() {
		e, scanErr := scanEvent(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		events = append(events, *e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range events {
		if err := loadModules(ctx, s.Pool, &events[i]); err != nil {
			return nil, err
		}
	}
	return events, nil
}
func affected(tag pgconn.CommandTag, err error) error {
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) UpdateEvent(ctx context.Context, id string, e *models.Event) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE events SET name=$2,description=$3,language=$4 WHERE id=$1`, id, e.Name, e.Description, e.Language)
	return affected(tag, err)
}
func (s *Store) SetEventStatus(ctx context.Context, id string, active bool) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE events SET is_active=$2 WHERE id=$1`, id, active)
	return affected(tag, err)
}
func (s *Store) SetEventScreensaver(ctx context.Context, id string, value models.ScreensaverConfig) error {
	raw, err := marshal(value)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE events SET screensaver_config=$2 WHERE id=$1`, id, raw)
	return affected(tag, err)
}
func (s *Store) SetEventSlides(ctx context.Context, id string, value []models.Slide) error {
	raw, err := marshal(value)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE events SET slides=$2 WHERE id=$1`, id, raw)
	return affected(tag, err)
}
func (s *Store) SetScreenState(ctx context.Context, id string, value models.ScreenState) error {
	raw, err := marshal(value)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE events SET screen_state=$2 WHERE id=$1`, id, raw)
	return affected(tag, err)
}
func (s *Store) DeleteEvent(ctx context.Context, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Preserve deletion markers after the foreign key clears event ownership.
	if _, err = tx.Exec(ctx, `DELETE FROM tracks WHERE media_id IN (SELECT id FROM media WHERE event_id=$1)`, id); err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM screensavers WHERE media_id IN (SELECT id FROM media WHERE event_id=$1)`, id); err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE media SET deleted_at=now() WHERE event_id=$1 AND deleted_at IS NULL`, id); err != nil {
		return translate(err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM events WHERE id=$1`, id)
	if err = affected(tag, err); err != nil {
		return err
	}
	return translate(tx.Commit(ctx))
}

func (s *Store) AddModule(ctx context.Context, eventID string, m models.Module) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM events WHERE id=$1 FOR UPDATE`, eventID).Scan(&id); err != nil {
		return translate(err)
	}
	var pos int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(position),-1)+1 FROM modules WHERE event_id=$1`, eventID).Scan(&pos); err != nil {
		return translate(err)
	}
	if err = insertModule(ctx, tx, eventID, pos, m); err != nil {
		return err
	}
	return translate(tx.Commit(ctx))
}
func (s *Store) GetModule(ctx context.Context, eventID, moduleID string) (*models.Module, error) {
	m, err := scanModule(s.Pool.QueryRow(ctx, `SELECT `+moduleColumns+` FROM modules WHERE event_id=$1 AND id=$2`, eventID, moduleID))
	if err != nil {
		return nil, err
	}
	return &m, nil
}
func (s *Store) SetModuleActive(ctx context.Context, eventID, moduleID string, active bool) (bool, error) {
	tag, err := s.Pool.Exec(ctx, `UPDATE modules SET is_active=$3 WHERE event_id=$1 AND id=$2`, eventID, moduleID, active)
	return tag.RowsAffected() > 0, translate(err)
}
func (s *Store) PublishModule(ctx context.Context, eventID, moduleID string, data map[string]any) (bool, error) {
	raw, err := marshal(data)
	if err != nil {
		return false, err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE modules SET data=$3 WHERE event_id=$1 AND id=$2`, eventID, moduleID, raw)
	return tag.RowsAffected() > 0, translate(err)
}
func (s *Store) PatchModule(ctx context.Context, eventID, moduleID string, data map[string]any) error {
	if data == nil {
		return nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE modules SET data=data || $3::jsonb WHERE event_id=$1 AND id=$2`, eventID, moduleID, raw)
	return affected(tag, err)
}
func (s *Store) DeleteModule(ctx context.Context, eventID, moduleID string) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var kind string
	err = tx.QueryRow(ctx, `SELECT type FROM modules WHERE event_id=$1 AND id=$2 FOR UPDATE`, eventID, moduleID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, translate(err)
	}
	if kind == string(models.ModuleTypeTeams) {
		var hasAwards bool
		err = tx.QueryRow(ctx, `SELECT jsonb_array_length(COALESCE(data->'awards','[]'::jsonb))>0 FROM teams_activities WHERE event_id=$1 AND module_id=$2 FOR UPDATE`, eventID, moduleID).Scan(&hasAwards)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return false, translate(err)
		}
		if hasAwards {
			return false, ErrHasAwards
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM tracks WHERE media_id IN (SELECT id FROM media WHERE event_id=$1 AND module_id=$2)`, eventID, moduleID); err != nil {
		return false, translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM screensavers WHERE media_id IN (SELECT id FROM media WHERE event_id=$1 AND module_id=$2)`, eventID, moduleID); err != nil {
		return false, translate(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE media SET deleted_at=now() WHERE event_id=$1 AND module_id=$2 AND deleted_at IS NULL`, eventID, moduleID); err != nil {
		return false, translate(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM modules WHERE event_id=$1 AND id=$2`, eventID, moduleID); err != nil {
		return false, translate(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return false, translate(err)
	}
	return true, nil
}

// EnsureJudgeToken atomically preserves an existing judge token during races.
func (s *Store) EnsureJudgeToken(ctx context.Context, eventID, moduleID, token string) (string, error) {
	var result string
	err := s.Pool.QueryRow(ctx, `UPDATE modules SET data=jsonb_set(data,'{judgeToken}',to_jsonb(COALESCE(NULLIF(data->>'judgeToken',''),$3::text)),true) WHERE event_id=$1 AND id=$2 RETURNING data->>'judgeToken'`, eventID, moduleID, token).Scan(&result)
	return result, translate(err)
}
