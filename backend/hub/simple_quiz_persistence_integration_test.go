//go:build integration

package hub

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestSimpleQuizRestoreAndConcurrentTransitions(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC()
	if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: "246810", CreatedAt: now, Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeSimpleQuiz}}}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveSimpleQuiz(ctx, nil, &models.SimpleQuizSession{EventID: eventID, ModuleID: moduleID, Title: "Restored quiz", Status: models.SimpleQuizStateQuestionOpen, Questions: []models.SimpleQuizQuestionSnapshot{{ID: "q", Text: "Question"}}, CreatedAt: now, UpdatedAt: now, DeadlineUnix: now.Add(time.Minute).Unix()}); err != nil {
		t.Fatal(err)
	}
	h := NewQuizHub(Dependencies{DB: db})
	const n = 12
	snapshots := make([]*models.SimpleQuizSession, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			snapshots[i] = h.GetSimpleQuiz(moduleID)
		}(i)
	}
	wg.Wait()
	for _, snapshot := range snapshots {
		if snapshot == nil || snapshot.Status != models.SimpleQuizStatePaused || snapshot.DeadlineUnix != 0 || snapshot.Title != "Restored quiz" || len(snapshot.Questions) != 1 {
			t.Fatalf("restored snapshot: %+v", snapshot)
		}
		if !snapshot.UpdatedAt.Equal(snapshots[0].UpdatedAt) {
			t.Fatal("concurrent readers repeated the recovery transition")
		}
	}
	// Two actions based on the same restored snapshot must commit only once.
	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := h.TryUpdateSimpleQuizExpected(moduleID, snapshots[0], func(s *models.SimpleQuizSession) {
				s.Status = models.SimpleQuizStateBetweenQuestions
				s.CurrentIndex++
			})
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	committed, conflicts := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			committed++
		case errors.Is(err, postgres.ErrConflict):
			conflicts++
		default:
			t.Fatalf("transition: %v", err)
		}
	}
	if committed != 1 || conflicts != 1 {
		t.Fatalf("committed=%d conflicts=%d", committed, conflicts)
	}
	stored, err := db.ReadSimpleQuiz(ctx, moduleID)
	if err != nil || stored.CurrentIndex != 1 || stored.Status != models.SimpleQuizStateBetweenQuestions {
		t.Fatalf("persisted transition: %+v error=%v", stored, err)
	}
}
