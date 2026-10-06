package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventSessionRepository struct {
	collection *mongo.Collection
}

func NewEventSessionRepository() *EventSessionRepository {
	return &EventSessionRepository{
		collection: db.GetCollection("event_sessions"),
	}
}

func (r *EventSessionRepository) Create(ctx context.Context, session *models.EventSession) error {
	_, err := r.collection.InsertOne(ctx, session)
	return err
}

func (r *EventSessionRepository) GetByID(ctx context.Context, id string) (*models.EventSession, error) {
	var session models.EventSession
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *EventSessionRepository) GetByEventID(ctx context.Context, eventID string) ([]models.EventSession, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []models.EventSession
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *EventSessionRepository) Update(ctx context.Context, session *models.EventSession) error {
	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": buildEventSessionUpdate(session)}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// buildEventSessionUpdate returns only mutable fields for $set.
// It intentionally excludes immutable identity fields (_id, eventId,
// createdAt) and omits nil pointers/slices so a partial input cannot
// null out previously stored StartedAt, EndedAt, ModuleOrder, etc.
func buildEventSessionUpdate(session *models.EventSession) bson.M {
	set := bson.M{
		"status":    session.Status,
		"updatedAt": session.UpdatedAt,
	}
	if session.ActiveModuleID != "" {
		set["activeModuleId"] = session.ActiveModuleID
	}
	if session.ModuleOrder != nil {
		set["moduleOrder"] = session.ModuleOrder
	}
	if session.ModuleStates != nil {
		set["moduleStates"] = session.ModuleStates
	}
	if session.StartedAt != nil {
		set["startedAt"] = session.StartedAt
	}
	if session.EndedAt != nil {
		set["endedAt"] = session.EndedAt
	}
	return set
}
