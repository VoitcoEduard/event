package hub

import (
	"context"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
)

// DeleteSimpleQuiz removes the session (timers + undo + DB).
func (h *QuizHub) DeleteSimpleQuiz(moduleID string) {
	h.cancelSimpleQuizTimer(moduleID)
	h.mu.Lock()
	delete(h.simpleQuizSessions, moduleID)
	delete(h.simpleQuizUndo, moduleID)
	h.mu.Unlock()
	h.BroadcastToSession(moduleID, ServerMessage{
		Type:    "simple:state",
		Payload: map[string]interface{}{"status": "inactive"},
	})
}

// ApplySimpleGrade sets a verdict and pushes per-judge undo.
func (h *QuizHub) ApplySimpleGrade(moduleID, judgeToken, by, answerID, verdict string) *models.SimpleQuizSession {
	judgeKey := judgeToken + "|" + by
	h.mu.Lock()
	s, ok := h.simpleQuizSessions[moduleID]
	if !ok {
		h.mu.Unlock()
		return nil
	}
	prev := ""
	prevBy := ""
	var prevAt time.Time
	if g, ok := s.Grades[answerID]; ok {
		prev = g.Verdict
		prevBy = g.UpdatedBy
		prevAt = g.UpdatedAt
	}
	now := time.Now()
	g := s.Grades[answerID]
	if g == nil {
		g = &models.SimpleQuizGrade{AnswerID: answerID}
		s.Grades[answerID] = g
	}
	g.Verdict = verdict
	g.UpdatedBy = by
	g.UpdatedAt = now
	g.History = append(g.History, models.SimpleQuizGradeHistoryEntry{
		Verdict: verdict, By: by, At: now,
	})
	if h.simpleQuizUndo[moduleID] == nil {
		h.simpleQuizUndo[moduleID] = make(map[string][]simpleQuizUndoEntry)
	}
	h.simpleQuizUndo[moduleID][judgeKey] = append(h.simpleQuizUndo[moduleID][judgeKey], simpleQuizUndoEntry{
		AnswerID: answerID, Prev: prev, PrevBy: prevBy, PrevAt: prevAt,
		NewVerdict: verdict, At: now, By: by,
	})
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.persistSimpleQuiz(ctx, s)
	h.BroadcastSimpleQuiz(moduleID)
	return s
}

// PopSimpleUndo pops the latest grading action for one judge key.
func (h *QuizHub) PopSimpleUndo(moduleID, judgeToken, by string) (simpleQuizUndoEntry, bool) {
	judgeKey := judgeToken + "|" + by
	h.mu.Lock()
	defer h.mu.Unlock()
	stack := h.simpleQuizUndo[moduleID][judgeKey]
	if len(stack) == 0 {
		return simpleQuizUndoEntry{}, false
	}
	entry := stack[len(stack)-1]
	h.simpleQuizUndo[moduleID][judgeKey] = stack[:len(stack)-1]
	return entry, true
}

// ArmSimpleQuizTimer schedules auto-lock for an open question (exported for handlers).
func (h *QuizHub) ArmSimpleQuizTimer(moduleID string, deadlineUnix int64) {
	h.scheduleSimpleQuizAutoLock(moduleID, deadlineUnix)
}

// SimpleQuiz submit outcomes for SubmitSimpleQuizAnswer.
const (
	SimpleQuizSubmitOK            = "ok"
	SimpleQuizSubmitNoSession     = "no_session"
	SimpleQuizSubmitUnknownPlayer = "unknown_player"
	SimpleQuizSubmitDuplicate     = "duplicate"
	SimpleQuizSubmitClosed        = "closed"
	SimpleQuizSubmitWrongQuestion = "wrong_question"
	SimpleQuizSubmitDeadline      = "deadline"
	SimpleQuizSubmitInvalid       = "invalid_question"
)

// SubmitSimpleQuizAnswer validates and appends one answer atomically.
// Check-then-append runs under a single hub lock so concurrent requests with
// the same answerId (idempotency key) or the same player+question cannot
// both pass duplicate checks and both be stored.
// Returns the answerId to report to the caller plus an outcome code.
// On "duplicate" the returned id is the already-stored answer's id.
func (h *QuizHub) SubmitSimpleQuizAnswer(moduleID, playerID, secret, answerID string, questionIndex int, text string, now time.Time) (string, string) {
	if now.IsZero() {
		now = time.Now()
	}
	h.mu.Lock()
	s, ok := h.simpleQuizSessions[moduleID]
	if !ok {
		h.mu.Unlock()
		return "", SimpleQuizSubmitNoSession
	}
	p, ok := s.Players[playerID]
	if !ok || p.Secret != secret {
		h.mu.Unlock()
		return "", SimpleQuizSubmitUnknownPlayer
	}
	// Idempotent retry with same answerId.
	for _, a := range s.Answers {
		if a.ID == answerID {
			existing := a.ID
			h.mu.Unlock()
			return existing, SimpleQuizSubmitDuplicate
		}
	}
	if s.Status != models.SimpleQuizStateQuestionOpen {
		h.mu.Unlock()
		return "", SimpleQuizSubmitClosed
	}
	if questionIndex != s.CurrentIndex {
		h.mu.Unlock()
		return "", SimpleQuizSubmitWrongQuestion
	}
	if s.DeadlineUnix > 0 && now.Unix() > s.DeadlineUnix {
		h.mu.Unlock()
		return "", SimpleQuizSubmitDeadline
	}
	for _, a := range s.Answers {
		if a.PlayerID == playerID && a.QuestionIndex == questionIndex {
			existing := a.ID
			h.mu.Unlock()
			return existing, SimpleQuizSubmitDuplicate
		}
	}
	if questionIndex < 0 || questionIndex >= len(s.Questions) {
		h.mu.Unlock()
		return "", SimpleQuizSubmitInvalid
	}
	ans := models.SimpleQuizAnswer{
		ID:            answerID,
		PlayerID:      playerID,
		QuestionIndex: questionIndex,
		QuestionID:    s.Questions[questionIndex].ID,
		Text:          text,
		CreatedAt:     now,
	}
	s.Answers = append(s.Answers, ans)
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.persistSimpleQuiz(ctx, s)
	h.BroadcastSimpleQuiz(moduleID)
	return ans.ID, SimpleQuizSubmitOK
}

