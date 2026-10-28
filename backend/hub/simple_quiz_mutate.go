package hub

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

var errNoSimpleChange = errors.New("no change")

func (h *QuizHub) TryDeleteSimpleQuiz(moduleID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	var err error
	if h.db != nil {
		err = h.db.DeleteSimpleQuiz(ctx, moduleID)
	}
	if err == nil {
		h.cancelSimpleQuizTimer(moduleID)
		h.mu.Lock()
		delete(h.simpleQuizSessions, moduleID)
		delete(h.simpleQuizUndo, moduleID)
		h.mu.Unlock()
	}
	lock.Unlock()
	if err == nil {
		h.BroadcastToSession(moduleID, ServerMessage{Type: "simple:state", Payload: map[string]interface{}{"status": "inactive"}})
	}
	return err
}
func (h *QuizHub) DeleteSimpleQuiz(moduleID string) { _ = h.TryDeleteSimpleQuiz(moduleID) }

func (h *QuizHub) TryApplySimpleGrade(moduleID, judgeToken, by, answerID, verdict string) (*models.SimpleQuizSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	judgeKey := judgeToken + "|" + by
	var entry simpleQuizUndoEntry
	next, err := h.tryUpdateSimpleLocked(ctx, moduleID, func(s *models.SimpleQuizSession) error {
		found := false
		for _, answer := range s.Answers {
			if answer.ID == answerID {
				found = true
				break
			}
		}
		if !found {
			return postgres.ErrNotFound
		}
		entry = simpleQuizUndoEntry{AnswerID: answerID, NewVerdict: verdict, At: time.Now().UTC(), By: by}
		g := s.Grades[answerID]
		if g != nil {
			entry.Prev = g.Verdict
			entry.PrevBy = g.UpdatedBy
			entry.PrevAt = g.UpdatedAt
		} else {
			g = &models.SimpleQuizGrade{AnswerID: answerID}
			s.Grades[answerID] = g
		}
		g.Verdict = verdict
		g.UpdatedBy = by
		g.UpdatedAt = entry.At
		g.History = append(g.History, models.SimpleQuizGradeHistoryEntry{Verdict: verdict, By: by, At: entry.At})
		return nil
	})
	if err == nil {
		h.mu.Lock()
		if h.simpleQuizUndo[moduleID] == nil {
			h.simpleQuizUndo[moduleID] = map[string][]simpleQuizUndoEntry{}
		}
		h.simpleQuizUndo[moduleID][judgeKey] = append(h.simpleQuizUndo[moduleID][judgeKey], entry)
		h.mu.Unlock()
	}
	lock.Unlock()
	if err == nil {
		h.BroadcastSimpleQuiz(moduleID)
	}
	return next, err
}
func (h *QuizHub) ApplySimpleGrade(moduleID, judgeToken, by, answerID, verdict string) *models.SimpleQuizSession {
	s, _ := h.TryApplySimpleGrade(moduleID, judgeToken, by, answerID, verdict)
	return s
}

// UndoSimpleGrade persists the reverted grade before removing the undo entry.
func (h *QuizHub) UndoSimpleGrade(moduleID, judgeToken, by string) (string, string, *models.SimpleQuizSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	key := judgeToken + "|" + by
	h.mu.RLock()
	stack := h.simpleQuizUndo[moduleID][key]
	var entry simpleQuizUndoEntry
	if len(stack) > 0 {
		entry = stack[len(stack)-1]
	}
	h.mu.RUnlock()
	if len(stack) == 0 {
		lock.Unlock()
		return "", "", nil, postgres.ErrNotFound
	}
	next, err := h.tryUpdateSimpleLocked(ctx, moduleID, func(s *models.SimpleQuizSession) error {
		if entry.Prev == "" {
			delete(s.Grades, entry.AnswerID)
			return nil
		}
		g := s.Grades[entry.AnswerID]
		if g == nil {
			g = &models.SimpleQuizGrade{AnswerID: entry.AnswerID}
			s.Grades[entry.AnswerID] = g
		}
		g.Verdict = entry.Prev
		g.UpdatedBy = by + " (undo)"
		g.UpdatedAt = time.Now().UTC()
		g.History = append(g.History, models.SimpleQuizGradeHistoryEntry{Verdict: entry.Prev, By: g.UpdatedBy, At: g.UpdatedAt})
		return nil
	})
	if err == nil {
		h.mu.Lock()
		h.simpleQuizUndo[moduleID][key] = h.simpleQuizUndo[moduleID][key][:len(stack)-1]
		h.mu.Unlock()
	}
	lock.Unlock()
	if err == nil {
		h.BroadcastSimpleQuiz(moduleID)
	}
	return entry.AnswerID, entry.Prev, next, err
}

func (h *QuizHub) ArmSimpleQuizTimer(moduleID string, deadlineUnix int64) {
	h.scheduleSimpleQuizAutoLock(moduleID, deadlineUnix)
}

const (
	SimpleQuizSubmitOK            = "ok"
	SimpleQuizSubmitNoSession     = "no_session"
	SimpleQuizSubmitUnknownPlayer = "unknown_player"
	SimpleQuizSubmitDuplicate     = "duplicate"
	SimpleQuizSubmitClosed        = "closed"
	SimpleQuizSubmitWrongQuestion = "wrong_question"
	SimpleQuizSubmitDeadline      = "deadline"
	SimpleQuizSubmitInvalid       = "invalid_question"
	SimpleQuizSubmitStorageError  = "storage_error"
)

