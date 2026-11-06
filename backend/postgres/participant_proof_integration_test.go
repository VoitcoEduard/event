//go:build integration

package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

func TestParticipantProofLinksPublishedSimpleQuizOnly(t *testing.T) {
	ctx, db := startDB(t)
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	event := &models.Event{ID: eventID, Pin: "345678", CreatedAt: time.Now().UTC(), Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeSimpleQuiz}}}
	if err := db.CreateEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	participant, err := db.JoinEventParticipant(ctx, eventID, "device", "", "Alice", nil, "")
	if err != nil || participant.JoinSecret == "" {
		t.Fatalf("new participant: %+v %v", participant, err)
	}
	if _, err := db.JoinEventParticipant(ctx, eventID, "device", "", "Alice", nil, ""); !errors.Is(err, postgres.ErrConflict) {
		t.Fatalf("public ID claimed without proof: %v", err)
	}
	if _, err := db.JoinEventParticipant(ctx, eventID, "device", "", "Alice", nil, participant.JoinSecret); err != nil {
		t.Fatalf("valid rejoin: %v", err)
	}
	has, valid, err := db.CheckParticipantProof(ctx, eventID, "device", participant.JoinSecret)
	if err != nil || !has || !valid {
		t.Fatalf("proof check: %v %v %v", has, valid, err)
	}
	now := time.Now().UTC()
	session := &models.SimpleQuizSession{EventID: eventID, ModuleID: moduleID, Status: models.SimpleQuizStateResultsPublished,
		Players:   map[string]*models.SimpleQuizPlayer{"sq_one": {ID: "sq_one", ParticipantID: "device", Nickname: "Alice", Secret: "private"}},
		Published: &models.SimpleQuizPublishedSnapshot{Version: 1, PublishedAt: now, Players: []models.SimpleQuizPlayerResult{{PlayerID: "sq_one", Nickname: "Alice", TotalScore: 42}}}, UpdatedAt: now}
	if err := db.SaveSimpleQuiz(ctx, nil, session); err != nil {
		t.Fatal(err)
	}
	scores, err := database.LoadEventScores(ctx, db, eventID)
	if err != nil || scores.PlayerTotal("device") != 42 || scores.PlayerTotal("sq_one") != 0 {
		t.Fatalf("published score not linked: %+v %v", scores, err)
	}
	total, err := db.PlayerTotal(ctx, eventID, "device")
	if err != nil || total != 42 {
		t.Fatalf("participant total: %v %v", total, err)
	}
	if err := db.DeleteSimpleQuiz(ctx, moduleID); err != nil {
		t.Fatal(err)
	}
	total, err = db.PlayerTotal(ctx, eventID, "device")
	if err != nil || total != 0 {
		t.Fatalf("reset score: %v %v", total, err)
	}
}
