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
	ID             string               `json:"id" bson:"_id"`
	EventID        string               `json:"eventId" bson:"eventId"`
	Status         EventSessionStatus   `json:"status" bson:"status"`
	ActiveModuleID string               `json:"activeModuleId,omitempty" bson:"activeModuleId,omitempty"`
	ModuleOrder    []string             `json:"moduleOrder" bson:"moduleOrder"`
	ModuleStates   []ModuleRuntimeState `json:"moduleStates,omitempty" bson:"moduleStates,omitempty"`
	StartedAt      *time.Time           `json:"startedAt,omitempty" bson:"startedAt,omitempty"`
	EndedAt        *time.Time           `json:"endedAt,omitempty" bson:"endedAt,omitempty"`
	CreatedAt      time.Time            `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time            `json:"updatedAt" bson:"updatedAt"`
}
