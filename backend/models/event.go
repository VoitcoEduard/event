package models

import (
	"time"
)

type ModuleType string

const (
	ModuleTypeForm          ModuleType = "form"
	ModuleTypeQuiz          ModuleType = "quiz"
	ModuleTypeWrongPictures ModuleType = "wrong_pictures"
	ModuleTypeSimpleQuiz    ModuleType = "simple_quiz"
	ModuleTypeTeams         ModuleType = "teams"
)

type ModuleInputBinding struct {
	TargetKey  string `json:"targetKey"`
	SourceType string `json:"sourceType"`
	SourceKey  string `json:"sourceKey,omitempty"`
	Required   bool   `json:"required"`
}

type ModuleOutputDefinition struct {
	Key       string `json:"key"`
	Scope     string `json:"scope"`
	ValueType string `json:"valueType"`
	Persist   bool   `json:"persist"`
}

type ActivationRule struct {
	Type           string `json:"type"`
	SourceModuleID string `json:"sourceModuleId,omitempty"`
	Key            string `json:"key,omitempty"`
	Operator       string `json:"operator,omitempty"`
	Value          any    `json:"value,omitempty"`
}

type Module struct {
	ID              string                   `json:"id"`
	Type            ModuleType               `json:"type"`
	IsActive        bool                     `json:"isActive"`
	Name            string                   `json:"name"`
	Data            map[string]any           `json:"data"` // Generic data holder, now including timeline configuration
	Inputs          []ModuleInputBinding     `json:"inputs,omitempty"`
	Outputs         []ModuleOutputDefinition `json:"outputs,omitempty"`
	ActivationRules []ActivationRule         `json:"activationRules,omitempty"`
}

type Template struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ModuleType  ModuleType     `json:"moduleType"`
	Data        map[string]any `json:"data"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type ScreensaverConfig struct {
	ImageID  string `json:"imageId"`
	ImageURL string `json:"imageUrl"` // Computed field for display
}

type CanvasElement struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"` // "text", "image", "qr"
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Content    string  `json:"content"` // text content or image URL/ID
	Color      string  `json:"color"`   // text color
	FontSize   int     `json:"fontSize"`
	TextAlign  string  `json:"textAlign"`
	FontWeight string  `json:"fontWeight"`
}

type Slide struct {
	ID              string          `json:"id"`
	Type            string          `json:"type"` // "welcome", "screensaver", "custom"
	Name            string          `json:"name"`
	BackgroundColor string          `json:"backgroundColor"`
	BackgroundImage string          `json:"backgroundImage"`
	Elements        []CanvasElement `json:"elements"`
}

type ScreenState struct {
	CurrentView    string `json:"currentView"` // e.g., "screensaver", "welcome", "custom_text", "module", "slide"
	CustomText     string `json:"customText"`
	ShowWelcomeQR  bool   `json:"showWelcomeQR"`
	ActiveModuleID string `json:"activeModuleId"` // ID of the quiz/form if view is "module"
	ActiveSlide    *Slide `json:"activeSlide"`
}

type ParticipantPolicy struct {
	IdentityMode      string   `json:"identityMode"`
	RequiredFields    []string `json:"requiredFields"`
	DedupeStrategy    string   `json:"dedupeStrategy"`
	ReconnectStrategy string   `json:"reconnectStrategy"`
}

type EventScoringPolicy struct {
	Aggregation     string             `json:"aggregation"`
	ModuleWeights   map[string]float64 `json:"moduleWeights"`
	RankingScope    string             `json:"rankingScope"`
	TieBreakerRules []string           `json:"tieBreakerRules"`
}

type Event struct {
	ID                string              `json:"id"`
	Pin               string              `json:"pin,omitempty"`
	Name              string              `json:"name"`
	Description       string              `json:"description"`
	Language          string              `json:"language"`
	IsActive          bool                `json:"isActive"`
	CreatedAt         time.Time           `json:"createdAt"`
	Modules           []Module            `json:"modules"`
	ScreensaverConfig *ScreensaverConfig  `json:"screensaverConfig"`
	ScreenState       *ScreenState        `json:"screenState"`
	Slides            []Slide             `json:"slides"`
	ParticipantPolicy *ParticipantPolicy  `json:"participantPolicy,omitempty"`
	ScoringPolicy     *EventScoringPolicy `json:"scoringPolicy,omitempty"`
}
