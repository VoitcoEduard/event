package models

import (
	"time"
)

type ModuleType string

const (
	ModuleTypeForm          ModuleType = "form"
	ModuleTypeQuiz          ModuleType = "quiz"
	ModuleTypeWrongPictures ModuleType = "wrong_pictures"
)

type ModuleInputBinding struct {
	TargetKey  string `json:"targetKey" bson:"targetKey"`
	SourceType string `json:"sourceType" bson:"sourceType"`
	SourceKey  string `json:"sourceKey,omitempty" bson:"sourceKey,omitempty"`
	Required   bool   `json:"required" bson:"required"`
}

type ModuleOutputDefinition struct {
	Key       string `json:"key" bson:"key"`
	Scope     string `json:"scope" bson:"scope"`
	ValueType string `json:"valueType" bson:"valueType"`
	Persist   bool   `json:"persist" bson:"persist"`
}

type ActivationRule struct {
	Type           string `json:"type" bson:"type"`
	SourceModuleID string `json:"sourceModuleId,omitempty" bson:"sourceModuleId,omitempty"`
	Key            string `json:"key,omitempty" bson:"key,omitempty"`
	Operator       string `json:"operator,omitempty" bson:"operator,omitempty"`
	Value          any    `json:"value,omitempty" bson:"value,omitempty"`
}

type Module struct {
	ID              string                   `json:"id" bson:"id"`
	Type            ModuleType               `json:"type" bson:"type"`
	IsActive        bool                     `json:"isActive" bson:"isActive"`
	Name            string                   `json:"name" bson:"name"`
	Data            map[string]any           `json:"data" bson:"data"` // Generic data holder, now including timeline configuration
	Inputs          []ModuleInputBinding     `json:"inputs,omitempty" bson:"inputs,omitempty"`
	Outputs         []ModuleOutputDefinition `json:"outputs,omitempty" bson:"outputs,omitempty"`
	ActivationRules []ActivationRule         `json:"activationRules,omitempty" bson:"activationRules,omitempty"`
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

type ParticipantPolicy struct {
	IdentityMode      string   `json:"identityMode" bson:"identityMode"`
	RequiredFields    []string `json:"requiredFields" bson:"requiredFields"`
	DedupeStrategy    string   `json:"dedupeStrategy" bson:"dedupeStrategy"`
	ReconnectStrategy string   `json:"reconnectStrategy" bson:"reconnectStrategy"`
}

type EventScoringPolicy struct {
	Aggregation     string             `json:"aggregation" bson:"aggregation"`
	ModuleWeights   map[string]float64 `json:"moduleWeights" bson:"moduleWeights"`
	RankingScope    string             `json:"rankingScope" bson:"rankingScope"`
	TieBreakerRules []string           `json:"tieBreakerRules" bson:"tieBreakerRules"`
}

type Event struct {
	ID                string               `json:"id" bson:"_id"`
	Name              string               `json:"name" bson:"name"`
	Description       string               `json:"description" bson:"description"`
	Language          string               `json:"language" bson:"language"`
	IsActive          bool                 `json:"isActive" bson:"isActive"`
	CreatedAt         time.Time            `json:"createdAt" bson:"createdAt"`
	Modules           []Module             `json:"modules" bson:"modules"`
	ScreensaverConfig *ScreensaverConfig   `json:"screensaverConfig" bson:"screensaverConfig"`
	ScreenState       *ScreenState         `json:"screenState" bson:"screenState"`
	Slides            []Slide              `json:"slides" bson:"slides"`
	ParticipantPolicy *ParticipantPolicy   `json:"participantPolicy,omitempty" bson:"participantPolicy,omitempty"`
	ScoringPolicy     *EventScoringPolicy  `json:"scoringPolicy,omitempty" bson:"scoringPolicy,omitempty"`
}
