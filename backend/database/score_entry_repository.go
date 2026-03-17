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

// GetByTeamID retrieves all score entries for a specific team
func (r *ScoreEntryRepository) GetByTeamID(ctx context.Context, teamID string) ([]models.ScoreEntry, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"teamId": teamID})
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

// GetTeamTotalScore calculates the total score for a team
func (r *ScoreEntryRepository) GetTeamTotalScore(ctx context.Context, teamID string) (float64, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"teamId": teamID})
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var total float64
	for cursor.Next(ctx) {
		var score models.ScoreEntry
		if err := cursor.Decode(&score); err != nil {
			return 0, err
		}
		total += score.Points
	}
	return total, nil
}
