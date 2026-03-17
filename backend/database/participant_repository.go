package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ParticipantRepository struct {
	collection *mongo.Collection
}

func NewParticipantRepository() *ParticipantRepository {
	return &ParticipantRepository{
		collection: db.GetCollection("event_participants"),
	}
}

func (r *ParticipantRepository) Create(ctx context.Context, participant *models.Participant) error {
	_, err := r.collection.InsertOne(ctx, participant)
	return err
}

func (r *ParticipantRepository) GetByID(ctx context.Context, id string) (*models.Participant, error) {
	var participant models.Participant
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&participant)
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

func (r *ParticipantRepository) GetByEventID(ctx context.Context, eventID string) ([]models.Participant, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var participants []models.Participant
	if err := cursor.All(ctx, &participants); err != nil {
		return nil, err
	}
	return participants, nil
}

func (r *ParticipantRepository) Update(ctx context.Context, participant *models.Participant) error {
	filter := bson.M{"_id": participant.ID}
	update := bson.M{"$set": participant}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// UpdateTeamID updates a participant's team assignment
func (r *ParticipantRepository) UpdateTeamID(ctx context.Context, participantID, teamID string) error {
	filter := bson.M{"_id": participantID}
	update := bson.M{"$set": bson.M{"teamId": teamID}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// BulkUpdateTeamID updates team assignment for multiple participants
func (r *ParticipantRepository) BulkUpdateTeamID(ctx context.Context, participantIDs []string, teamID string) error {
	if len(participantIDs) == 0 {
		return nil
	}
	filter := bson.M{"_id": bson.M{"$in": participantIDs}}
	update := bson.M{"$set": bson.M{"teamId": teamID}}
	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

// ClearTeamID removes team assignment from all participants in an event
func (r *ParticipantRepository) ClearTeamID(ctx context.Context, eventID string) error {
	filter := bson.M{"eventId": eventID}
	update := bson.M{"$set": bson.M{"teamId": ""}, "$unset": bson.M{"teamId": 1}}
	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

// GetByTeamID retrieves all participants in a specific team
func (r *ParticipantRepository) GetByTeamID(ctx context.Context, teamID string) ([]models.Participant, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"teamId": teamID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var participants []models.Participant
	if err := cursor.All(ctx, &participants); err != nil {
		return nil, err
	}
	return participants, nil
}
