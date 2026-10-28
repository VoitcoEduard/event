package models

import "time"

type EventSessionStatus string

const (
	EventSessionStatusDraft     EventSessionStatus = "draft"
	EventSessionStatusWaiting   EventSessionStatus = "waiting"
	EventSessionStatusLive      EventSessionStatus = "live"
	EventSessionStatusPaused    EventSessionStatus = "paused"
	EventSessionStatusCompleted EventSessionStatus = "completed"
)

type EventSession struct {
	ID             string               `json:"id"`
	EventID        string               `json:"eventId"`
	Status         EventSessionStatus   `json:"status"`
	ActiveModuleID string               `json:"activeModuleId,omitempty"`
	ModuleOrder    []string             `json:"moduleOrder"`
	ModuleStates   []ModuleRuntimeState `json:"moduleStates,omitempty"`
	StartedAt      *time.Time           `json:"startedAt,omitempty"`
	EndedAt        *time.Time           `json:"endedAt,omitempty"`
	CreatedAt      time.Time            `json:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}
