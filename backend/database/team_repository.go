package database

import (
	"context"

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
		collection: db.GetCollection("teams"),
	}
}

func (r *TeamRepository) Create(ctx context.Context, team *models.Team) error {
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

func (r *TeamRepository) GetBySessionID(ctx context.Context, sessionID string) ([]models.Team, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventSessionId": sessionID})
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

// GetByEventSessionID retrieves all teams for an event session
func (r *TeamRepository) GetByEventSessionID(ctx context.Context, eventSessionID string) ([]models.Team, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"eventSessionId": eventSessionID})
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

// DeleteBySessionID removes all teams for an event session
func (r *TeamRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"eventSessionId": sessionID})
	return err
}

// AddMember adds a participant to a team
func (r *TeamRepository) AddMember(ctx context.Context, teamID, participantID string) error {
	filter := bson.M{"_id": teamID}
	update := bson.M{"$addToSet": bson.M{"memberParticipantIds": participantID}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// RemoveMember removes a participant from a team
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, participantID string) error {
	filter := bson.M{"_id": teamID}
	update := bson.M{"$pull": bson.M{"memberParticipantIds": participantID}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

// BulkCreate creates multiple teams at once
func (r *TeamRepository) BulkCreate(ctx context.Context, teams []*models.Team) error {
	if len(teams) == 0 {
		return nil
	}
	documents := make([]interface{}, len(teams))
	for i, team := range teams {
		documents[i] = team
	}
	_, err := r.collection.InsertMany(ctx, documents)
	return err
}

// UpdateTeamName updates a team's name
func (r *TeamRepository) UpdateTeamName(ctx context.Context, teamID, name string) error {
	filter := bson.M{"_id": teamID}
	update := bson.M{"$set": bson.M{"name": name}}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}
