package hub

import (
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
	c := &Client{ModuleID: "m", PlayerID: "p", send: make(chan []byte, 5)}
	msg := []byte(`{"type":"player:reconnect","payload":{"moduleId":"m","lastKnownQuestionIndex":0}}`)
	h.ProcessMessage(c, msg)
	h.ProcessMessage(c, msg)
	got := h.GetSession("m")
	if len(got.PlayerSignals) != 2 || got.PlayerSignals[0].AnswerIndex != 2 {
		t.Fatalf("history: %+v", got.PlayerSignals)
	}
}
