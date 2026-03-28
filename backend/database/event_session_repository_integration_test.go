//go:build integration
// +build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestEventSessionRepository_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mongoContainer := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, mongoContainer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongoContainer.GetCollection("eventdb", "event_sessions")

	session := &models.EventSession{
		ID:          "test-session-123",
		EventID:     "test-event-456",
		Status:      models.EventSessionStatusLive,
		ModuleOrder: []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, session)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	var retrieved models.EventSession
	err = collection.FindOne(ctx, bson.M{"_id": "test-session-123"}).Decode(&retrieved)
	if err != nil {
		t.Fatalf("Failed to find session: %v", err)
	}

	if retrieved.Title != "" {
		t.Errorf("Expected empty title for new session, got %q", retrieved.Title)
	}
	if retrieved.EventID != session.EventID {
		t.Errorf("Expected eventID %q, got %q", session.EventID, retrieved.EventID)
	}
}

func TestEventSessionRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mongoContainer := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, mongoContainer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongoContainer.GetCollection("eventdb", "event_sessions")

	session := &models.EventSession{
		ID:          "test-session-update",
		EventID:     "test-event-789",
		Status:      models.EventSessionStatusDraft,
		ModuleOrder: []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, session)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": bson.M{"status": models.EventSessionStatusLive}}
	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	var retrieved models.EventSession
	err = collection.FindOne(ctx, bson.M{"_id": "test-session-update"}).Decode(&retrieved)
	if err != nil {
		t.Fatalf("Failed to find session: %v", err)
	}

	if retrieved.Status != models.EventSessionStatusLive {
		t.Errorf("Expected status %q, got %q", models.EventSessionStatusLive, retrieved.Status)
	}
}

func TestEventSessionRepository_GetByEventID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mongoContainer := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, mongoContainer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongoContainer.GetCollection("eventdb", "event_sessions")

	eventID := "test-event-multiple"

	for i := 0; i < 3; i++ {
		session := &models.EventSession{
			ID:          "test-session-" + string(rune('a'+i)),
			EventID:     eventID,
			Status:      models.EventSessionStatusLive,
			ModuleOrder: []string{},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		_, err := collection.InsertOne(ctx, session)
		if err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}
	}

	cursor, err := collection.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		t.Fatalf("Failed to find sessions: %v", err)
	}
	defer cursor.Close(ctx)

	var sessions []models.EventSession
	if err := cursor.All(ctx, &sessions); err != nil {
		t.Fatalf("Failed to decode sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}
}

func TestEventSessionRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mongoContainer := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, mongoContainer)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongoContainer.GetCollection("eventdb", "event_sessions")

	session := &models.EventSession{
		ID:          "test-session-delete",
		EventID:     "test-event-delete",
		Status:      models.EventSessionStatusCompleted,
		ModuleOrder: []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, session)
	if err != nil {
		t.Fatalf("Failed to insert session: %v", err)
	}

	_, err = collection.DeleteOne(ctx, bson.M{"_id": session.ID})
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	var retrieved models.EventSession
	err = collection.FindOne(ctx, bson.M{"_id": session.ID}).Decode(&retrieved)
	if err != mongo.ErrNoDocuments {
		t.Errorf("Expected ErrNoDocuments after deletion, got: %v", err)
	}
}
