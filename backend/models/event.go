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

type CanvasElement struct {
	ID         string  `json:"id" bson:"id"`
	Type       string  `json:"type" bson:"type"` // "text", "image", "qr"
	X          float64 `json:"x" bson:"x"`
	Y          float64 `json:"y" bson:"y"`
	Width      float64 `json:"width" bson:"width"`
	Height     float64 `json:"height" bson:"height"`
	Content    string  `json:"content" bson:"content"` // text content or image URL/ID
	Color      string  `json:"color" bson:"color"`     // text color
	FontSize   int     `json:"fontSize" bson:"fontSize"`
	TextAlign  string  `json:"textAlign" bson:"textAlign"`
	FontWeight string  `json:"fontWeight" bson:"fontWeight"`
}

type Slide struct {
	ID              string          `json:"id" bson:"id"`
	Type            string          `json:"type" bson:"type"` // "welcome", "screensaver", "custom"
	Name            string          `json:"name" bson:"name"`
	BackgroundColor string          `json:"backgroundColor" bson:"backgroundColor"`
	BackgroundImage string          `json:"backgroundImage" bson:"backgroundImage"`
	Elements        []CanvasElement `json:"elements" bson:"elements"`
}

type ScreenState struct {
	CurrentView    string `json:"currentView" bson:"currentView"` // e.g., "screensaver", "welcome", "custom_text", "module", "slide"
	CustomText     string `json:"customText" bson:"customText"`
	ShowWelcomeQR  bool   `json:"showWelcomeQR" bson:"showWelcomeQR"`
	ActiveModuleID string `json:"activeModuleId" bson:"activeModuleId"` // ID of the quiz/form if view is "module"
	ActiveSlide    *Slide `json:"activeSlide" bson:"activeSlide"`
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
	Slides            []Slide            `json:"slides" bson:"slides"`
}
