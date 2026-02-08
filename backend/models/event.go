package models

import (
	"time"
)

type ModuleType string

const (
	ModuleTypeForm ModuleType = "form"
	// Add other module types here
)

type ModuleStatus string

const (
	ModuleStatusPending ModuleStatus = "pending"
	ModuleStatusActive  ModuleStatus = "active"
	ModuleStatusEnded   ModuleStatus = "ended"
)

type FormQuestion struct {
	ID   string `json:"id" bson:"id"`
	Text string `json:"text" bson:"text"`
	Type string `json:"type" bson:"type"` // e.g., "text", "choice"
	// Add options for choice type if needed
}

type Module struct {
	ID        string         `json:"id" bson:"id"`
	Type      ModuleType     `json:"type" bson:"type"`
	Status    ModuleStatus   `json:"status" bson:"status"`
	Name      string         `json:"name" bson:"name"`
	Data      map[string]any `json:"data" bson:"data"`                               // Generic data holder
	Questions []FormQuestion `json:"questions,omitempty" bson:"questions,omitempty"` // Specific to Form module for now, can be generic later
}

type Event struct {
	ID          string    `json:"id" bson:"_id"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	IsActive    bool      `json:"isActive" bson:"isActive"`
	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
	Modules     []Module  `json:"modules" bson:"modules"`
}
