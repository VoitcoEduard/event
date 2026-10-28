package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
)

func scanSession(row scanner) (*models.EventSession, error) {
	var v models.EventSession
	var order, states []byte
	if err := row.Scan(&v.ID, &v.EventID, &v.Status, &v.ActiveModuleID, &order, &states, &v.StartedAt, &v.EndedAt, &v.CreatedAt, &v.UpdatedAt); err != nil {
		return nil, translate(err)
	}
	if err := unmarshal(order, &v.ModuleOrder); err != nil {
		return nil, err
	}
	if err := unmarshal(states, &v.ModuleStates); err != nil {
		return nil, err
	}
	return &v, nil
}

const sessionColumns = `id::text,event_id::text,status,COALESCE(active_module_id::text,''),module_order,module_states,started_at,ended_at,created_at,updated_at`

func (s *Store) CreateEventSession(ctx context.Context, v *models.EventSession) error {
	order, err := marshal(v.ModuleOrder)
	if err != nil {
		return err
	}
	states, err := marshal(v.ModuleStates)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO event_sessions(id,event_id,status,active_module_id,module_order,module_states,started_at,ended_at,created_at,updated_at) VALUES ($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10)`, v.ID, v.EventID, v.Status, v.ActiveModuleID, order, states, v.StartedAt, v.EndedAt, v.CreatedAt, v.UpdatedAt)
	return translate(err)
}
func (s *Store) EventSession(ctx context.Context, id string) (*models.EventSession, error) {
	return scanSession(s.Pool.QueryRow(ctx, `SELECT `+sessionColumns+` FROM event_sessions WHERE id=$1`, id))
}
func (s *Store) EventSessions(ctx context.Context, eventID string) ([]models.EventSession, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+sessionColumns+` FROM event_sessions WHERE event_id=$1 ORDER BY created_at,id`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.EventSession{}
	for rows.Next() {
		v, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}
func (s *Store) UpdateEventSession(ctx context.Context, v *models.EventSession) error {
	var order, states []byte
	var err error
	if v.ModuleOrder != nil {
		order, err = marshal(v.ModuleOrder)
		if err != nil {
			return err
		}
	}
	if v.ModuleStates != nil {
		states, err = marshal(v.ModuleStates)
		if err != nil {
			return err
		}
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE event_sessions SET status=$2,active_module_id=COALESCE(NULLIF($3,'')::uuid,active_module_id),module_order=COALESCE($4::jsonb,module_order),module_states=COALESCE($5::jsonb,module_states),started_at=COALESCE($6,started_at),ended_at=COALESCE($7,ended_at),updated_at=$8 WHERE id=$1`, v.ID, v.Status, v.ActiveModuleID, order, states, v.StartedAt, v.EndedAt, v.UpdatedAt)
	return affected(tag, err)
}

func scanAttempt(row scanner) (*models.ModuleAttempt, error) {
	var a models.ModuleAttempt
	var data []byte
	if err := row.Scan(&a.ID, &a.EventID, &a.ParticipantID, &a.ModuleID, &a.Status, &a.ScoreDelta, &data, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, translate(err)
	}
	if err := unmarshal(data, &a.ResultData); err != nil {
		return nil, err
	}
	return &a, nil
}

const attemptColumns = `id::text,event_id::text,participant_id,module_id::text,status,score_delta,result_data,created_at,updated_at`

