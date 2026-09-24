package database

import (
	"context"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TeamRepository struct {
	collection *mongo.Collection
}

func NewTeamRepository() *TeamRepository {
	return &TeamRepository{
		collection: db.GetCollection("event_teams"),
	}
}

func (r *TeamRepository) Create(ctx context.Context, team *models.Team) error {
	team.CreatedAt = time.Now()
	_, err := r.collection.InsertOne(ctx, team)
	return err
}

func (r *TeamRepository) GetByID(ctx context.Context, id string) (*models.Team, error) {
	var team models.Team
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&team)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepository) GetByEventID(ctx context.Context, eventID string) ([]models.Team, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var teams []models.Team
	if err := cursor.All(ctx, &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *TeamRepository) Update(ctx context.Context, team *models.Team) error {
	filter := bson.M{"_id": team.ID}
	update := bson.M{"$set": team}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *TeamRepository) Delete(ctx context.Context, teamID string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": teamID})
	return err
}

func (r *TeamRepository) DeleteAllForEvent(ctx context.Context, eventID string) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"eventId": eventID})
	return err
}
