package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/user/go-expo-sample/backend/models"
)

// SaveQuizResult uses a caller-supplied stable run ID to make retries idempotent.
func (s *Store) SaveQuizResult(ctx context.Context, runID string, result *models.QuizResult) error {
	if _, err := uuid.Parse(runID); err != nil {
		return errors.New("valid run ID required")
	}
	signals, err := marshal(result.PlayerSignals)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM modules WHERE event_id=$1 AND id=$2 FOR UPDATE`, result.EventID, result.ModuleID).Scan(&locked); err != nil {
		return translate(err)
	}
	tag, err := tx.Exec(ctx, `INSERT INTO quiz_runs(id,event_id,module_id,finished_at,player_signals) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(id) DO NOTHING`, runID, result.EventID, result.ModuleID, result.FinishedAt, signals)
	if err != nil {
		return translate(err)
	}
	if tag.RowsAffected() == 0 {
		var existingEvent, existingModule string
		if err = tx.QueryRow(ctx, `SELECT event_id::text,module_id::text FROM quiz_runs WHERE id=$1`, runID).Scan(&existingEvent, &existingModule); err != nil {
			return translate(err)
		}
		if existingEvent != result.EventID || existingModule != result.ModuleID {
			return ErrConflict
		}
		return translate(tx.Commit(ctx))
	}
	for _, p := range result.Players {
		if _, err = tx.Exec(ctx, `INSERT INTO quiz_run_players(run_id,participant_id,nickname,score) VALUES ($1,$2,$3,$4)`, runID, p.ID, p.Nickname, p.Score); err != nil {
			return translate(err)
		}
	}
	return translate(tx.Commit(ctx))
}

type runRecord struct {
	ID     string
	Result models.QuizResult
}

func scanRun(row scanner) (runRecord, error) {
	var r runRecord
	var signals []byte
	if err := row.Scan(&r.ID, &r.Result.EventID, &r.Result.ModuleID, &r.Result.FinishedAt, &signals); err != nil {
		return r, translate(err)
	}
	if err := unmarshal(signals, &r.Result.PlayerSignals); err != nil {
		return r, err
	}
	return r, nil
}
func (s *Store) loadRunPlayers(ctx context.Context, r *runRecord) error {
	rows, err := s.Pool.Query(ctx, `SELECT participant_id,nickname,score FROM quiz_run_players WHERE run_id=$1 ORDER BY participant_id`, r.ID)
	if err != nil {
		return translate(err)
	}
	defer rows.Close()
	r.Result.Players = []models.PlayerResult{}
	for rows.Next() {
		var p models.PlayerResult
		if err = rows.Scan(&p.ID, &p.Nickname, &p.Score); err != nil {
			return err
		}
		r.Result.Players = append(r.Result.Players, p)
	}
	return rows.Err()
}
func (s *Store) queryRuns(ctx context.Context, query string, args ...any) ([]models.QuizResult, error) {
	rows, err := s.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, translate(err)
	}
	records := []runRecord{}
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		records = append(records, r)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	results := make([]models.QuizResult, 0, len(records))
	for i := range records {
		if err = s.loadRunPlayers(ctx, &records[i]); err != nil {
			return nil, err
		}
		results = append(results, records[i].Result)
	}
	return results, nil
}

const runColumns = `id::text,event_id::text,module_id::text,finished_at,player_signals`

func (s *Store) QuizResults(ctx context.Context, eventID, moduleID string) ([]models.QuizResult, error) {
	return s.queryRuns(ctx, `SELECT `+runColumns+` FROM quiz_runs WHERE event_id=$1 AND module_id=$2 ORDER BY finished_at DESC,sequence DESC`, eventID, moduleID)
}
func (s *Store) LatestQuizResults(ctx context.Context, eventID string) ([]models.QuizResult, error) {
	return s.queryRuns(ctx, `SELECT DISTINCT ON (r.module_id) r.id::text,r.event_id::text,r.module_id::text,r.finished_at,r.player_signals FROM quiz_runs r JOIN modules m ON m.event_id=r.event_id AND m.id=r.module_id WHERE r.event_id=$1 AND m.type IN ('quiz','wrong_pictures') ORDER BY r.module_id,r.finished_at DESC,r.sequence DESC`, eventID)
}
func (s *Store) UpdateLatestScore(ctx context.Context, eventID, moduleID, playerID string, score int) (bool, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var locked string
	err = tx.QueryRow(ctx, `SELECT id::text FROM modules WHERE event_id=$1 AND id=$2 FOR UPDATE`, eventID, moduleID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, translate(err)
	}
	tag, err := tx.Exec(ctx, `UPDATE quiz_run_players SET score=$4 WHERE participant_id=$3 AND run_id=(SELECT id FROM quiz_runs WHERE event_id=$1 AND module_id=$2 ORDER BY finished_at DESC,sequence DESC LIMIT 1)`, eventID, moduleID, playerID, score)
	if err != nil {
		return false, translate(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return false, translate(err)
	}
	return tag.RowsAffected() > 0, nil
}

func scanScore(row scanner) (models.ScoreEntry, error) {
	var e models.ScoreEntry
	err := row.Scan(&e.ID, &e.EventID, &e.ModuleID, &e.ParticipantID, &e.TeamID, &e.Category, &e.Points, &e.Reason, &e.CreatedAt)
	return e, translate(err)
}

const scoreColumns = `id,event_id::text,module_id::text,COALESCE(participant_id,''),COALESCE(team_id,''),category,points,reason,created_at`

func (s *Store) SaveScore(ctx context.Context, e *models.ScoreEntry, upsert bool) error {
	if math.IsNaN(e.Points) || math.IsInf(e.Points, 0) {
		return errors.New("points must be finite")
	}
	query := `INSERT INTO score_entries(id,event_id,module_id,participant_id,team_id,category,points,reason,created_at) VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9)`
	if upsert {
		query += ` ON CONFLICT(id) DO UPDATE SET points=excluded.points,reason=excluded.reason,created_at=excluded.created_at`
	}
	_, err := s.Pool.Exec(ctx, query, e.ID, e.EventID, e.ModuleID, e.ParticipantID, e.TeamID, e.Category, e.Points, e.Reason, e.CreatedAt)
	return translate(err)
}
func (s *Store) ScoreEntries(ctx context.Context, eventID string) ([]models.ScoreEntry, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+scoreColumns+` FROM score_entries WHERE event_id=$1 ORDER BY created_at,id`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.ScoreEntry{}
	for rows.Next() {
		e, err := scanScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (s *Store) SetManualScore(ctx context.Context, eventID, moduleID, target, targetID string, points float64) error {
	key, _ := json.Marshal([]string{eventID, moduleID, target, targetID})
	e := models.ScoreEntry{ID: "manual:" + string(key), EventID: eventID, ModuleID: moduleID, Category: models.ScoreCategoryAdjustment, Points: points, Reason: "Manual points", CreatedAt: time.Now().UTC()}
	switch target {
	case "players":
		e.ParticipantID = targetID
	case "teams":
		e.TeamID = targetID
	default:
		return errors.New("invalid scoring target")
	}
	return s.SaveScore(ctx, &e, true)
}

func scanActivity(row scanner) (*models.TeamsActivity, error) {
	var raw []byte
	var revision int64
	var deleted bool
	var status string
	if err := row.Scan(&raw, &revision, &deleted, &status); err != nil {
		return nil, translate(err)
	}
	var a models.TeamsActivity
	if err := unmarshal(raw, &a); err != nil {
		return nil, err
	}
	a.Revision = revision
	a.Deleted = deleted
	a.Status = status
	return &a, nil
}
func (s *Store) TeamsActivity(ctx context.Context, eventID, moduleID string) (*models.TeamsActivity, error) {
	return scanActivity(s.Pool.QueryRow(ctx, `SELECT data,revision,deleted,status FROM teams_activities WHERE event_id=$1 AND module_id=$2 AND NOT deleted`, eventID, moduleID))
}
func (s *Store) CreateTeamsActivity(ctx context.Context, a *models.TeamsActivity) error {
	data, err := marshal(a)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM modules WHERE event_id=$1 AND id=$2 FOR UPDATE`, a.EventID, a.ModuleID).Scan(&id); err != nil {
		return translate(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO teams_activities(id,event_id,module_id,status,revision,deleted,data) VALUES ($1,$2,$3,$4,$5,$6,$7)`, a.ID, a.EventID, a.ModuleID, a.Status, a.Revision, a.Deleted, data); err != nil {
		return translate(err)
	}
	return translate(tx.Commit(ctx))
}
func (s *Store) CompareAndSwapTeamsActivity(ctx context.Context, a *models.TeamsActivity, revision int64) (bool, error) {
	data, err := marshal(a)
	if err != nil {
		return false, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var id string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM modules WHERE event_id=$1 AND id=$2 FOR UPDATE`, a.EventID, a.ModuleID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, translate(err)
	}
	tag, err := tx.Exec(ctx, `UPDATE teams_activities SET status=$4,revision=$5,deleted=$6,data=$7 WHERE event_id=$1 AND module_id=$2 AND id=$3 AND revision=$8 AND NOT deleted`, a.EventID, a.ModuleID, a.ID, a.Status, a.Revision, a.Deleted, data, revision)
	if err != nil {
		return false, translate(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return false, translate(err)
	}
	return tag.RowsAffected() == 1, nil
}
func (s *Store) TeamsActivities(ctx context.Context, eventID string) ([]models.TeamsActivity, error) {
	rows, err := s.Pool.Query(ctx, `SELECT data,revision,deleted,status FROM teams_activities WHERE event_id=$1 AND NOT deleted ORDER BY module_id`, eventID)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()
	out := []models.TeamsActivity{}
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}