func (s *Store) CreateAttempt(ctx context.Context, a *models.ModuleAttempt) error {
	data, err := marshal(a.ResultData)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO module_attempts(id,event_id,participant_id,module_id,status,score_delta,result_data,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, a.ID, a.EventID, a.ParticipantID, a.ModuleID, a.Status, a.ScoreDelta, data, a.CreatedAt, a.UpdatedAt)
	return translate(err)
}
func (s *Store) Attempt(ctx context.Context, id string) (*models.ModuleAttempt, error) {
	return scanAttempt(s.Pool.QueryRow(ctx, `SELECT `+attemptColumns+` FROM module_attempts WHERE id=$1`, id))
}
func (s *Store) AttemptsForParticipant(ctx context.Context, eventID, participantID string) ([]models.ModuleAttempt, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+attemptColumns+` FROM module_attempts WHERE event_id=$1 AND participant_id=$2 ORDER BY created_at,id`, eventID, participantID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.ModuleAttempt{}
	for rows.Next() {
		v, err := scanAttempt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}
func (s *Store) UpdateAttempt(ctx context.Context, a *models.ModuleAttempt) error {
	data, err := marshal(a.ResultData)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE module_attempts SET status=$4,score_delta=$5,result_data=$6,updated_at=$7 WHERE id=$1 AND event_id=$2 AND module_id=$3`, a.ID, a.EventID, a.ModuleID, a.Status, a.ScoreDelta, data, a.UpdatedAt)
	return affected(tag, err)
}

func scanSharedState(row scanner) (*models.SharedStateValue, error) {
	var v models.SharedStateValue
	var consumers, value []byte
	if err := row.Scan(&v.ID, &v.EventSessionID, &v.Key, &v.Scope, &v.ProducerModuleID, &consumers, &v.ValueType, &value, &v.UpdatedAt); err != nil {
		return nil, translate(err)
	}
	if err := unmarshal(consumers, &v.ConsumerModuleIDs); err != nil {
		return nil, err
	}
	if err := unmarshal(value, &v.Value); err != nil {
		return nil, err
	}
	return &v, nil
}

const sharedStateColumns = `id::text,event_session_id::text,key,scope,producer_module_id,consumer_module_ids,value_type,value,updated_at`

func (s *Store) CreateSharedState(ctx context.Context, v *models.SharedStateValue) error {
	consumers, err := marshal(v.ConsumerModuleIDs)
	if err != nil {
		return err
	}
	value, err := marshal(v.Value)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO shared_state(id,event_session_id,key,scope,producer_module_id,consumer_module_ids,value_type,value,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, v.ID, v.EventSessionID, v.Key, v.Scope, v.ProducerModuleID, consumers, v.ValueType, value, v.UpdatedAt)
	return translate(err)
}
func (s *Store) SharedStateByKey(ctx context.Context, sessionID, key string) (*models.SharedStateValue, error) {
	return scanSharedState(s.Pool.QueryRow(ctx, `SELECT `+sharedStateColumns+` FROM shared_state WHERE event_session_id=$1 AND key=$2`, sessionID, key))
}
func (s *Store) SharedStateBySession(ctx context.Context, sessionID string) ([]models.SharedStateValue, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+sharedStateColumns+` FROM shared_state WHERE event_session_id=$1 ORDER BY key`, sessionID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.SharedStateValue{}
	for rows.Next() {
		v, err := scanSharedState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}
func (s *Store) UpdateSharedState(ctx context.Context, v *models.SharedStateValue) error {
	consumers, err := marshal(v.ConsumerModuleIDs)
	if err != nil {
		return err
	}
	value, err := marshal(v.Value)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE shared_state SET scope=$4,producer_module_id=$5,consumer_module_ids=$6,value_type=$7,value=$8,updated_at=$9 WHERE id=$1 AND event_session_id=$2 AND key=$3`, v.ID, v.EventSessionID, v.Key, v.Scope, v.ProducerModuleID, consumers, v.ValueType, value, v.UpdatedAt)
	return affected(tag, err)
}

func scanTemplate(row scanner) (*models.Template, error) {
	var v models.Template
	var data []byte
	if err := row.Scan(&v.ID, &v.Name, &v.Description, &v.ModuleType, &data, &v.CreatedAt); err != nil {
		return nil, translate(err)
	}
	if err := unmarshal(data, &v.Data); err != nil {
		return nil, err
	}
	return &v, nil
}

const templateColumns = `id::text,name,description,module_type,data,created_at`

