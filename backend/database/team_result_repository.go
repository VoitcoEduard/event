package database

import (
	"context"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TeamResultRepository struct {
	collection *mongo.Collection
}

func NewTeamResultRepository() *TeamResultRepository {
	return &TeamResultRepository{
		collection: db.GetCollection("team_results"),
	}
}

func (r *TeamResultRepository) Create(ctx context.Context, result *models.TeamResult) error {
	result.FinishedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, result)
	return err
}

func (r *TeamResultRepository) GetByEventModule(ctx context.Context, eventID, moduleID string) (*models.TeamResult, error) {
	var result models.TeamResult
	err := r.collection.FindOne(ctx, bson.M{"eventId": eventID, "moduleId": moduleID}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *TeamResultRepository) GetByEventID(ctx context.Context, eventID string) ([]models.TeamResult, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.TeamResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *TeamResultRepository) DeleteByModuleID(ctx context.Context, moduleID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"moduleId": moduleID})
	return err
}
