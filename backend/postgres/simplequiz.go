package postgres

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/user/go-expo-sample/backend/models"
)

// simpleQuizMetadata excludes frequently changing collections so submissions
// update one answer row rather than rewriting the full session history.
func simpleQuizMetadata(s *models.SimpleQuizSession) ([]byte, error) {
	v := *s
	v.Players = nil
	v.Answers = nil
	v.Grades = nil
	return json.Marshal(&v)
}

// SaveSimpleQuiz saves a single in-memory transition transactionally. Callers
// must serialize transitions per module and publish the new snapshot only after
// this function succeeds. A nil previous snapshot starts a fresh session.
func (s *Store) SaveSimpleQuiz(ctx context.Context, previous, next *models.SimpleQuizSession) error {
	metadata, err := simpleQuizMetadata(next)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM modules WHERE event_id=$1 AND id=$2 FOR UPDATE`, next.EventID, next.ModuleID).Scan(&locked); err != nil {
		return translate(err)
	}
	if previous == nil {
		if _, err = tx.Exec(ctx, `DELETE FROM simple_quiz_sessions WHERE module_id=$1`, next.ModuleID); err != nil {
			return translate(err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO simple_quiz_sessions(module_id,event_id,status,current_index,data,updated_at) VALUES ($1,$2,$3,$4,$5,$6)`, next.ModuleID, next.EventID, next.Status, next.CurrentIndex, metadata, next.UpdatedAt); err != nil {
			return translate(err)
		}
	} else {
		tag, updateErr := tx.Exec(ctx, `UPDATE simple_quiz_sessions SET status=$3,current_index=$4,data=$5,updated_at=$6 WHERE module_id=$1 AND event_id=$2`, next.ModuleID, next.EventID, next.Status, next.CurrentIndex, metadata, next.UpdatedAt)
		if err = affected(tag, updateErr); err != nil {
			return err
		}
	}
	oldPlayers := map[string]*models.SimpleQuizPlayer{}
	oldAnswers := map[string]models.SimpleQuizAnswer{}
	oldGrades := map[string]*models.SimpleQuizGrade{}
	if previous != nil {
		oldPlayers = previous.Players
		oldGrades = previous.Grades
		for _, a := range previous.Answers {
			oldAnswers[a.ID] = a
		}
	}
	for id, player := range next.Players {
		if reflect.DeepEqual(oldPlayers[id], player) {
			continue
		}
		raw, err := marshal(player)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO simple_quiz_players(module_id,id,nickname,secret,data) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(module_id,id) DO UPDATE SET nickname=excluded.nickname,secret=excluded.secret,data=excluded.data`, next.ModuleID, player.ID, player.Nickname, player.Secret, raw); err != nil {
			return translate(err)
		}
	}
	for id := range oldPlayers {
		if _, ok := next.Players[id]; !ok {
			if _, err = tx.Exec(ctx, `DELETE FROM simple_quiz_players WHERE module_id=$1 AND id=$2`, next.ModuleID, id); err != nil {
				return translate(err)
			}
		}
	}
	newAnswers := make(map[string]struct{}, len(next.Answers))
	for _, answer := range next.Answers {
		newAnswers[answer.ID] = struct{}{}
		if old, ok := oldAnswers[answer.ID]; ok && reflect.DeepEqual(old, answer) {
			continue
		}
		raw, err := marshal(answer)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO simple_quiz_answers(module_id,id,player_id,question_index,data) VALUES ($1,$2,$3,$4,$5) ON CONFLICT(module_id,id) DO UPDATE SET data=excluded.data`, next.ModuleID, answer.ID, answer.PlayerID, answer.QuestionIndex, raw); err != nil {
			return translate(err)
		}
	}
	for id := range oldAnswers {
		if _, ok := newAnswers[id]; !ok {
			if _, err = tx.Exec(ctx, `DELETE FROM simple_quiz_answers WHERE module_id=$1 AND id=$2`, next.ModuleID, id); err != nil {
				return translate(err)
			}
		}
	}
	for id, grade := range next.Grades {
		if reflect.DeepEqual(oldGrades[id], grade) {
			continue
		}
		raw, err := marshal(grade)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO simple_quiz_grades(module_id,answer_id,data) VALUES ($1,$2,$3) ON CONFLICT(module_id,answer_id) DO UPDATE SET data=excluded.data`, next.ModuleID, id, raw); err != nil {
			return translate(err)
		}
	}
	for id := range oldGrades {
		if _, ok := next.Grades[id]; !ok {
			if _, err = tx.Exec(ctx, `DELETE FROM simple_quiz_grades WHERE module_id=$1 AND answer_id=$2`, next.ModuleID, id); err != nil {
				return translate(err)
			}
		}
	}
	return translate(tx.Commit(ctx))
}

func (s *Store) ReadSimpleQuiz(ctx context.Context, moduleID string) (*models.SimpleQuizSession, error) {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var raw []byte
	if err = tx.QueryRow(ctx, `SELECT data FROM simple_quiz_sessions WHERE module_id=$1`, moduleID).Scan(&raw); err != nil {
		return nil, translate(err)
	}
	var session models.SimpleQuizSession
	if err = unmarshal(raw, &session); err != nil {
		return nil, err
	}
	session.Players = map[string]*models.SimpleQuizPlayer{}
	session.Answers = []models.SimpleQuizAnswer{}
	session.Grades = map[string]*models.SimpleQuizGrade{}
	rows, err := tx.Query(ctx, `SELECT data FROM simple_quiz_players WHERE module_id=$1 ORDER BY id`, moduleID)
	if err != nil {
		return nil, translate(err)
	}
	for rows.Next() {
		var value []byte
		var p models.SimpleQuizPlayer
		if err = rows.Scan(&value); err == nil {
			err = unmarshal(value, &p)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		session.Players[p.ID] = &p
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.Query(ctx, `SELECT data FROM simple_quiz_answers WHERE module_id=$1 ORDER BY question_index,id`, moduleID)
	if err != nil {
		return nil, translate(err)
	}
	for rows.Next() {
		var value []byte
		var a models.SimpleQuizAnswer
		if err = rows.Scan(&value); err == nil {
			err = unmarshal(value, &a)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		session.Answers = append(session.Answers, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.Query(ctx, `SELECT data FROM simple_quiz_grades WHERE module_id=$1 ORDER BY answer_id`, moduleID)
	if err != nil {
		return nil, translate(err)
	}
	for rows.Next() {
		var value []byte
		var g models.SimpleQuizGrade
		if err = rows.Scan(&value); err == nil {
			err = unmarshal(value, &g)
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		session.Grades[g.AnswerID] = &g
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if session.Revealed == nil {
		session.Revealed = map[int]bool{}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, translate(err)
	}
	return &session, nil
}

func (s *Store) DeleteSimpleQuiz(ctx context.Context, moduleID string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM simple_quiz_sessions WHERE module_id=$1`, moduleID)
	return translate(err)
}

// PauseRestoredSimpleQuiz persists the pause before allowing clients to resume.
func (s *Store) PauseRestoredSimpleQuiz(ctx context.Context, old *models.SimpleQuizSession) (*models.SimpleQuizSession, error) {
	if old.Status != models.SimpleQuizStateQuestionOpen {
		return old, nil
	}
	next := *old
	next.PausedFrom = old.Status
	next.Status = models.SimpleQuizStatePaused
	next.DeadlineUnix = 0
	next.UpdatedAt = time.Now().UTC()
	if err := s.SaveSimpleQuiz(ctx, old, &next); err != nil {
		return nil, err
	}
	return &next, nil
}
