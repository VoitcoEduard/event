package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuizResultRepository struct {
	events       *mongo.Collection
	participants *mongo.Collection
	results      *mongo.Collection
}

func NewQuizResultRepository() *QuizResultRepository {
	return &QuizResultRepository{
		events: db.GetCollection("events"), participants: db.GetCollection("participants"),
		results: db.GetCollection("quiz_results"),
	}
}

func (r *QuizResultRepository) EventForModule(ctx context.Context, moduleID string) (*models.Event, error) {
	var event models.Event
	err := r.events.FindOne(ctx, bson.M{"modules.id": moduleID}).Decode(&event)
	return &event, err
}

func (r *QuizResultRepository) ParticipantsForEvent(ctx context.Context, eventID string) ([]models.Participant, error) {
	cursor, err := r.participants.Find(ctx, bson.M{"eventId": eventID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	participants := make([]models.Participant, 0)
	err = cursor.All(ctx, &participants)
	return participants, err
}

func (r *QuizResultRepository) Create(ctx context.Context, result *models.QuizResult) error {
	_, err := r.results.InsertOne(ctx, result)
	return err
}
