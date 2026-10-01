package database

import (
	"context"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TeamsActivityRepository struct{ collection *mongo.Collection }

func NewTeamsActivityRepository() *TeamsActivityRepository {
	return &TeamsActivityRepository{db.GetCollection("teams_activities")}
}
func (r *TeamsActivityRepository) Get(ctx context.Context, eventID, moduleID string) (*models.TeamsActivity, error) {
	var a models.TeamsActivity
	err := r.collection.FindOne(ctx, bson.M{"eventId": eventID, "moduleId": moduleID, "deleted": bson.M{"$ne": true}}).Decode(&a)
	return &a, err
}
func (r *TeamsActivityRepository) Create(ctx context.Context, a *models.TeamsActivity) error {
	_, err := r.collection.InsertOne(ctx, a)
	return err
}
func (r *TeamsActivityRepository) CompareAndSwap(ctx context.Context, a *models.TeamsActivity, revision int64) (bool, error) {
	res, err := r.collection.ReplaceOne(ctx, bson.M{"_id": a.ID, "revision": revision, "deleted": bson.M{"$ne": true}}, a)
	return err == nil && res.MatchedCount == 1, err
}
func (r *TeamsActivityRepository) ByEvent(ctx context.Context, eventID string) ([]models.TeamsActivity, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventId": eventID, "deleted": bson.M{"$ne": true}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rows := []models.TeamsActivity{}
	err = cursor.All(ctx, &rows)
	return rows, err
}
