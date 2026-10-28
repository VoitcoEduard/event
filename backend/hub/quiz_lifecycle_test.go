package hub

import (
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func presentingHub(t *testing.T) (*QuizHub, *Client) {
	t.Helper()
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.CreateSession("m", "e")
	h.mu.Lock()
	s := h.sessions["m"]
	s.RunID, s.State = "run-1", "question-presenting"
	s.Questions = []models.QuizQuestion{{TimeLimit: 30, ExplainTime: 3600}, {TimeLimit: 20, ExplainTime: 3600}}
	s.QuestionWindows = []models.QuestionWindow{{QuestionIndex: 0}}
	h.scheduleQuizTransitionLocked(s, time.Hour)
	h.mu.Unlock()
	return h, &Client{IsHost: true, ModuleID: "m"}
}

func TestQuizTransitionRejectsAbandonedCallbacks(t *testing.T) {
	for _, action := range []string{"reset", "restart", "next", "results", "leaderboard", "end failure", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			h, host := presentingHub(t)
			old := h.quizTimers["m"]
			switch action {
			case "reset":
				h.ResetSession("m")
			case "restart":
				h.mu.Lock()
				s := h.sessions["m"]
				s.RunID = "run-2"
				h.scheduleQuizTransitionLocked(s, time.Hour)
				h.mu.Unlock()
			case "end failure":
				// No result store: ending fails and must restore the pending transition.
				h.processHostEndQuiz(host, ClientMessage{})
			case "shutdown":
				h.Shutdown()
			default:
				h.mu.Lock()
				switch action {
				case "next":
					h.processHostNextQuestion(host, ClientMessage{})
				case "results":
					h.processHostShowResults(host, ClientMessage{})
				case "leaderboard":
					h.processHostShowLeaderboard(host, ClientMessage{})
				}
				h.mu.Unlock()
			}
			before := h.GetSession("m")
			// Simulate an expired callback that was queued before Timer.Stop.
			h.activateQuizQuestion("m", old)
			after := h.GetSession("m")
			if after.State != before.State || after.CurrentIndex != before.CurrentIndex || after.StartTime != before.StartTime {
				t.Fatalf("abandoned callback changed session: before=%+v after=%+v", before, after)
			}
			h.mu.RLock()
			pending := h.quizTimers["m"]
			h.mu.RUnlock()
			if action == "restart" || action == "next" || action == "end failure" {
				if pending == nil || pending == old {
					t.Fatal("replacement transition missing")
				}
				h.activateQuizQuestion("m", pending)
				if got := h.GetSession("m").State; got != "question-active" {
					t.Fatalf("replacement transition failed: %s", got)
				}
			} else if pending != nil {
				t.Fatal("abandoned transition was not cancelled")
			}
		})
	}
}

func TestQuizTransitionFiresAndUpdatesWindow(t *testing.T) {
	h, _ := presentingHub(t)
	client := &Client{ModuleID: "m", send: make(chan []byte, 1)}
	h.Register(client)
	h.mu.Lock()
	h.scheduleQuizTransitionLocked(h.sessions["m"], time.Millisecond)
	h.mu.Unlock()
	select {
	case <-client.send:
	case <-time.After(3 * time.Second):
		t.Fatal("presentation timer did not fire")
	}
	s := h.GetSession("m")
	w := s.QuestionWindows[0]
	if s.State != "question-active" || w.StartTime != s.StartTime || w.EndTime-w.StartTime != 30 || w.GraceEndTime-w.EndTime != GracePeriodSeconds {
		t.Fatalf("timer did not initialize active question window: %+v", s)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.quizTimers) != 0 {
		t.Fatal("completed timer retained in hub")
	}
}

func TestShutdownRejectsRegistrationsAndCancelsTimers(t *testing.T) {
	h, _ := presentingHub(t)
	h.simpleQuizSessions["m"] = &models.SimpleQuizSession{ModuleID: "m", EventID: "e"}
	h.scheduleSimpleQuizAutoLock("m", time.Now().Add(time.Hour).Unix())
	h.Shutdown()
	if h.Register(&Client{send: make(chan []byte, 1)}) {
		t.Fatal("registration accepted after shutdown")
	}
	h.scheduleSimpleQuizAutoLock("m", time.Now().Add(time.Hour).Unix())
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.quizTimers) != 0 || len(h.simpleQuizTimers) != 0 || len(h.clients) != 0 {
		t.Fatal("shutdown retained timers or accepted new clients")
	}
}

func TestForgetEventSessionsClearsModuleDataAndWaitsForWriters(t *testing.T) {
	h, _ := presentingHub(t)
	h.CreateSession("other", "other-event")
	h.simpleQuizSessions["m"] = &models.SimpleQuizSession{ModuleID: "m", EventID: "e"}
	h.simpleQuizUndo["m"] = map[string][]simpleQuizUndoEntry{"judge": {{AnswerID: "answer"}}}
	h.scheduleSimpleQuizAutoLock("m", time.Now().Add(time.Hour).Unix())
	old := h.quizTimers["m"]
	lock := h.simpleLock("m")
	lock.Lock()
	done := make(chan struct{})
	go func() {
		h.ForgetEventSessions("e", []string{"m"})
		close(done)
	}()
	// Model an operation publishing its last snapshot before releasing the module lock.
	h.mu.Lock()
	h.simpleQuizSessions["m"] = &models.SimpleQuizSession{ModuleID: "m", EventID: "e", Status: "lobby"}
	h.mu.Unlock()
	lock.Unlock()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("session eviction deadlocked")
	}
	h.activateQuizQuestion("m", old)
	// A handler that finished opening a question before deletion may try to arm
	// its timer late; it must not recreate runtime entries for a deleted module.
	h.scheduleSimpleQuizAutoLock("m", time.Now().Add(time.Hour).Unix())
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.sessions["m"] != nil || h.simpleQuizSessions["m"] != nil || h.simpleQuizUndo["m"] != nil || h.quizTimers["m"] != nil || h.simpleQuizTimers["m"] != nil {
		t.Fatal("deleted module retained session data or timers")
	}
	if h.sessions["other"] == nil {
		t.Fatal("deleting an event removed another event's session")
	}
}
