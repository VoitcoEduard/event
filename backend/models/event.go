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

type ScreensaverConfig struct {
	ImageID  string `json:"imageId" bson:"imageId"`
	ImageURL string `json:"imageUrl" bson:"imageUrl"` // Computed field for display
}

type ScreenState struct {
	CurrentView    string `json:"currentView" bson:"currentView"` // e.g., "screensaver", "welcome", "custom_text", "module"
	CustomText     string `json:"customText" bson:"customText"`
	ShowWelcomeQR  bool   `json:"showWelcomeQR" bson:"showWelcomeQR"`
	ActiveModuleID string `json:"activeModuleId" bson:"activeModuleId"` // ID of the quiz/form if view is "module"
}

type Event struct {
	ID                string             `json:"id" bson:"_id"`
	Name              string             `json:"name" bson:"name"`
	Description       string             `json:"description" bson:"description"`
	IsActive          bool               `json:"isActive" bson:"isActive"`
	CreatedAt         time.Time          `json:"createdAt" bson:"createdAt"`
	Modules           []Module           `json:"modules" bson:"modules"`
	ScreensaverConfig *ScreensaverConfig `json:"screensaverConfig" bson:"screensaverConfig"`
	ScreenState       *ScreenState       `json:"screenState" bson:"screenState"`
}
