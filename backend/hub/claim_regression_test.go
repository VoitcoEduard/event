package hub

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestSimpleJoinMembershipSynchronized(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	c := &Client{send: make(chan []byte, 1)}
	h.Register(c)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			h.processSimplePlayerJoin(c, ClientMessage{Payload: map[string]interface{}{"moduleId": "m", "playerId": "p"}})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			h.BroadcastToSession("other", ServerMessage{Type: "ping"})
		}
	}()
	wg.Wait()
}

func TestProjectorReceivesCurrentRevealOnReconnect(t *testing.T) {
	for _, state := range []string{"results", "leaderboard", "finished"} {
		h := NewQuizHub(Dependencies{})
		h.CreateSession("m", "e")
		h.UpdateSession("m", func(s *models.QuizSession) {
			s.State = state
			s.Questions = []models.QuizQuestion{{ID: "q", Type: models.QuestionTypeClassic, Answers: []string{"yes", "no"}, CorrectIndex: 0}, {ID: "future", Type: models.QuestionTypeClassic, Answers: []string{"hidden"}}}
			s.Batches = CalculateBatches(s.Questions)
		})
		c := &Client{send: make(chan []byte, 10)}
		h.Register(c)
		h.ProcessMessage(c, []byte(`{"type":"projector:join","payload":{"moduleId":"m","eventId":"e"}}`))
		var types []string
		for len(c.send) > 0 {
			var msg ServerMessage
			if err := json.Unmarshal(<-c.send, &msg); err != nil {
				t.Fatal(err)
			}
			types = append(types, msg.Type)
		}
		if len(types) < 4 || types[len(types)-2] != "data:batch" || types[len(types)-1] != "data:reveal" {
			t.Fatalf("%s messages: %v", state, types)
		}
	}
}

func TestSimpleHostJoinRequiresAdmin(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	c := &Client{send: make(chan []byte, 2)}
	h.Register(c)
	h.ProcessMessage(c, []byte(`{"type":"simple:host_join","payload":{"moduleId":"m"}}`))
	if c.IsHost || c.ModuleID != "" {
		t.Fatal("unauthenticated host join accepted")
	}
}

func TestReconnectDoesNotDuplicateHistory(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.CreateSession("m", "e")
	h.mu.Lock()
	s := h.sessions["m"]
	s.CurrentIndex = 3
	s.Players["p"] = &models.Player{ID: "p", Nickname: "P"}
	s.PlayerSignals = []models.PlayerSignal{{PlayerID: "p", QuestionIndex: 1, AnswerIndex: 2}}
	h.mu.Unlock()
	c := &Client{ModuleID: "m", PlayerID: "p", quizBinding: &quizPlayerBinding{eventID: "e", moduleID: "m", playerID: "p"}, send: make(chan []byte, 5)}
	msg := []byte(`{"type":"player:reconnect","payload":{"moduleId":"m","lastKnownQuestionIndex":0}}`)
	h.ProcessMessage(c, msg)
	h.ProcessMessage(c, msg)
	got := h.GetSession("m")
	if len(got.PlayerSignals) != 2 || got.PlayerSignals[0].AnswerIndex != 2 {
		t.Fatalf("history: %+v", got.PlayerSignals)
	}
}

func TestChangingRoleRevokesQuizPlayerBinding(t *testing.T) {
	for _, join := range []string{
		`{"type":"projector:join","payload":{"moduleId":"m","eventId":"e"}}`,
		`{"type":"participant:join","payload":{"eventId":"e","userId":"p"}}`,
		`{"type":"simple:player_join","payload":{"moduleId":"m","playerId":"p"}}`,
	} {
		h := NewQuizHub(Dependencies{})
		h.CreateSession("m", "e")
		h.mu.Lock()
		h.sessions["m"].CurrentIndex = 2
		h.sessions["m"].Players["p"] = &models.Player{ID: "p"}
		h.mu.Unlock()
		c := &Client{ModuleID: "m", PlayerID: "p", quizBinding: &quizPlayerBinding{eventID: "e", moduleID: "m", playerID: "p"}, send: make(chan []byte, 20)}
		h.Register(c)
		h.ProcessMessage(c, []byte(join))
		h.ProcessMessage(c, []byte(`{"type":"player:reconnect","payload":{"moduleId":"m","lastKnownQuestionIndex":0}}`))
		if len(h.GetSession("m").PlayerSignals) != 0 {
			t.Fatalf("%s allowed reconnect to change history", join)
		}
	}
}
