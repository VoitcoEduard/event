package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
