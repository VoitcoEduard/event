//go:build integration

package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestEventPINMigrationAndLookup(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	container := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, container)
	previous := db.Client
	db.Client = container.Client
	defer func() { db.Client = previous }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	events := db.GetCollection("events")
	if _, err := events.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "pin", Value: 1}}}); err != nil {
		t.Fatal(err)
	}
	db.EnsureIndexes()

	service := NewEventService()
	event := &models.Event{Name: "PIN test"}
	if err := service.Create(ctx, event); err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(event.ID); err != nil {
		t.Fatalf("event ID %q is not a UUID: %v", event.ID, err)
	}
	if len(event.Pin) != 6 {
		t.Fatalf("PIN %q is not six digits", event.Pin)
	}
	found, err := service.FindByPIN(ctx, event.Pin)
	if err != nil || found.ID != event.ID {
		t.Fatalf("PIN lookup = %+v, %v", found, err)
	}
	if _, err := events.InsertOne(ctx, models.Event{ID: uuid.NewString(), Pin: event.Pin}); !mongo.IsDuplicateKeyError(err) {
		t.Fatalf("duplicate PIN insert error = %v, want duplicate key", err)
	}
	if _, err := events.InsertOne(ctx, models.Event{ID: "123456", Name: "legacy"}); err != nil {
		t.Fatal(err)
	}
	legacy, err := service.FindByPIN(ctx, "123456")
	if err != nil || legacy.ID != "123456" {
		t.Fatalf("legacy PIN lookup = %+v, %v", legacy, err)
	}
}
