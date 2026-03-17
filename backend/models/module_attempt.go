package models

import "time"

type ModuleAttemptStatus string

const (
	ModuleAttemptStatusNotStarted ModuleAttemptStatus = "not_started"
	ModuleAttemptStatusInProgress ModuleAttemptStatus = "in_progress"
	ModuleAttemptStatusSubmitted  ModuleAttemptStatus = "submitted"
	ModuleAttemptStatusCompleted  ModuleAttemptStatus = "completed"
)

type ModuleAttempt struct {
	ID            string              `json:"id" bson:"_id"`
	EventID       string              `json:"eventId" bson:"eventId"`
	ParticipantID string              `json:"participantId" bson:"participantId"`
	ModuleID      string              `json:"moduleId" bson:"moduleId"`
	Status        ModuleAttemptStatus `json:"status" bson:"status"`
	ScoreDelta    float64             `json:"scoreDelta" bson:"scoreDelta"`
	ResultData    map[string]any      `json:"resultData" bson:"resultData"`
	CreatedAt     time.Time           `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt" bson:"updatedAt"`
}