func (h *QuizHub) SubmitSimpleQuizAnswer(moduleID, playerID, secret, answerID string, questionIndex int, text string, now time.Time) (string, string) {
	if now.IsZero() {
		now = time.Now()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	code := SimpleQuizSubmitOK
	id := answerID
	_, err := h.tryUpdateSimpleLocked(ctx, moduleID, func(s *models.SimpleQuizSession) error {
		p, ok := s.Players[playerID]
		if !ok || p.Secret != secret {
			code = SimpleQuizSubmitUnknownPlayer
			return errNoSimpleChange
		}
		for _, a := range s.Answers {
			if a.ID == answerID {
				id = a.ID
				code = SimpleQuizSubmitDuplicate
				return errNoSimpleChange
			}
		}
		if s.Status != models.SimpleQuizStateQuestionOpen {
			code = SimpleQuizSubmitClosed
			return errNoSimpleChange
		}
		if questionIndex != s.CurrentIndex {
			code = SimpleQuizSubmitWrongQuestion
			return errNoSimpleChange
		}
		if s.DeadlineUnix > 0 && now.Unix() > s.DeadlineUnix {
			code = SimpleQuizSubmitDeadline
			return errNoSimpleChange
		}
		for _, a := range s.Answers {
			if a.PlayerID == playerID && a.QuestionIndex == questionIndex {
				id = a.ID
				code = SimpleQuizSubmitDuplicate
				return errNoSimpleChange
			}
		}
		if questionIndex < 0 || questionIndex >= len(s.Questions) {
			code = SimpleQuizSubmitInvalid
			return errNoSimpleChange
		}
		s.Answers = append(s.Answers, models.SimpleQuizAnswer{ID: answerID, PlayerID: playerID, QuestionIndex: questionIndex, QuestionID: s.Questions[questionIndex].ID, Text: text, CreatedAt: now})
		return nil
	})
	lock.Unlock()
	if errors.Is(err, postgres.ErrNotFound) {
		return "", SimpleQuizSubmitNoSession
	}
	if err != nil && !errors.Is(err, errNoSimpleChange) {
		return "", SimpleQuizSubmitStorageError
	}
	if code == SimpleQuizSubmitOK {
		h.BroadcastSimpleQuiz(moduleID)
	}
	if code != SimpleQuizSubmitOK && code != SimpleQuizSubmitDuplicate {
		return "", code
	}
	return id, code
}

const (
	SimpleQuizJoinOK           = "ok"
	SimpleQuizJoinNoSession    = "no_session"
	SimpleQuizJoinClosed       = "closed"
	SimpleQuizJoinTaken        = "taken"
	SimpleQuizJoinStorageError = "storage_error"
)

func (h *QuizHub) JoinSimpleQuiz(moduleID, nick, providedSecret, newID, newSecret string, now time.Time) (models.SimpleQuizPlayer, bool, string) {
	if now.IsZero() {
		now = time.Now()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	var out models.SimpleQuizPlayer
	var reconnected bool
	code := SimpleQuizJoinOK
	_, err := h.tryUpdateSimpleLocked(ctx, moduleID, func(s *models.SimpleQuizSession) error {
		if s.Status != models.SimpleQuizStateLobby && s.Status != models.SimpleQuizStateQuestionOpen && s.Status != models.SimpleQuizStateAnswersLocked && s.Status != models.SimpleQuizStateBetweenQuestions {
			code = SimpleQuizJoinClosed
			return errNoSimpleChange
		}
		if providedSecret != "" {
			for _, p := range s.Players {
				if p.Secret == providedSecret && strings.EqualFold(p.Nickname, nick) {
					p.Connected = true
					out = *p
					reconnected = true
					return nil
				}
			}
		}
		for _, p := range s.Players {
			if strings.EqualFold(p.Nickname, nick) {
				code = SimpleQuizJoinTaken
				return errNoSimpleChange
			}
		}
		if s.Players == nil {
			s.Players = map[string]*models.SimpleQuizPlayer{}
		}
		s.Players[newID] = &models.SimpleQuizPlayer{ID: newID, Nickname: nick, Secret: newSecret, JoinedAt: now, Connected: true}
		out = *s.Players[newID]
		return nil
	})
	lock.Unlock()
	if errors.Is(err, postgres.ErrNotFound) {
		return models.SimpleQuizPlayer{}, false, SimpleQuizJoinNoSession
	}
	if err != nil && !errors.Is(err, errNoSimpleChange) {
		return models.SimpleQuizPlayer{}, false, SimpleQuizJoinStorageError
	}
	if code == SimpleQuizJoinOK {
		h.BroadcastSimpleQuiz(moduleID)
	}
	return out, reconnected, code
}
func (h *QuizHub) GenerateJudgeTokenForModule(ctx context.Context, eventID, moduleID string) string {
	if h.db == nil {
		return ""
	}
	token := GenerateJudgeToken()
	if token == "" {
		return ""
	}
	if err := h.db.PatchModule(ctx, eventID, moduleID, map[string]any{"judgeToken": token}); err != nil {
		return ""
	}
	return token
}
