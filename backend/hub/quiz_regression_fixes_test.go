package hub

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

func readClientMessage(t *testing.T, c *Client) map[string]any {
	t.Helper()
	select {
	case raw := <-c.send:
		var msg map[string]any
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unreadable message: %v", err)
		}
		return msg
	default:
		t.Fatal("expected a message from the hub, got none")
		return nil
	}
}

func payloadOf(t *testing.T, msg map[string]any) map[string]any {
	t.Helper()
	payload, _ := msg["payload"].(map[string]any)
	if payload == nil {
		t.Fatalf("message %+v has no object payload", msg)
	}
	return payload
}

// H3: a signal that arrives while results are being saved must be rejected with
// an explicit answer:ack instead of leaving the player stuck waiting.
func TestPlayerSignalDuringFinishingIsAcked(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.CreateSession("m", "e")
	runID, submissionID := uuid.NewString(), uuid.NewString()
	h.mu.Lock()
	s := h.sessions["m"]
	s.RunID = runID
	s.State = "finishing"
	s.Players["p1"] = &models.Player{ID: "p1", Nickname: "P"}
	h.mu.Unlock()

	c := &Client{
		ModuleID: "m", PlayerID: "p1",
		quizBinding: &quizPlayerBinding{eventID: "e", moduleID: "m", playerID: "p1"},
		send:        make(chan []byte, 4),
	}
	h.Register(c)
	h.ProcessMessage(c, []byte(`{"type":"player:signal","payload":{"questionIndex":0,"answerIndex":1,"submissionId":"`+submissionID+`","runId":"`+runID+`"}}`))

	msg := readClientMessage(t, c)
	if msg["type"] != "answer:ack" {
		t.Fatalf("message type = %v, want answer:ack", msg["type"])
	}
	payload := payloadOf(t, msg)
	if payload["accepted"] != false || payload["reason"] != "quiz_finishing" {
		t.Fatalf("unexpected ack payload: %+v", payload)
	}
	if payload["submissionId"] != submissionID || payload["runId"] != runID || payload["questionIndex"] != float64(0) {
		t.Fatalf("ack does not echo submission identity: %+v", payload)
	}
	if len(h.sessions["m"].PlayerSignals) != 0 {
		t.Fatal("finishing-state signal was recorded")
	}
}

// M1: while results are being saved, a host:start_quiz attempt must surface a
// cmd:error instead of silently doing nothing.
func TestHostStartQuizDuringFinishingSurfacesError(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.CreateSession("m", "e")
	h.mu.Lock()
	h.sessions["m"].State = "finishing"
	h.mu.Unlock()
	h.eventByModule = func(ctx context.Context, moduleID string) (*models.Event, error) {
		return &models.Event{ID: "e", Modules: []models.Module{{
			ID: moduleID, Type: models.ModuleTypeQuiz,
			Data: map[string]any{"questions": []any{map[string]any{
				"id": "q0", "type": "classic", "text": "Q", "answers": []any{"a", "b"},
				"correctIndex": 0, "timeLimit": 20,
			}}},
		}}}, nil
	}
	c := &Client{IsHost: true, ModuleID: "m", send: make(chan []byte, 4)}
	h.Register(c)

	h.processHostStartQuiz(c, ClientMessage{})

	msg := readClientMessage(t, c)
	if msg["type"] != "cmd:error" {
		t.Fatalf("message type = %v, want cmd:error", msg["type"])
	}
	if payload := payloadOf(t, msg); payload["code"] != "quiz_finishing" {
		t.Fatalf("cmd:error code = %v, want quiz_finishing", payload["code"])
	}
}

// M1: lookup failures must reach the host as cmd:error start_failed.
func TestHostStartQuizLookupFailureSurfacesError(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	t.Cleanup(h.Shutdown)
	h.CreateSession("m", "e")
	h.eventByModule = func(ctx context.Context, moduleID string) (*models.Event, error) {
		return nil, errors.New("store unreachable")
	}
	c := &Client{IsHost: true, ModuleID: "m", send: make(chan []byte, 4)}
	h.Register(c)

	h.processHostStartQuiz(c, ClientMessage{})

	msg := readClientMessage(t, c)
	if msg["type"] != "cmd:error" {
		t.Fatalf("message type = %v, want cmd:error", msg["type"])
	}
	if payload := payloadOf(t, msg); payload["code"] != "start_failed" {
		t.Fatalf("cmd:error code = %v, want start_failed", payload["code"])
	}
}

type stubPlayerIdentity struct {
	event      *models.Event
	eventErr   error
	proofHas   bool
	proofValid bool
	proofErr   error
}

