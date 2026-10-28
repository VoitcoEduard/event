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
	ID            string              `json:"id"`
	EventID       string              `json:"eventId"`
	ParticipantID string              `json:"participantId"`
	ModuleID      string              `json:"moduleId"`
	Status        ModuleAttemptStatus `json:"status"`
	ScoreDelta    float64             `json:"scoreDelta"`
	ResultData    map[string]any      `json:"resultData"`
	CreatedAt     time.Time           `json:"createdAt"`
	UpdatedAt     time.Time           `json:"updatedAt"`
}
