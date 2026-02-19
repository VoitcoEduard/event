package models

import "time"

// Question model
type QuizQuestion struct {
	ID             string        `json:"id" bson:"id"`
	Type           string        `json:"type" bson:"type"` // "multiple_choice", "true_false", "multi_select"
	Text           string        `json:"text" bson:"text"`
	Answers        []string      `json:"answers" bson:"answers"` // 4 answer options (or 2 for True/False)
	CorrectIndex   int           `json:"correctIndex" bson:"correctIndex"`
	CorrectIndices []int         `json:"correctIndices" bson:"correctIndices"` // For multi_select
	CorrectValue   float64       `json:"correctValue" bson:"correctValue"`     // For slider
	SliderConfig   SliderConfig  `json:"sliderConfig" bson:"sliderConfig"`     // For slider
	ScoringConfig  ScoringConfig `json:"scoringConfig" bson:"scoringConfig"`   // For slider
	TimeLimit      int           `json:"timeLimit" bson:"timeLimit"`           // seconds
	ImageID        *string       `json:"imageId" bson:"imageId"`
	ImageAlt       string        `json:"imageAlt" bson:"imageAlt"`
	ImageOptions   []ImageOption `json:"imageOptions" bson:"imageOptions"` // For correct_picture
}

// SliderConfig stores slider-specific configuration
type SliderConfig struct {
	MinValue      float64 `json:"minValue" bson:"minValue"`
	MaxValue      float64 `json:"maxValue" bson:"maxValue"`
	Step          float64 `json:"step" bson:"step"`
	DecimalPlaces int     `json:"decimalPlaces" bson:"decimalPlaces"`
	Unit          string  `json:"unit" bson:"unit"`
}

// ScoringConfig stores scoring rules
type ScoringConfig struct {
	Mode           string  `json:"mode" bson:"mode"`                     // "symmetric" or "asymmetric"
	MaxPoints      int     `json:"maxPoints" bson:"maxPoints"`           // Points for exact answer
	Tolerance      float64 `json:"tolerance" bson:"tolerance"`           // Distance for partial credit
	PartialPercent float64 `json:"partialPercent" bson:"partialPercent"` // % of max points at tolerance edge

}

const (
	QuestionTypeMultipleChoice = "multiple_choice"
	QuestionTypeTrueFalse      = "true_false"
	QuestionTypeMultiSelect    = "multi_select"
	QuestionTypeSlider         = "slider"
	QuestionTypeCorrectPicture = "correct_picture"
)

// ImageOption struct for correct_picture question type
type ImageOption struct {
	ID      string `json:"id" bson:"id"`           // Unique option ID
	ImageID string `json:"imageId" bson:"imageId"` // Reference to media library
	Label   string `json:"label" bson:"label"`     // Optional label for admin
}

// Quiz module data - stored in Module.Data
type QuizModuleData struct {
	Questions          []QuizQuestion `json:"questions" bson:"questions"`
	BackgroundMusicURL string         `json:"backgroundMusicUrl" bson:"backgroundMusicUrl"`
	BackgroundMusicID  string         `json:"backgroundMusicId" bson:"backgroundMusicId"`
	AudioOutput        string         `json:"audioOutput" bson:"audioOutput"` // "admin", "bigscreen", "both"
}

// Player in quiz
type Player struct {
	ID       string `json:"id" bson:"id"`
	Nickname string `json:"nickname" bson:"nickname"`
	Score    int    `json:"score" bson:"score"`
}

// PlayerSignal represents a player's answer signal with metadata
type PlayerSignal struct {
	PlayerID      string  `json:"playerId" bson:"playerId"`
	QuestionIndex int     `json:"questionIndex" bson:"questionIndex"`
	AnswerIndex   int     `json:"answerIndex" bson:"answerIndex"`     // For single choice
	AnswerIndices []int   `json:"answerIndices" bson:"answerIndices"` // For multi_select
	AnswerValue   float64 `json:"answerValue" bson:"answerValue"`     // For slider
	ScoreEarned   int     `json:"scoreEarned" bson:"scoreEarned"`
	TimeSpent     int     `json:"timeSpent" bson:"timeSpent"` // ms
	Timestamp     int64   `json:"timestamp" bson:"timestamp"`
	Status        string  `json:"status" bson:"status"` // "on_time", "grace_period", "missed"
}

// QuestionWindow tracks the timing and acceptance window for a question
type QuestionWindow struct {
	QuestionIndex int64  `json:"questionIndex" bson:"questionIndex"`
	StartTime     int64  `json:"startTime" bson:"startTime"`       // Unix timestamp
	EndTime       int64  `json:"endTime" bson:"endTime"`           // When timer expires
	GraceEndTime  int64  `json:"graceEndTime" bson:"graceEndTime"` // EndTime + GracePeriod
	State         string `json:"state" bson:"state"`               // "active", "grace", "closed"
}

// QuestionBatch represents a chunk of questions to be sent to clients
type QuestionBatch struct {
	BatchIndex   int            `json:"batchIndex"`
	TotalBatches int            `json:"totalBatches"`
	Questions    []QuizQuestion `json:"questions"`
	NextBatchAt  int            `json:"nextBatchAt"` // Question index when to request next batch
	StartIndex   int            `json:"startIndex"`  // Starting question index in this batch
	EndIndex     int            `json:"endIndex"`    // Ending question index in this batch
}

// Quiz state - potentially transient or persisted separately
type QuizSession struct {
	EventID         string             `json:"eventId" bson:"eventId"`
	ModuleID        string             `json:"moduleId" bson:"moduleId"`
	CurrentIndex    int                `json:"currentIndex" bson:"currentIndex"`
	State           string             `json:"state" bson:"state"` // "lobby", "question", "results", "finished"
	Players         map[string]*Player `json:"players" bson:"players"`
	PlayerSignals   []PlayerSignal     `json:"playerSignals" bson:"playerSignals"` // All signals received
	AnswerCounts    map[int]int        `json:"answerCounts" bson:"answerCounts"`   // answerIndex -> count
	QuestionWindows []QuestionWindow   `json:"questionWindows" bson:"questionWindows"`
	StartTime       int64              `json:"startTime" bson:"startTime"` // Unix timestamp for when the question started

	// Batch management
	Batches []QuestionBatch `json:"-" bson:"-"`

	// Server-side only (not exported to JSON generally)
	Questions []QuizQuestion `json:"-" bson:"-"`
}

// QuizResult document (new "quiz_results" collection)
type QuizResult struct {
	EventID       string         `json:"eventId" bson:"eventId"`
	ModuleID      string         `json:"moduleId" bson:"moduleId"`
	FinishedAt    time.Time      `json:"finishedAt" bson:"finishedAt"`
	Players       []PlayerResult `json:"players" bson:"players"`
	PlayerSignals []SignalResult `json:"playerSignals" bson:"playerSignals"`
}

type PlayerResult struct {
	ID       string `json:"id" bson:"id"`
	Nickname string `json:"nickname" bson:"nickname"`
	Score    int    `json:"score" bson:"score"`
}

type SignalResult struct {
	PlayerID      string  `json:"playerId" bson:"playerId"`
	QuestionIndex int     `json:"questionIndex" bson:"questionIndex"`
	AnswerValue   float64 `json:"answerValue" bson:"answerValue"` // For slider questions
	ScoreEarned   int     `json:"scoreEarned" bson:"scoreEarned"`
	TimeSpent     int     `json:"timeSpent" bson:"timeSpent"` // milliseconds
}
