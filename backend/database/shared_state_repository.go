package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SharedStateRepository struct {
	collection *mongo.Collection
}

func NewSharedStateRepository() *SharedStateRepository {
	return &SharedStateRepository{
		collection: db.GetCollection("shared_state"),
	}
}

func (r *SharedStateRepository) Create(ctx context.Context, state *models.SharedStateValue) error {
	_, err := r.collection.InsertOne(ctx, state)
	return err
}

func (r *SharedStateRepository) GetByKey(ctx context.Context, sessionID, key string) (*models.SharedStateValue, error) {
	var state models.SharedStateValue
	err := r.collection.FindOne(ctx, bson.M{"eventSessionId": sessionID, "key": key}).Decode(&state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}

func (r *SharedStateRepository) GetBySessionID(ctx context.Context, sessionID string) ([]models.SharedStateValue, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventSessionId": sessionID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var states []models.SharedStateValue
	if err := cursor.All(ctx, &states); err != nil {
		return nil, err
	}
	return states, nil
}

func (r *SharedStateRepository) Update(ctx context.Context, state *models.SharedStateValue) error {
	filter := bson.M{"_id": state.ID}
	update := bson.M{"$set": state}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}
