//go:build integration

package eventscoring_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestEventScoringLatestAndManual(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	eventID, otherID, q, w, f := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	event := &models.Event{ID: eventID, Pin: "123456", CreatedAt: now, Modules: []models.Module{{ID: q, Type: models.ModuleTypeQuiz}, {ID: w, Type: models.ModuleTypeWrongPictures}, {ID: f, Type: models.ModuleTypeForm}}}
	if err := db.CreateEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateEvent(ctx, &models.Event{ID: otherID, Pin: "654321", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	for _, v := range []models.QuizResult{{EventID: eventID, ModuleID: q, FinishedAt: now.Add(-time.Hour), Players: []models.PlayerResult{{ID: "p", Score: 10}, {ID: "old-only", Score: 100}}}, {EventID: eventID, ModuleID: q, FinishedAt: now, Players: []models.PlayerResult{{ID: "p", Score: 20}}}, {EventID: eventID, ModuleID: w, FinishedAt: now, Players: []models.PlayerResult{{ID: "p", Score: 7}}}} {
		if err := db.SaveQuizResult(ctx, uuid.NewString(), &v); err != nil {
			t.Fatal(err)
		}
	}
	repo := database.NewModuleRepository(db)
	scores, err := database.LoadEventScores(ctx, db, eventID)
	if err != nil || scores.PlayerTotal("p") != 27 || scores.PlayerTotal("old-only") != 0 {
		t.Fatalf("latest: %+v %v", scores, err)
	}
	if found, err := repo.UpdateScore(ctx, eventID, q, "p", 40); err != nil || !found {
		t.Fatalf("update: %v %v", found, err)
	}
	if found, err := repo.UpdateScore(ctx, eventID, q, "old-only", 50); err != nil || found {
		t.Fatalf("historical update: %v %v", found, err)
	}
	manual := database.NewScoreEntryRepository(db)
	for _, points := range []float64{3, 5, 5} {
		if err = manual.SetManual(ctx, eventID, q, "players", "p", points); err != nil {
			t.Fatal(err)
		}
	}
	if err = manual.SetManual(ctx, eventID, q, "teams", "t", 100); err != nil {
		t.Fatal(err)
	}
	scores, err = database.LoadEventScores(ctx, db, eventID)
	if err != nil || scores.PlayerTotal("p") != 52 || len(scores.Entries) != 2 {
		t.Fatalf("manual: %+v %v", scores, err)
	}
	if _, err = db.EnsureParticipant(ctx, &models.Participant{ID: "p", EventID: eventID, Status: models.ParticipantStatusJoined, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	_, total, err := database.NewEventRepository(db).ParticipantStatus(ctx, eventID, "p")
	if err != nil || total != 52 {
		t.Fatalf("participant score: %v %v", total, err)
	}
	response := &models.Response{ID: uuid.NewString(), EventID: eventID, ModuleID: f, UserID: "p", Data: map[string]any{"name": "Old"}, CreatedAt: now}
	if err = db.InsertResponse(ctx, response); err != nil {
		t.Fatal(err)
	}
	if found, err := repo.UpdateResponse(ctx, otherID, f, response.ID, map[string]any{"name": "Wrong"}); err != nil || found {
		t.Fatalf("cross-event edit: %v %v", found, err)
	}
	if found, err := repo.UpdateResponse(ctx, eventID, f, response.ID, map[string]any{"name": "New"}); err != nil || !found {
		t.Fatalf("edit: %v %v", found, err)
	}
}
