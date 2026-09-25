package hub

import (
	"context"
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
