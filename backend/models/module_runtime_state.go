package models

import "time"

type ModuleParticipantStatus string

const (
	ModuleParticipantStatusNotStarted ModuleParticipantStatus = "not_started"
	ModuleParticipantStatusInProgress ModuleParticipantStatus = "in_progress"
	ModuleParticipantStatusSubmitted  ModuleParticipantStatus = "submitted"
	ModuleParticipantStatusCompleted  ModuleParticipantStatus = "completed"
)

type ModuleParticipantState struct {
	ParticipantID string                  `json:"participantId"`
	Status        ModuleParticipantStatus `json:"status"`
	ScoreDelta    float64                 `json:"scoreDelta,omitempty"`
	ResultData    map[string]any          `json:"resultData,omitempty"`
}

type ModuleRuntimeStatus string

const (
	ModuleRuntimeStatusPending   ModuleRuntimeStatus = "pending"
	ModuleRuntimeStatusReady     ModuleRuntimeStatus = "ready"
	ModuleRuntimeStatusActive    ModuleRuntimeStatus = "active"
	ModuleRuntimeStatusCompleted ModuleRuntimeStatus = "completed"
	ModuleRuntimeStatusFailed    ModuleRuntimeStatus = "failed"
	ModuleRuntimeStatusSkipped   ModuleRuntimeStatus = "skipped"
)

// ModuleRuntimeState tracks the runtime execution of a specific module within a session.
// It is typically embedded inside EventSession.ModuleStates.
type ModuleRuntimeState struct {
	ModuleID            string                            `json:"moduleId"`
	Status              ModuleRuntimeStatus               `json:"status"`
	StartedAt           *time.Time                        `json:"startedAt,omitempty"`
	EndedAt             *time.Time                        `json:"endedAt,omitempty"`
	Outputs             map[string]any                    `json:"outputs,omitempty"`
	ParticipantProgress map[string]ModuleParticipantState `json:"participantProgress,omitempty"`
}
