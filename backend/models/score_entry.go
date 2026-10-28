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
	ID            string        `json:"id"`
	EventID       string        `json:"eventId"`
	ParticipantID string        `json:"participantId,omitempty"`
	TeamID        string        `json:"teamId,omitempty"`
	ModuleID      string        `json:"moduleId"`
	Category      ScoreCategory `json:"category"`
	Points        float64       `json:"points"`
	Reason        string        `json:"reason,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
}
