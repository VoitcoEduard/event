package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ModuleRepository struct {
	events      *mongo.Collection
	responses   *mongo.Collection
	quizResults *mongo.Collection
	teamResults *mongo.Collection
}

func NewModuleRepository() *ModuleRepository {
	return &ModuleRepository{
		events: db.GetCollection("events"), responses: db.GetCollection("responses"),
		quizResults: db.GetCollection("quiz_results"), teamResults: db.GetCollection("team_results"),
	}
}

func (r *ModuleRepository) Add(ctx context.Context, eventID string, module models.Module) error {
	_, _ = r.events.UpdateOne(ctx, bson.M{"_id": eventID, "modules": nil}, bson.M{"$set": bson.M{"modules": []models.Module{}}})
	_, err := r.events.UpdateOne(ctx, bson.M{"_id": eventID}, bson.M{"$push": bson.M{"modules": module}})
	return err
}

func (r *ModuleRepository) SetActive(ctx context.Context, eventID, moduleID string, active bool) (bool, error) {
	result, err := r.events.UpdateOne(ctx, bson.M{"_id": eventID, "modules.id": moduleID}, bson.M{"$set": bson.M{"modules.$.isActive": active}})
	return err == nil && result.MatchedCount > 0, err
}

func (r *ModuleRepository) Publish(ctx context.Context, eventID, moduleID string, data map[string]any) (bool, error) {
	result, err := r.events.UpdateOne(ctx, bson.M{"_id": eventID, "modules.id": moduleID}, bson.M{"$set": bson.M{"modules.$.data": data}})
	return err == nil && result.MatchedCount > 0, err
}

func (r *ModuleRepository) Delete(ctx context.Context, eventID, moduleID string) (bool, error) {
	result, err := r.events.UpdateOne(ctx, bson.M{"_id": eventID}, bson.M{"$pull": bson.M{"modules": bson.M{"id": moduleID}}})
	return err == nil && result.ModifiedCount > 0, err
}

func (r *ModuleRepository) UpdateData(ctx context.Context, eventID, moduleID string, data map[string]any) error {
	if data == nil {
		return nil
	}
	fields := bson.M{}
	for key, value := range data {
		fields["modules.$.data."+key] = value
	}
	_, err := r.events.UpdateOne(ctx, bson.M{"_id": eventID, "modules.id": moduleID}, bson.M{"$set": fields})
	return err
}

func (r *ModuleRepository) GetEvent(ctx context.Context, eventID string) (*models.Event, error) {
	var event models.Event
	err := r.events.FindOne(ctx, bson.M{"_id": eventID}).Decode(&event)
	return &event, err
}

func (r *ModuleRepository) InsertResponse(ctx context.Context, response *models.Response) error {
	_, err := r.responses.InsertOne(ctx, response)
	return err
}

func (r *ModuleRepository) TeamResults(ctx context.Context, eventID, moduleID string) ([]models.TeamResult, error) {
	cursor, err := r.teamResults.Find(ctx, bson.M{"eventId": eventID, "moduleId": moduleID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var results []models.TeamResult
	err = cursor.All(ctx, &results)
	return results, err
}

func (r *ModuleRepository) QuizResults(ctx context.Context, eventID, moduleID string) ([]models.QuizResult, error) {
	cursor, err := r.quizResults.Find(ctx, bson.M{"eventId": eventID, "moduleId": moduleID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var results []models.QuizResult
	err = cursor.All(ctx, &results)
	return results, err
}

func (r *ModuleRepository) FormResponses(ctx context.Context, eventID, moduleID string) ([]models.Response, error) {
	cursor, err := r.responses.Find(ctx, bson.M{"eventId": eventID, "moduleId": moduleID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var results []models.Response
	err = cursor.All(ctx, &results)
	return results, err
}

func (r *ModuleRepository) UpdateResponse(ctx context.Context, responseID string, data map[string]any) (bool, error) {
	result, err := r.responses.UpdateOne(ctx, bson.M{"_id": responseID}, bson.M{"$set": bson.M{"data": data}})
	return err == nil && result.MatchedCount > 0, err
}

func (r *ModuleRepository) UpdateScore(ctx context.Context, eventID, moduleID, userID string, score int) (bool, error) {
	result, err := r.quizResults.UpdateOne(ctx,
		bson.M{"eventId": eventID, "moduleId": moduleID, "players.id": userID},
		bson.M{"$set": bson.M{"players.$.score": score}})
	return err == nil && result.MatchedCount > 0, err
}
