package models

import "time"

type ScoreCategory string

const (
	ScoreCategoryBase       ScoreCategory = "base"
	ScoreCategoryBonus      ScoreCategory = "bonus"
	ScoreCategoryPenalty    ScoreCategory = "penalty"
	ScoreCategoryAdjustment ScoreCategory = "adjustment"
)

type ScoreEntry struct {
	ID            string        `json:"id" bson:"_id"`
	EventID       string        `json:"eventId" bson:"eventId"`
	ParticipantID string        `json:"participantId,omitempty" bson:"participantId,omitempty"`
	TeamID        string        `json:"teamId,omitempty" bson:"teamId,omitempty"`
	ModuleID      string        `json:"moduleId" bson:"moduleId"`
	Category      ScoreCategory `json:"category" bson:"category"`
	Points        float64       `json:"points" bson:"points"`
	Reason        string        `json:"reason,omitempty" bson:"reason,omitempty"`
	CreatedAt     time.Time     `json:"createdAt" bson:"createdAt"`
}
