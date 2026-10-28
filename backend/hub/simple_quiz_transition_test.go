package hub

import (
	"errors"
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
