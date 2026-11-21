package hub

import (
	"encoding/json"
	"fmt"
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

func TestResetSessionClearsRunStateAndUnbindsPlayers(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.CreateSession("m", "e")
	h.CreateSession("other", "e2")

	h.mu.Lock()
	s := h.sessions["m"]
	s.RunID = "run-1"
	s.State = "question-active"
	s.CurrentIndex = 1
	s.Players["p1"] = &models.Player{ID: "p1", Nickname: "Alice", Score: 500}
	s.Streaks = map[string]int{"p1": 2}
	s.WrongPictureAspects = map[int]float64{0: 1.5}
	s.Questions = []models.QuizQuestion{{ID: "q0", Type: "classic", TimeLimit: 20, CorrectIndex: 1, Answers: []string{"no", "yes"}}}
	s.QuestionWindows = []models.QuestionWindow{{QuestionIndex: 0, StartTime: time.Now().Unix(), EndTime: time.Now().Unix() + 20, GraceEndTime: time.Now().Unix() + 22}}
	h.sessions["other"].Players["p2"] = &models.Player{ID: "p2", Nickname: "Bob", Score: 100}
	h.mu.Unlock()

	host := &Client{ModuleID: "m", IsHost: true, send: make(chan []byte, 8)}
	projector := &Client{ModuleID: "m", IsProjector: true, send: make(chan []byte, 8)}
	player := &Client{
		ModuleID:    "m",
		PlayerID:    "p1",
		quizBinding: &quizPlayerBinding{eventID: "e", moduleID: "m", playerID: "p1"},
		send:        make(chan []byte, 8),
	}
	otherPlayer := &Client{
		ModuleID:    "other",
		PlayerID:    "p2",
		quizBinding: &quizPlayerBinding{eventID: "e2", moduleID: "other", playerID: "p2"},
		send:        make(chan []byte, 8),
	}
	for _, c := range []*Client{host, projector, player, otherPlayer} {
		h.Register(c)
	}

	if !h.ResetSession("m") {
		t.Fatal("ResetSession returned false")
	}

	// Player received the reset cmd:state before unbinding.
	select {
	case raw := <-player.send:
		var msg struct {
			Type    string `json:"type"`
			Payload struct {
				State string `json:"state"`
				RunID string `json:"runId"`
				Reset bool   `json:"reset"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type != "cmd:state" || msg.Payload.State != "lobby" || msg.Payload.RunID != "" || !msg.Payload.Reset {
			t.Fatalf("unexpected reset message: %s", raw)
		}
	default:
		t.Fatal("player did not receive cmd:state reset notification")
	}

	h.mu.RLock()
	if s.RunID != "" || s.State != "lobby" || len(s.Players) != 0 || len(s.Streaks) != 0 || len(s.WrongPictureAspects) != 0 {
		h.mu.RUnlock()
		t.Fatalf("session state not cleared on reset: %+v", s)
	}
	if player.ModuleID != "" || player.PlayerID != "" || player.quizBinding != nil {
		h.mu.RUnlock()
		t.Fatalf("player remained bound after reset: module=%q player=%q binding=%+v", player.ModuleID, player.PlayerID, player.quizBinding)
	}
	if host.ModuleID != "m" || !host.IsHost || projector.ModuleID != "m" || !projector.IsProjector {
		h.mu.RUnlock()
		t.Fatal("host or projector was unbound by reset")
	}
	if otherPlayer.ModuleID != "other" || otherPlayer.PlayerID != "p2" || otherPlayer.quizBinding == nil {
		h.mu.RUnlock()
		t.Fatal("other module's player was affected by reset")
	}
	h.mu.RUnlock()

	// Start a new run and send a signal from the stale socket; it must be ignored.
	h.mu.Lock()
	s.RunID = "run-2"
	s.State = "question-active"
	s.CurrentIndex = 0
	s.Questions = []models.QuizQuestion{{ID: "q0", Type: "classic", TimeLimit: 20, CorrectIndex: 1, Answers: []string{"no", "yes"}}}
	s.QuestionWindows = []models.QuestionWindow{{QuestionIndex: 0, StartTime: time.Now().Unix(), EndTime: time.Now().Unix() + 20, GraceEndTime: time.Now().Unix() + 22}}
	h.mu.Unlock()

	h.ProcessMessage(player, []byte(fmt.Sprintf(`{"type":"player:signal","payload":{"questionIndex":0,"answerIndex":1,"submissionId":"00000000-0000-4000-8000-000000000001","runId":%q}}`, "run-2")))

	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(s.PlayerSignals) != 0 || len(s.AnswerCounts) != 0 || len(s.AnsweredPlayerIDs) != 0 {
		t.Fatalf("unbound player polluted next run: signals=%d counts=%v answered=%v", len(s.PlayerSignals), s.AnswerCounts, s.AnsweredPlayerIDs)
	}
}

func TestProcessPlayerSignalRejectsMissingSessionPlayer(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.CreateSession("m", "e")

	h.mu.Lock()
	s := h.sessions["m"]
	s.RunID = "run-2"
	s.State = "question-active"
	s.CurrentIndex = 0
	s.Questions = []models.QuizQuestion{{ID: "q0", Type: "classic", TimeLimit: 20, CorrectIndex: 1, Answers: []string{"no", "yes"}}}
	s.QuestionWindows = []models.QuestionWindow{{QuestionIndex: 0, StartTime: time.Now().Unix(), EndTime: time.Now().Unix() + 20, GraceEndTime: time.Now().Unix() + 22}}
	h.mu.Unlock()

	// Simulate a bound client missing from session.Players.
	c := &Client{
		ModuleID:    "m",
		PlayerID:    "ghost",
		quizBinding: &quizPlayerBinding{eventID: "e", moduleID: "m", playerID: "ghost"},
		send:        make(chan []byte, 4),
	}
	h.Register(c)

	h.ProcessMessage(c, []byte(`{"type":"player:signal","payload":{"questionIndex":0,"answerIndex":1,"submissionId":"00000000-0000-4000-8000-000000000001","runId":"run-2"}}`))

	select {
	case raw := <-c.send:
		var msg struct {
			Type    string `json:"type"`
			Payload struct {
				Accepted bool   `json:"accepted"`
				Reason   string `json:"reason"`
				Awarded  int    `json:"awarded"`
				Total    int    `json:"total"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type != "answer:ack" || msg.Payload.Accepted || msg.Payload.Reason != "no_player" || msg.Payload.Awarded != 0 || msg.Payload.Total != 0 {
			t.Fatalf("unexpected ack for missing player: %s", raw)
		}
	default:
		t.Fatal("missing ack for rejected missing-player signal")
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(s.PlayerSignals) != 0 || len(s.AnswerCounts) != 0 || len(s.AnsweredPlayerIDs) != 0 || len(s.Streaks) != 0 {
		t.Fatalf("missing player signal mutated session: signals=%d counts=%v answered=%v streaks=%v", len(s.PlayerSignals), s.AnswerCounts, s.AnsweredPlayerIDs, s.Streaks)
	}
}

func TestResetAndRejoinScoresInNextRun(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.CreateSession("m", "e")

	c := &Client{send: make(chan []byte, 16)}
	h.Register(c)
	h.ProcessMessage(c, []byte(`{"type":"player:join","payload":{"moduleId":"m","userId":"p1","nickname":"Alice"}}`))

	if !h.ResetSession("m") {
		t.Fatal("ResetSession returned false")
	}
	for len(c.send) > 0 {
		<-c.send
	}

	// Re-join after reset, then start run 2 and answer.
	h.ProcessMessage(c, []byte(`{"type":"player:join","payload":{"moduleId":"m","userId":"p1","nickname":"Alice"}}`))
	h.mu.Lock()
	c.quizBinding = &quizPlayerBinding{eventID: "e", moduleID: "m", playerID: "p1"}
	s := h.sessions["m"]
	s.RunID = "run-2"
	s.State = "question-active"
	s.CurrentIndex = 0
	s.Questions = []models.QuizQuestion{{ID: "q0", Type: "classic", TimeLimit: 20, CorrectIndex: 1, Answers: []string{"no", "yes"}}}
	s.QuestionWindows = []models.QuestionWindow{{QuestionIndex: 0, StartTime: time.Now().Unix(), EndTime: time.Now().Unix() + 20, GraceEndTime: time.Now().Unix() + 22}}
	h.mu.Unlock()
	for len(c.send) > 0 {
		<-c.send
	}

	h.ProcessMessage(c, []byte(`{"type":"player:signal","payload":{"questionIndex":0,"answerIndex":1,"submissionId":"00000000-0000-4000-8000-000000000002","runId":"run-2"}}`))

	var acked bool
	for len(c.send) > 0 {
		raw := <-c.send
		var msg struct {
			Type    string `json:"type"`
			Payload struct {
				Accepted bool `json:"accepted"`
				Awarded  int  `json:"awarded"`
				Total    int  `json:"total"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type == "answer:ack" {
			acked = true
			if !msg.Payload.Accepted || msg.Payload.Awarded <= 0 || msg.Payload.Total != msg.Payload.Awarded {
				t.Fatalf("unexpected ack after rejoin: %s", raw)
			}
		}
	}
	if !acked {
		t.Fatal("did not receive answer:ack")
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	if s.Players["p1"] == nil || s.Players["p1"].Score <= 0 {
		t.Fatalf("rejoined player score not recorded: %+v", s.Players["p1"])
	}
}
