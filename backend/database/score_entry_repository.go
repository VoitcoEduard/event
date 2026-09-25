package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ScoreEntryRepository struct {
	collection *mongo.Collection
}

func NewScoreEntryRepository() *ScoreEntryRepository {
	return &ScoreEntryRepository{
		collection: db.GetCollection("event_scores"),
	}
}

func (r *ScoreEntryRepository) Create(ctx context.Context, score *models.ScoreEntry) error {
	_, err := r.collection.InsertOne(ctx, score)
	return err
}

func (r *ScoreEntryRepository) GetByEventID(ctx context.Context, eventID string) ([]models.ScoreEntry, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var scores []models.ScoreEntry
	if err := cursor.All(ctx, &scores); err != nil {
		return nil, err
	}
	return scores, nil
}

func (r *ScoreEntryRepository) GetByParticipantID(ctx context.Context, participantID string) ([]models.ScoreEntry, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"participantId": participantID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var scores []models.ScoreEntry
	if err := cursor.All(ctx, &scores); err != nil {
		return nil, err
	}
	return scores, nil
}

// SetManual replaces one manual contribution, making retries idempotent.
func (r *ScoreEntryRepository) SetManual(ctx context.Context, eventID, moduleID, target, targetID string, points float64) error {
	key, _ := json.Marshal([]string{eventID, moduleID, target, targetID})
	entry := models.ScoreEntry{ID: "manual:" + string(key), EventID: eventID, ModuleID: moduleID,
		Category: models.ScoreCategoryAdjustment, Points: points, Reason: "Manual points", CreatedAt: time.Now()}
	if target == "players" {
		entry.ParticipantID = targetID
	} else {
		entry.TeamID = targetID
	}
	_, err := r.collection.ReplaceOne(ctx, bson.M{"_id": entry.ID}, entry, options.Replace().SetUpsert(true))
	return err
}
