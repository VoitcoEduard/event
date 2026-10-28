package hub

import (
	"log"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

type quizTransition struct {
	timer    *time.Timer
	runID    string
	index    int
	deadline time.Time
}

// All timer-map changes use h.mu, including callbacks already queued by Stop.
func (h *QuizHub) cancelQuizTransitionLocked(moduleID string) {
	if pending := h.quizTimers[moduleID]; pending != nil {
		pending.timer.Stop()
		delete(h.quizTimers, moduleID)
	}
}

func (h *QuizHub) scheduleQuizTransitionLocked(session *models.QuizSession, delay time.Duration) {
	h.cancelQuizTransitionLocked(session.ModuleID)
	if h.closed {
		return
	}
	moduleID := session.ModuleID
	pending := &quizTransition{runID: session.RunID, index: session.CurrentIndex, deadline: time.Now().Add(delay)}
	h.quizTimers[moduleID] = pending
	pending.timer = time.AfterFunc(delay, func() {
		h.activateQuizQuestion(moduleID, pending)
	})
}

func (h *QuizHub) activateQuizQuestion(moduleID string, pending *quizTransition) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in question-timer for module %s: %v", moduleID, r)
		}
	}()
	h.mu.Lock()
	defer h.mu.Unlock()
	// Identity also rejects callbacks from a replaced timer for the same question.
	if h.closed || h.quizTimers[moduleID] != pending {
		return
	}
	delete(h.quizTimers, moduleID)
	session := h.sessions[moduleID]
	idx := pending.index
	if session == nil || session.RunID != pending.runID || session.CurrentIndex != idx ||
		session.State != "question-presenting" || idx < 0 || idx >= len(session.Questions) {
		return
	}
	session.State = "question-active"
	session.StartTime = time.Now().Unix()
	limit := EffectiveTimeLimit(session.Questions[idx])
	if idx < len(session.QuestionWindows) {
		window := &session.QuestionWindows[idx]
		window.StartTime = session.StartTime
		window.EndTime = session.StartTime + limit
		window.GraceEndTime = window.EndTime + GracePeriodSeconds
	}
	h.broadcastToSessionLocked(moduleID, ServerMessage{
		Type: "cmd:state",
		Payload: map[string]interface{}{
			"state": "question-active", "currentIndex": idx, "timeLimit": limit,
		},
	})
}