// SimpleQuiz join outcomes for JoinSimpleQuiz.
const (
	SimpleQuizJoinOK        = "ok"
	SimpleQuizJoinNoSession = "no_session"
	SimpleQuizJoinClosed    = "closed"
	SimpleQuizJoinTaken     = "taken"
)

// JoinSimpleQuiz performs reconnect-or-join atomically under a single hub
// lock, so concurrent joins with the same nickname cannot both pass the
// uniqueness check and both be stored. The nickname comparison is
// case-insensitive.
//
// nick must already be trimmed. providedSecret is the client-supplied
// reconnect secret (may be ""). newID/newSecret are the pre-generated
// identity for the fresh-join case; they are only consumed when a new player
// is actually inserted. Returns a copy of the joined player, whether it was
// a reconnect, and an outcome code.
func (h *QuizHub) JoinSimpleQuiz(moduleID, nick, providedSecret, newID, newSecret string, now time.Time) (models.SimpleQuizPlayer, bool, string) {
	if now.IsZero() {
		now = time.Now()
	}
	// Restore from DB on cold hub (same fallback as GetSimpleQuiz), keeping
	// the blocking DB read outside the hub lock.
	h.mu.RLock()
	_, ok := h.simpleQuizSessions[moduleID]
	h.mu.RUnlock()
	if !ok {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		restored := h.loadSimpleQuizFromDB(ctx, moduleID)
		cancel()
		if restored == nil {
			return models.SimpleQuizPlayer{}, false, SimpleQuizJoinNoSession
		}
		h.mu.Lock()
		if _, dup := h.simpleQuizSessions[moduleID]; !dup {
			h.simpleQuizSessions[moduleID] = restored
		}
		h.mu.Unlock()
	}

	h.mu.Lock()
	s, ok := h.simpleQuizSessions[moduleID]
	if !ok {
		h.mu.Unlock()
		return models.SimpleQuizPlayer{}, false, SimpleQuizJoinNoSession
	}
	if s.Status != models.SimpleQuizStateLobby &&
		s.Status != models.SimpleQuizStateQuestionOpen &&
		s.Status != models.SimpleQuizStateAnswersLocked &&
		s.Status != models.SimpleQuizStateBetweenQuestions {
		h.mu.Unlock()
		return models.SimpleQuizPlayer{}, false, SimpleQuizJoinClosed
	}
	// Reconnect with secret.
	if providedSecret != "" {
		for _, p := range s.Players {
			if p.Secret == providedSecret && strings.EqualFold(p.Nickname, nick) {
				p.Connected = true
				out := *p
				h.mu.Unlock()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				h.persistSimpleQuiz(ctx, s)
				cancel()
				h.BroadcastSimpleQuiz(moduleID)
				return out, true, SimpleQuizJoinOK
			}
		}
	}
	for _, p := range s.Players {
		if strings.EqualFold(p.Nickname, nick) {
			h.mu.Unlock()
			return models.SimpleQuizPlayer{}, false, SimpleQuizJoinTaken
		}
	}
	if s.Players == nil {
		s.Players = make(map[string]*models.SimpleQuizPlayer)
	}
	s.Players[newID] = &models.SimpleQuizPlayer{
		ID: newID, Nickname: nick, Secret: newSecret, JoinedAt: now, Connected: true,
	}
	out := *s.Players[newID]
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.persistSimpleQuiz(ctx, s)
	h.BroadcastSimpleQuiz(moduleID)
	return out, false, SimpleQuizJoinOK
}

// GenerateJudgeTokenForModule rotates the secret judge token in Module.Data.
func (h *QuizHub) GenerateJudgeTokenForModule(ctx context.Context, eventID, moduleID string) string {
	if db.Client == nil {
		return ""
	}
	token := GenerateJudgeToken()
	coll := db.GetCollection("events")
	filter := bson.M{"_id": eventID, "modules.id": moduleID}
	update := bson.M{"$set": bson.M{"modules.$.data.judgeToken": token}}
	if _, err := coll.UpdateOne(ctx, filter, update); err != nil {
		return ""
	}
	return token
}