func (s *stubPlayerIdentity) GetEventByModule(ctx context.Context, id string) (*models.Event, error) {
	return s.event, s.eventErr
}
func (s *stubPlayerIdentity) CheckParticipantProof(ctx context.Context, eventID, userID, secret string) (bool, bool, error) {
	return s.proofHas, s.proofValid, s.proofErr
}
func (s *stubPlayerIdentity) EnsureParticipant(ctx context.Context, p *models.Participant) (*models.Participant, error) {
	return p, nil
}

func joinPayload(moduleID string) []byte {
	return []byte(`{"type":"player:join","payload":{"moduleId":"` + moduleID + `","userId":"p1","participantSecret":"sec","nickname":"Nick"}}`)
}

func activeEvent(moduleType models.ModuleType) *models.Event {
	return &models.Event{ID: "e", IsActive: true, Modules: []models.Module{{
		ID: "m", Type: moduleType,
	}}}
}

// M2: transient DB errors during player:join must surface as join_failed.
func TestPlayerJoinSendErrorOnLookupFailure(t *testing.T) {
	h := NewQuizHub(Dependencies{PlayerIdentity: &stubPlayerIdentity{eventErr: errors.New("db down")}})
	t.Cleanup(h.Shutdown)
	c := &Client{send: make(chan []byte, 4)}
	h.Register(c)
	h.ProcessMessage(c, joinPayload("m"))
	msg := readClientMessage(t, c)
	if msg["type"] != "cmd:error" {
		t.Fatalf("message type = %v, want cmd:error", msg["type"])
	}
	payload := payloadOf(t, msg)
	if payload["code"] != "join_failed" {
		t.Fatalf("cmd:error code = %v, want join_failed", payload["code"])
	}
}

// M2: a missing module must not be reported as a transient failure.
func TestPlayerJoinSendJoinClosedOnMissingModule(t *testing.T) {
	h := NewQuizHub(Dependencies{PlayerIdentity: &stubPlayerIdentity{eventErr: postgres.ErrNotFound}})
	t.Cleanup(h.Shutdown)
	c := &Client{send: make(chan []byte, 4)}
	h.Register(c)
	h.ProcessMessage(c, joinPayload("m"))
	msg := readClientMessage(t, c)
	if msg["type"] != "cmd:error" {
		t.Fatalf("message type = %v, want cmd:error", msg["type"])
	}
	payload := payloadOf(t, msg)
	if payload["code"] != "join_closed" {
		t.Fatalf("cmd:error code = %v, want join_closed", payload["code"])
	}
}

// M2: identity-verification DB errors must surface as join_failed.
func TestPlayerJoinSendErrorOnProofFailure(t *testing.T) {
	h := NewQuizHub(Dependencies{PlayerIdentity: &stubPlayerIdentity{
		event: activeEvent(models.ModuleTypeQuiz), proofErr: errors.New("db down"),
	}})
	t.Cleanup(h.Shutdown)
	c := &Client{send: make(chan []byte, 4)}
	h.Register(c)
	h.ProcessMessage(c, joinPayload("m"))
	msg := readClientMessage(t, c)
	if msg["type"] != "cmd:error" {
		t.Fatalf("message type = %v, want cmd:error", msg["type"])
	}
	payload := payloadOf(t, msg)
	if payload["code"] != "join_failed" {
		t.Fatalf("cmd:error code = %v, want join_failed", payload["code"])
	}
}

// M2 bonus: wrong_pictures modules join through the same live-quiz path.
func TestPlayerJoinAcceptsWrongPicturesModule(t *testing.T) {
	h := NewQuizHub(Dependencies{PlayerIdentity: &stubPlayerIdentity{
		event: activeEvent(models.ModuleTypeWrongPictures), proofHas: true, proofValid: true,
	}})
	t.Cleanup(h.Shutdown)
	c := &Client{send: make(chan []byte, 4)}
	h.Register(c)
	h.ProcessMessage(c, joinPayload("m"))
	if c.ModuleID != "m" || c.PlayerID != "p1" {
		t.Fatalf("wrong_pictures player failed to join: moduleID=%q playerID=%q", c.ModuleID, c.PlayerID)
	}
}

// M2 bonus: non-quiz modules must still be rejected with an explicit error.
func TestPlayerJoinRejectsNonQuizModule(t *testing.T) {
	h := NewQuizHub(Dependencies{PlayerIdentity: &stubPlayerIdentity{
		event: activeEvent(models.ModuleTypeForm), proofHas: true, proofValid: true,
	}})
	t.Cleanup(h.Shutdown)
	c := &Client{send: make(chan []byte, 4)}
	h.Register(c)
	h.ProcessMessage(c, joinPayload("m"))
	msg := readClientMessage(t, c)
	if msg["type"] != "cmd:error" {
		t.Fatalf("message type = %v, want cmd:error", msg["type"])
	}
	payload := payloadOf(t, msg)
	if payload["code"] != "join_closed" {
		t.Fatalf("cmd:error code = %v, want join_closed", payload["code"])
	}
}
