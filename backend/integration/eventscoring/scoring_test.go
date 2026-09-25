//go:build integration

package eventscoring_test

import (
	"context"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestEventScoringLatestAndManual(t *testing.T) {
	container := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, container)
	previous := db.Client
	db.Client = container.Client
	defer func() { db.Client = previous }()
	ctx := context.Background()
	if _, err := db.GetCollection("events").InsertOne(ctx, models.Event{ID: "e", Modules: []models.Module{{ID: "q", Type: models.ModuleTypeQuiz}, {ID: "w", Type: models.ModuleTypeWrongPictures}, {ID: "f", Type: models.ModuleTypeForm}}}); err != nil {
		t.Fatal(err)
	}
	results := db.GetCollection("quiz_results")
	oldID, latestID := primitive.NewObjectID(), primitive.NewObjectID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	for _, doc := range []bson.M{
		{"_id": oldID, "eventId": "e", "moduleId": "q", "finishedAt": now.Add(-time.Hour), "players": []models.PlayerResult{{ID: "p", Score: 10}, {ID: "old-only", Score: 100}}},
		{"_id": latestID, "eventId": "e", "moduleId": "q", "finishedAt": now, "players": []models.PlayerResult{{ID: "p", Score: 20}}},
		{"eventId": "e", "moduleId": "w", "finishedAt": now, "players": []models.PlayerResult{{ID: "p", Score: 7}}},
		{"eventId": "other", "moduleId": "q", "finishedAt": now.Add(time.Hour), "players": []models.PlayerResult{{ID: "p", Score: 900}}},
	} {
		if _, err := results.InsertOne(ctx, doc); err != nil {
			t.Fatal(err)
		}
	}
	repo := database.NewModuleRepository()
	scores, err := database.LoadEventScores(ctx, "e")
	if err != nil || scores.PlayerTotal("p") != 27 || scores.PlayerTotal("old-only") != 0 {
		t.Fatalf("latest scores: %+v, %v", scores, err)
	}
	if found, err := repo.UpdateScore(ctx, "e", "q", "p", 40); err != nil || !found {
		t.Fatalf("update: %v %v", found, err)
	}
	if found, err := repo.UpdateScore(ctx, "e", "q", "old-only", 50); err != nil || found {
		t.Fatalf("must not edit old run: %v %v", found, err)
	}
	var old models.QuizResult
	if err := results.FindOne(ctx, bson.M{"_id": oldID}).Decode(&old); err != nil || old.Players[0].Score != 10 {
		t.Fatalf("historical score changed: %+v %v", old, err)
	}
	manual := database.NewScoreEntryRepository()
	for _, points := range []float64{3, 5, 5} {
		if err := manual.SetManual(ctx, "e", "q", "players", "p", points); err != nil {
			t.Fatal(err)
		}
	}
	if err := manual.SetManual(ctx, "e", "q", "teams", "t", 100); err != nil {
		t.Fatal(err)
	}
	scores, err = database.LoadEventScores(ctx, "e")
	if err != nil || scores.PlayerTotal("p") != 52 || len(scores.Entries) != 2 {
		t.Fatalf("manual points/retry: %+v %v", scores, err)
	}
	if _, err := db.GetCollection("participants").InsertOne(ctx, models.Participant{ID: "p", EventID: "e"}); err != nil {
		t.Fatal(err)
	}
	_, total, err := database.NewEventRepository().ParticipantStatus(ctx, "e", "p")
	if err != nil || total != 52 {
		t.Fatalf("participant total: %v %v", total, err)
	}
	// Legacy form responses use Mongo ObjectIDs. Edits must be scoped to their event/module.
	responseID := primitive.NewObjectID()
	if _, err := db.GetCollection("responses").InsertOne(ctx, bson.M{"_id": responseID, "eventId": "e", "moduleId": "f", "data": bson.M{"name": "Old"}}); err != nil {
		t.Fatal(err)
	}
	if found, err := repo.UpdateResponse(ctx, "other", "f", responseID.Hex(), map[string]any{"name": "Wrong"}); err != nil || found {
		t.Fatalf("cross-event form edit: %v %v", found, err)
	}
	if found, err := repo.UpdateResponse(ctx, "e", "f", responseID.Hex(), map[string]any{"name": "New"}); err != nil || !found {
		t.Fatalf("ObjectID form edit: %v %v", found, err)
	}
}
