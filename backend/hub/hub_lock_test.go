package hub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func assertHubUnlocked(t *testing.T, h *QuizHub) {
	t.Helper()
	if !h.mu.TryLock() {
		t.Fatal("database operation ran while the hub lock was held")
	}
	h.mu.Unlock()
}

func TestHostStartQuizLoadsOutsideLock(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.CreateSession("module-1", "")
	h.eventByModule = func(_ context.Context, moduleID string) (*models.Event, error) {
		assertHubUnlocked(t, h)
		return &models.Event{
			ID: "event-1",
			Modules: []models.Module{{ID: moduleID, Data: map[string]any{
				"questions": []models.QuizQuestion{{ID: "question-1", TimeLimit: 30, ExplainTime: 60}},
			}}},
		}, nil
	}
	h.processHostStartQuiz(&Client{ModuleID: "module-1", IsHost: true}, ClientMessage{})

	h.mu.RLock()
	defer h.mu.RUnlock()
	session := h.sessions["module-1"]
	if session.EventID != "event-1" || session.State != "question-presenting" || len(session.Questions) != 1 {
		t.Fatalf("quiz did not start with loaded questions: %+v", session)
	}
}

func TestHostStartQuizRejectsMissingQuestions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event *models.Event
		err   error
	}{
		{name: "lookup error", err: errors.New("database unavailable")},
		{name: "empty module", event: &models.Event{Modules: []models.Module{{ID: "module-1"}}}},
		{name: "missing module", event: &models.Event{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := NewQuizHub(Dependencies{})
			h.CreateSession("module-1", "")
			h.eventByModule = func(context.Context, string) (*models.Event, error) {
				return tc.event, tc.err
			}
			h.processHostStartQuiz(&Client{ModuleID: "module-1", IsHost: true}, ClientMessage{})
			h.mu.RLock()
			defer h.mu.RUnlock()
			if h.sessions["module-1"].State != "lobby" {
				t.Fatal("failed lookup changed quiz state")
			}
		})
	}
}

func TestHostEndTeamsSavesAndBroadcastsWithoutDeadlock(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.saveTeams = func(eventID, moduleID, moduleName string) (*models.TeamResult, error) {
		assertHubUnlocked(t, h)
		return &models.TeamResult{EventID: eventID, ModuleID: moduleID}, nil
	}
	admin := &Client{EventID: "event-1", IsTeamsAdmin: true, send: make(chan []byte, 1)}
	h.Register(admin)
	h.processHostEndTeams(&Client{IsHost: true}, ClientMessage{Payload: map[string]any{
		"eventId": "event-1", "moduleId": "module-1", "moduleName": "Teams",
	}})
	assertMessageType(t, admin.send, "teams:ended")
}

func TestParticipantJoinLoadsTeamsAndBroadcastsWithoutDeadlock(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.teamsByEvent = func(_ context.Context, eventID string) ([]models.Team, error) {
		assertHubUnlocked(t, h)
		return []models.Team{{ID: "team-1"}}, nil
	}
	admin := &Client{EventID: "event-1", IsTeamsAdmin: true, send: make(chan []byte, 1)}
	h.Register(admin)
	participant := &Client{send: make(chan []byte, 1)}
	h.ProcessMessage(participant, []byte(`{"type":"participant:join","payload":{"eventId":"event-1","userId":"user-1","displayName":"Alex"}}`))
	assertMessageType(t, participant.send, "teams:available")
	assertMessageType(t, admin.send, "participant:joined")
}

// A panic inside a message handler must not crash the server (readPump runs
// ProcessMessage in a bare goroutine) and must not leave the hub lock held.
func TestProcessMessageRecoversFromHandlerPanic(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.mu.Lock()
	// Partially-initialized session: Players is nil, so player:join panics on
	// map assignment — the class of runtime bug recovery must contain.
	h.sessions["module-1"] = &models.QuizSession{ModuleID: "module-1", State: "lobby"}
	h.mu.Unlock()

	player := &Client{send: make(chan []byte, 1)}
	h.ProcessMessage(player, []byte(`{"type":"player:join","payload":{"moduleId":"module-1","userId":"user-1","nickname":"Alex"}}`))

	assertHubUnlocked(t, h)
}