func (s *Store) CreateTemplate(ctx context.Context, v *models.Template) error {
	data, err := marshal(v.Data)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO templates(id,name,description,module_type,data,created_at) VALUES ($1,$2,$3,$4,$5,$6)`, v.ID, v.Name, v.Description, v.ModuleType, data, v.CreatedAt)
	return translate(err)
}
func (s *Store) Templates(ctx context.Context) ([]models.Template, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+templateColumns+` FROM templates ORDER BY created_at,id`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.Template{}
	for rows.Next() {
		v, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}
func (s *Store) UpdateTemplate(ctx context.Context, id string, v *models.Template) error {
	data, err := marshal(v.Data)
	if err != nil {
		return err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE templates SET name=$2,description=$3,module_type=$4,data=$5 WHERE id=$1`, id, v.Name, v.Description, v.ModuleType, data)
	return affected(tag, err)
}
func (s *Store) DeleteTemplate(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM templates WHERE id=$1`, id)
	return affected(tag, err)
}

func (s *Store) ActiveCards(ctx context.Context, eventID string) ([]models.DynamicCard, error) {
	rows, err := s.Pool.Query(ctx, `SELECT data FROM dynamic_cards WHERE event_id=$1 AND is_active ORDER BY id`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.DynamicCard{}
	for rows.Next() {
		var raw []byte
		var v models.DynamicCard
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err = unmarshal(raw, &v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) InsertResponse(ctx context.Context, v *models.Response) error {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	data, err := marshal(v.Data)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if v.UserID != "" {
		name, _ := v.Data["step_welcome_name"].(string)
		name = strings.TrimSpace(name)
		_, err = tx.Exec(ctx, `INSERT INTO participants(event_id,id,display_name,normalized_name,profile,status,created_at,updated_at) VALUES ($1,$2,$3,$4,'{}','joined',$5,$5) ON CONFLICT(event_id,id) DO UPDATE SET display_name=CASE WHEN participants.display_name='' THEN excluded.display_name ELSE participants.display_name END, normalized_name=CASE WHEN participants.display_name='' THEN excluded.normalized_name ELSE participants.normalized_name END,updated_at=excluded.updated_at`, v.EventID, v.UserID, name, strings.ToLower(name), v.CreatedAt)
		if err != nil {
			return translate(err)
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO responses(id,event_id,module_id,user_id,data,created_at) VALUES ($1,$2,$3,$4,$5,$6)`, v.ID, v.EventID, v.ModuleID, v.UserID, data, v.CreatedAt); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}

func scanResponse(row scanner) (models.Response, error) {
	var v models.Response
	var data []byte
	if err := row.Scan(&v.ID, &v.EventID, &v.ModuleID, &v.UserID, &data, &v.CreatedAt); err != nil {
		return v, translate(err)
	}
	return v, unmarshal(data, &v.Data)
}
func (s *Store) Responses(ctx context.Context, eventID, moduleID string) ([]models.Response, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id::text,event_id::text,module_id::text,user_id,data,created_at FROM responses WHERE event_id=$1 AND module_id=$2 ORDER BY created_at,id`, eventID, moduleID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.Response{}
	for rows.Next() {
		v, err := scanResponse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) UpdateResponse(ctx context.Context, eventID, moduleID, responseID string, data map[string]any) (bool, error) {
	raw, err := marshal(data)
	if err != nil {
		return false, err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE responses SET data=$4 WHERE event_id=$1 AND module_id=$2 AND id=$3`, eventID, moduleID, responseID, raw)
	return tag.RowsAffected() > 0, translate(err)
}

func (s *Store) SaveTeamResult(ctx context.Context, v *models.TeamResult) error {
	if v.FinishedAt.IsZero() {
		v.FinishedAt = time.Now().UTC()
	}
	teams, err := marshal(v.Teams)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `INSERT INTO team_results(id,event_id,module_id,module_name,finished_at,teams) VALUES ($1,$2,$3,$4,$5,$6)`, uuid.NewString(), v.EventID, v.ModuleID, v.ModuleName, v.FinishedAt, teams)
	return translate(err)
}
func scanTeamResult(row scanner) (models.TeamResult, error) {
	var v models.TeamResult
	var teams []byte
	if err := row.Scan(&v.EventID, &v.ModuleID, &v.ModuleName, &v.FinishedAt, &teams); err != nil {
		return v, translate(err)
	}
	return v, unmarshal(teams, &v.Teams)
}
func (s *Store) TeamResults(ctx context.Context, eventID, moduleID string) ([]models.TeamResult, error) {
	rows, err := s.Pool.Query(ctx, `SELECT event_id::text,module_id::text,module_name,finished_at,teams FROM team_results WHERE event_id=$1 AND ($2='' OR module_id=NULLIF($2,'')::uuid) ORDER BY finished_at DESC,id`, eventID, moduleID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.TeamResult{}
	for rows.Next() {
		v, err := scanTeamResult(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) DeleteTeamResults(ctx context.Context, moduleID string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM team_results WHERE module_id=$1`, moduleID)
	return translate(err)
}
