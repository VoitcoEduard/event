package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ModuleAttemptRepository struct {
	collection *mongo.Collection
}

func NewModuleAttemptRepository() *ModuleAttemptRepository {
	return &ModuleAttemptRepository{
		collection: db.GetCollection("module_attempts"),
	}
}

func (r *ModuleAttemptRepository) Create(ctx context.Context, attempt *models.ModuleAttempt) error {
	_, err := r.collection.InsertOne(ctx, attempt)
	return err
}

func (r *ModuleAttemptRepository) GetByID(ctx context.Context, id string) (*models.ModuleAttempt, error) {
	var attempt models.ModuleAttempt
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&attempt)
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}

func (r *ModuleAttemptRepository) GetByParticipantID(ctx context.Context, participantID string) ([]models.ModuleAttempt, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"participantId": participantID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var attempts []models.ModuleAttempt
	if err := cursor.All(ctx, &attempts); err != nil {
		return nil, err
	}
	return attempts, nil
}

func (r *ModuleAttemptRepository) Update(ctx context.Context, attempt *models.ModuleAttempt) error {
	filter := bson.M{"_id": attempt.ID}
	update := bson.M{"$set": attempt}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}
