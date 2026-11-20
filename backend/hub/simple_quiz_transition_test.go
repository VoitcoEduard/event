package hub

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

func TestSimpleQuizRejectsStaleTransition(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	now := time.Now().UTC()
	h.SetSimpleQuiz(&models.SimpleQuizSession{ModuleID: "module", Status: models.SimpleQuizStateLobby, CreatedAt: now, UpdatedAt: now})
	snapshot := h.GetSimpleQuiz("module")
	if _, err := h.TryUpdateSimpleQuizExpected("module", snapshot, func(s *models.SimpleQuizSession) { s.Status = models.SimpleQuizStateQuestionOpen }); err != nil {
		t.Fatal(err)
	}
	if _, err := h.TryUpdateSimpleQuizExpected("module", snapshot, func(s *models.SimpleQuizSession) { s.Status = models.SimpleQuizStateCompleted }); !errors.Is(err, postgres.ErrConflict) {
		t.Fatalf("stale transition error = %v", err)
	}
	if got := h.GetSimpleQuiz("module").Status; got != models.SimpleQuizStateQuestionOpen {
		t.Fatalf("state = %q", got)
	}
}

func TestTryCreateSimpleQuizRejectsConcurrentOrDuplicateLaunch(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	const n = 10
	start := make(chan struct{})
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			now := time.Now().UTC()
			errs <- h.TryCreateSimpleQuiz(&models.SimpleQuizSession{
				ModuleID:  "module-launch",
				Title:     "Quiz",
				Status:    models.SimpleQuizStateLobby,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	created, conflicts := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, postgres.ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if created != 1 || conflicts != n-1 {
		t.Fatalf("created=%d conflicts=%d, want 1 and %d", created, conflicts, n-1)
	}
}

func TestSubmitSimpleQuizAnswerRejectsEmptyStoredSecret(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.SetSimpleQuiz(&models.SimpleQuizSession{
		ModuleID:  "m-empty-sec",
		Status:    models.SimpleQuizStateQuestionOpen,
		Questions: []models.SimpleQuizQuestionSnapshot{{ID: "q1", Text: "Q1"}},
		Players: map[string]*models.SimpleQuizPlayer{
			"p1": {ID: "p1", Nickname: "NoSec", Secret: ""},
		},
	})
	if _, code := h.SubmitSimpleQuizAnswer("m-empty-sec", "p1", "", "a1", 0, "hello", time.Now()); code != SimpleQuizSubmitUnknownPlayer {
		t.Fatalf("code = %q, want %q", code, SimpleQuizSubmitUnknownPlayer)
	}
}
