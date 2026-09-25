package database

import (
	"context"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
)

type EventScores struct {
	Games   []models.QuizResult `json:"games"`
	Entries []models.ScoreEntry `json:"entries"`
}

func LoadEventScores(ctx context.Context, eventID string) (*EventScores, error) {
	event, err := NewEventRepository().Get(ctx, eventID)
	if err != nil {
		return nil, err
	}
	moduleIDs := bson.A{}
	modules := make(map[string]bool)
	for _, module := range event.Modules {
		modules[module.ID] = true
		if module.Type == models.ModuleTypeQuiz || module.Type == models.ModuleTypeWrongPictures {
			moduleIDs = append(moduleIDs, module.ID)
		}
	}
	cursor, err := db.GetCollection("quiz_results").Aggregate(ctx, []bson.M{
		{"$match": bson.M{"eventId": eventID, "moduleId": bson.M{"$in": moduleIDs}}},
		{"$sort": bson.D{{Key: "finishedAt", Value: -1}, {Key: "_id", Value: -1}}},
		{"$group": bson.M{"_id": "$moduleId", "latest": bson.M{"$first": "$$ROOT"}}},
		{"$replaceRoot": bson.M{"newRoot": "$latest"}},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	result := &EventScores{Games: []models.QuizResult{}}
	if err = cursor.All(ctx, &result.Games); err != nil {
		return nil, err
	}
	entries, err := NewScoreEntryRepository().GetByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	result.Entries = []models.ScoreEntry{}
	for _, entry := range entries {
		if modules[entry.ModuleID] {
			result.Entries = append(result.Entries, entry)
		}
	}
	return result, nil
}

func (s *EventScores) PlayerTotal(playerID string) float64 {
	total := 0.0
	for _, game := range s.Games {
		for _, player := range game.Players {
			if player.ID == playerID {
				total += float64(player.Score)
			}
		}
	}
	for _, entry := range s.Entries {
		if entry.ParticipantID == playerID {
			total += entry.Points
		}
	}
	return total
}