// A join greeting to a client whose send buffer is full must not block while
// holding the hub lock: the stalled client is evicted and the hub stays usable.
func TestJoinToStalledClientDoesNotHoldHubLock(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	victim := &Client{send: make(chan []byte, 1)}
	victim.send <- []byte(`full`)
	h.Register(victim)
	healthy := &Client{ModuleID: "module-1", send: make(chan []byte, 4)}
	h.Register(healthy)

	done := make(chan struct{})
	go func() {
		// Mirrors ProcessMessage, which holds h.mu across the join handlers.
		h.mu.Lock()
		defer h.mu.Unlock()
		defer close(done)
		h.processHostJoin(victim, ClientMessage{Payload: map[string]any{"moduleId": "module-1"}})
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("hub stalled: join greeting to a client with a full buffer blocked while holding h.mu")
	}
	assertHubUnlocked(t, h)

	h.mu.RLock()
	_, victimKept := h.clients[victim]
	_, healthyKept := h.clients[healthy]
	h.mu.RUnlock()
	if victimKept {
		t.Fatal("stalled client was not evicted from the hub")
	}
	if !healthyKept {
		t.Fatal("healthy client was dropped from the hub")
	}

	h.BroadcastToSession("module-1", ServerMessage{Type: "ping"})
	assertMessageType(t, healthy.send, "ping")
}

// A simple-quiz join greeting to a stalled client must not block readPump.
// The stalled client is evicted and the hub stays usable.
func TestSimpleJoinToStalledClientDoesNotBlock(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.simpleQuizSessions["module-1"] = &models.SimpleQuizSession{
		ModuleID: "module-1",
		Status:   "lobby",
		Players:  make(map[string]*models.SimpleQuizPlayer),
	}
	victim := &Client{send: make(chan []byte, 1)}
	victim.send <- []byte(`full`)
	h.Register(victim)
	healthy := &Client{ModuleID: "module-1", send: make(chan []byte, 4)}
	h.Register(healthy)

	done := make(chan struct{})
	go func() {
		defer close(done)
		// Called from readPump without holding h.mu.
		h.processSimpleHostJoin(victim, ClientMessage{Payload: map[string]any{"moduleId": "module-1"}})
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("simple join greeting to a stalled client blocked readPump")
	}
	assertHubUnlocked(t, h)

	h.mu.RLock()
	_, victimKept := h.clients[victim]
	_, healthyKept := h.clients[healthy]
	h.mu.RUnlock()
	if victimKept {
		t.Fatal("stalled simple client was not evicted from the hub")
	}
	if !healthyKept {
		t.Fatal("healthy simple client was dropped from the hub")
	}
}

// A simple-quiz join greeting must not panic when cleanup concurrently closes
// the client's send channel.
func TestSimpleJoinDoesNotRaceWithConcurrentUnregister(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.simpleQuizSessions["module-1"] = &models.SimpleQuizSession{
		ModuleID: "module-1",
		Status:   "lobby",
		Players:  make(map[string]*models.SimpleQuizPlayer),
	}
	client := &Client{send: make(chan []byte, 1)}
	client.send <- []byte(`full`)
	h.Register(client)

	go func() {
		h.Unregister(client)
	}()

	// This used to panic with "send on closed channel" when the blocking send
	// in the simple join handler raced with Unregister or broadcast eviction.
	h.processSimpleHostJoin(client, ClientMessage{Payload: map[string]any{"moduleId": "module-1"}})
}

func assertMessageType(t *testing.T, messages <-chan []byte, want string) {
	t.Helper()
	select {
	case raw := <-messages:
		var message ServerMessage
		if err := json.Unmarshal(raw, &message); err != nil {
			t.Fatal(err)
		}
		if message.Type != want {
			t.Fatalf("message type = %q, want %q", message.Type, want)
		}
	default:
		t.Fatalf("no %s message sent", want)
	}
}
