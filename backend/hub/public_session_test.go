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
	if bytes.Contains(public, []byte("playerSignals")) || !bytes.Contains(public, []byte("Player")) {
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
	public := publicBatchMessage(batch, 0)
	for _, secret := range [][]byte{[]byte("correctIndex"), []byte("Future secret"), []byte("hidden-target")} {
		if bytes.Contains(public, secret) {
			t.Fatalf("public batch leaked %s: %s", secret, public)
		}
	}
}
