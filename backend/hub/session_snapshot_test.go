package hub

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestGetSessionReturnsIndependentSnapshot(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.CreateSession("m", "e")
	imageID := "image"
	h.UpdateSession("m", func(s *models.QuizSession) {
		s.Players["p"] = &models.Player{ID: "p", Score: 10}
		s.PlayerSignals = []models.PlayerSignal{{AnswerIndices: []int{1}, Guesses: []models.PlayerGuess{{TargetID: "t"}}}}
		s.Questions = []models.QuizQuestion{{Answers: []string{"A"}, ImageID: &imageID, WrongPictureConfig: &models.WrongPictureConfig{Targets: []models.WrongPictureTarget{{ID: "target"}}}}}
		s.Batches = CalculateBatches(s.Questions)
	})
	snapshot := h.GetSession("m")
	h.UpdateSession("m", func(s *models.QuizSession) {
		s.Players["p"].Score = 20
		s.PlayerSignals[0].AnswerIndices[0] = 2
		s.PlayerSignals[0].Guesses[0].TargetID = "new"
		s.Questions[0].Answers[0] = "B"
		*s.Questions[0].ImageID = "new-image"
		s.Questions[0].WrongPictureConfig.Targets[0].ID = "new-target"
	})
	q := snapshot.Questions[0]
	if snapshot.Players["p"].Score != 10 || snapshot.PlayerSignals[0].AnswerIndices[0] != 1 || snapshot.PlayerSignals[0].Guesses[0].TargetID != "t" || q.Answers[0] != "A" || *q.ImageID != "image" || q.WrongPictureConfig.Targets[0].ID != "target" || snapshot.Batches[0].Questions[0].Answers[0] != "A" {
		t.Fatal("snapshot changed after the live session was updated")
	}
}

func TestGetSessionConcurrentReadAndUpdate(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.CreateSession("m", "e")
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			h.UpdateSession("m", func(s *models.QuizSession) { s.CurrentIndex++; s.Players["p"] = &models.Player{Score: i} })
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			if _, err := json.Marshal(h.GetSession("m")); err != nil {
				t.Error(err)
				return
			}
		}
	}()
	wg.Wait()
}
