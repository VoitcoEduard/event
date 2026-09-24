package hub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
	h := NewQuizHub()
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
			h := NewQuizHub()
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
	h := NewQuizHub()
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
	h := NewQuizHub()
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
