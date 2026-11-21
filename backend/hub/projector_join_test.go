package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestProjectorJoinDoesNotAllocatePhantomSessions(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)

	c := &Client{
		IsHost:       true,
		IsTeamsAdmin: true,
		PlayerID:     "stale-player",
		send:         make(chan []byte, 64),
	}
	h.Register(c)

	for i := 0; i < 50; i++ {
		mod := fmt.Sprintf("phantom-%d", i)
		h.ProcessMessage(c, []byte(fmt.Sprintf(`{"type":"projector:join","payload":{"moduleId":%q,"eventId":"e1"}}`, mod)))
		if h.GetSession(mod) != nil {
			t.Fatalf("projector:join allocated phantom session for %s", mod)
		}
	}

	h.mu.RLock()
	sessionCount := len(h.sessions)
	isHost, isTeamsAdmin, isProjector, playerID, moduleID := c.IsHost, c.IsTeamsAdmin, c.IsProjector, c.PlayerID, c.ModuleID
	h.mu.RUnlock()

	if sessionCount != 0 {
		t.Fatalf("len(h.sessions) = %d, want 0", sessionCount)
	}
	if isHost || isTeamsAdmin || !isProjector || playerID != "" || moduleID != "phantom-49" {
		t.Fatalf("unexpected client role after projector:join: host=%v teamsAdmin=%v projector=%v playerID=%q moduleID=%q",
			isHost, isTeamsAdmin, isProjector, playerID, moduleID)
	}

	// Verify initial cmd:state lobby messages were delivered.
	if len(c.send) != 50 {
		t.Fatalf("queued messages = %d, want 50", len(c.send))
	}
	raw := <-c.send
	var msg struct {
		Type    string `json:"type"`
		Payload struct {
			State        string `json:"state"`
			CurrentIndex int    `json:"currentIndex"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal cmd:state: %v", err)
	}
	if msg.Type != "cmd:state" || msg.Payload.State != "lobby" || msg.Payload.CurrentIndex != 0 {
		t.Fatalf("unexpected initial message: %s", raw)
	}
}

func TestPreconnectedProjectorReceivesLiveUpdatesWhenHostStartsQuiz(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.eventByModule = func(_ context.Context, moduleID string) (*models.Event, error) {
		return &models.Event{
			ID: "e1",
			Modules: []models.Module{{
				ID:   moduleID,
				Type: models.ModuleTypeQuiz,
				Data: map[string]any{
					"questions": []models.QuizQuestion{{
						ID: "q0", Type: models.QuestionTypeClassic, TimeLimit: 20, ExplainTime: 60,
						CorrectIndex: 1, Answers: []string{"No", "Yes"},
					}},
				},
			}},
		}, nil
	}

	// Projector connects BEFORE any session exists.
	projector := &Client{send: make(chan []byte, 16)}
	h.Register(projector)
	h.ProcessMessage(projector, []byte(`{"type":"projector:join","payload":{"moduleId":"mod-live","eventId":"e1"}}`))

	if h.GetSession("mod-live") != nil {
		t.Fatal("projector:join created a session before host started")
	}
	// Drain initial lobby cmd:state.
	for len(projector.send) > 0 {
		<-projector.send
	}

	// Host now creates the session and starts the quiz.
	h.CreateSession("mod-live", "e1")
	host := &Client{ModuleID: "mod-live", IsHost: true, send: make(chan []byte, 16)}
	h.Register(host)
	h.processHostStartQuiz(host, ClientMessage{})

	gotTypes := map[string]bool{}
	var gotState string
	for len(projector.send) > 0 {
		raw := <-projector.send
		var msg struct {
			Type    string         `json:"type"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal projector message: %v", err)
		}
		gotTypes[msg.Type] = true
		if msg.Type == "cmd:state" {
			gotState, _ = msg.Payload["state"].(string)
		}
	}
	if !gotTypes["data:batch"] || !gotTypes["cmd:state"] || !gotTypes["state:update"] {
		t.Fatalf("preconnected projector missed live start broadcasts: %v", gotTypes)
	}
	if gotState != "question-presenting" {
		t.Fatalf("projector cmd:state = %q, want question-presenting", gotState)
	}
}
