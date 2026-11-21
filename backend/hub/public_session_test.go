package hub

import (
	"bytes"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestPublicSessionUpdateOmitsAnswerHistory(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.CreateSession("m", "e")
	h.UpdateSession("m", func(s *models.QuizSession) {
		s.PlayerSignals = []models.PlayerSignal{{PlayerID: "p", AnswerIndex: 2}}
		s.Players["p"] = &models.Player{ID: "p", Nickname: "Player", Score: 10}
		s.AnsweredPlayerIDs = map[string]struct{}{"p": {}}
	})
	projector := &Client{ModuleID: "m", IsProjector: true, send: make(chan []byte, 2)}
	host := &Client{ModuleID: "m", IsHost: true, send: make(chan []byte, 2)}
	h.Register(projector)
	h.Register(host)
	h.mu.Lock()
	h.broadcastToHostsAndProjectorsLocked("m", ServerMessage{Type: "state:update", Payload: h.sessions["m"]})
	h.mu.Unlock()
	public := <-projector.send
	private := <-host.send
	if bytes.Contains(public, []byte("playerSignals")) || !bytes.Contains(public, []byte("Player")) || !bytes.Contains(public, []byte(`"answeredCount":1`)) {
		t.Fatalf("bad public projection: %s", public)
	}
	if !bytes.Contains(private, []byte("playerSignals")) {
		t.Fatalf("host lost answer history: %s", private)
	}
}

func TestPlayerProjectionContainsOnlyOwnIdentity(t *testing.T) {
	s := &models.QuizSession{ModuleID: "m", Players: map[string]*models.Player{
		"mine":  {ID: "mine", Nickname: "Me", Score: 10},
		"other": {ID: "other", Nickname: "Other", Score: 20},
	}}
	client := &Client{PlayerID: "mine"}
	public := publicSessionMessage(ServerMessage{Type: "state:update", Payload: s}, nil, s, client)
	if bytes.Contains(public, []byte("other")) || !bytes.Contains(public, []byte(`"rank":2`)) {
		t.Fatalf("player received other identity or wrong rank: %s", public)
	}
}

func TestPublicBatchOmitsAnswersAndFutureQuestions(t *testing.T) {
	batch := models.QuestionBatch{StartIndex: 0, EndIndex: 1, Questions: []models.QuizQuestion{
		{ID: "now", Text: "Current", Type: "classic", Answers: []string{"A", "B"}, CorrectIndex: 1},
		{ID: "future", Text: "Future secret", Type: "wrong_pictures", WrongPictureConfig: &models.WrongPictureConfig{Targets: []models.WrongPictureTarget{{ID: "hidden-target", XPercent: 70}}}},
	}}
	public := publicBatchMessage(batch, 0, len(batch.Questions), "run-1")
	if !bytes.Contains(public, []byte(`"totalQuestions":2`)) {
		t.Fatalf("public batch missing totalQuestions: %s", public)
	}
	for _, secret := range [][]byte{[]byte("correctIndex"), []byte("Future secret"), []byte("hidden-target")} {
		if bytes.Contains(public, secret) {
			t.Fatalf("public batch leaked %s: %s", secret, public)
		}
	}
}

func TestProjectorJoinIncludesTotalQuestions(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.CreateSession("m", "e")
	questions := []models.QuizQuestion{
		{ID: "q1", Text: "Q1", Type: "classic", Answers: []string{"A", "B"}, CorrectIndex: 0},
		{ID: "q2", Text: "Q2", Type: "classic", Answers: []string{"A", "B"}, CorrectIndex: 1},
		{ID: "q3", Text: "Q3", Type: "classic", Answers: []string{"A", "B"}, CorrectIndex: 0},
	}
	h.UpdateSession("m", func(s *models.QuizSession) {
		s.State = "question-active"
		s.CurrentIndex = 1
		s.Questions = questions
		s.Batches = CalculateBatches(questions)
	})
	projector := &Client{send: make(chan []byte, 8)}
	h.Register(projector)
	h.mu.Lock()
	h.processProjectorJoin(projector, ClientMessage{
		Type:    "projector:join",
		Payload: map[string]interface{}{"moduleId": "m", "eventId": "e"},
	})
	h.mu.Unlock()

	cmdState := <-projector.send
	stateUpdate := <-projector.send
	dataBatch := <-projector.send
	for name, msg := range map[string][]byte{"cmd:state": cmdState, "state:update": stateUpdate, "data:batch": dataBatch} {
		if !bytes.Contains(msg, []byte(`"totalQuestions":3`)) {
			t.Fatalf("%s missing totalQuestions:3 in %s", name, msg)
		}
	}
}
