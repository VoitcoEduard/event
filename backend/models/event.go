package models

import (
	"time"
)

type ModuleType string

const (
	ModuleTypeForm ModuleType = "form"
	ModuleTypeQuiz ModuleType = "quiz"
	// Add other module types here

)

type Module struct {
	ID       string         `json:"id" bson:"id"`
	Type     ModuleType     `json:"type" bson:"type"`
	IsActive bool           `json:"isActive" bson:"isActive"`
	Name     string         `json:"name" bson:"name"`
	Data     map[string]any `json:"data" bson:"data"` // Generic data holder, now including timeline configuration
}

type Template struct {
	ID          string         `json:"id" bson:"_id"`
	Name        string         `json:"name" bson:"name"`
	Description string         `json:"description" bson:"description"`
	ModuleType  ModuleType     `json:"moduleType" bson:"moduleType"`
	Data        map[string]any `json:"data" bson:"data"`
	CreatedAt   time.Time      `json:"createdAt" bson:"createdAt"`
}

type Event struct {
	ID          string    `json:"id" bson:"_id"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	IsActive    bool      `json:"isActive" bson:"isActive"`
	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
	Modules     []Module  `json:"modules" bson:"modules"`
}
