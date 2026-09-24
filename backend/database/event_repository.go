package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventRepository struct {
	collection *mongo.Collection
}

func NewEventRepository() *EventRepository {
	return &EventRepository{collection: db.GetCollection("events")}
}

func (r *EventRepository) Create(ctx context.Context, event *models.Event) error {
	_, err := r.collection.InsertOne(ctx, event)
	return err
}

func (r *EventRepository) PINExists(ctx context.Context, pin string) (bool, error) {
	n, err := r.collection.CountDocuments(ctx, bson.M{"$or": []bson.M{{"pin": pin}, {"_id": pin}}})
	return n > 0, err
}

func (r *EventRepository) FindByPIN(ctx context.Context, pin string) (*models.Event, error) {
	var event models.Event
	err := r.collection.FindOne(ctx, bson.M{"$or": []bson.M{{"pin": pin}, {"_id": pin}}}).Decode(&event)
	return &event, err
}

func (r *EventRepository) Get(ctx context.Context, id string) (*models.Event, error) {
	var event models.Event
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&event)
	return &event, err
}

func (r *EventRepository) GetByModule(ctx context.Context, moduleID string) (*models.Event, error) {
	var event models.Event
	err := r.collection.FindOne(ctx, bson.M{"modules.id": moduleID}).Decode(&event)
	return &event, err
}

func (r *EventRepository) List(ctx context.Context) ([]models.Event, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	events := make([]models.Event, 0)
	err = cursor.All(ctx, &events)
	return events, err
}

func (r *EventRepository) Update(ctx context.Context, id string, event *models.Event) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"name": event.Name, "description": event.Description, "language": event.Language,
	}})
	return err
}

func (r *EventRepository) Delete(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *EventRepository) SetStatus(ctx context.Context, id string, active bool) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"isActive": active}})
	return err
}
