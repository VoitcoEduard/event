package models

import "time"

// Question model
type QuizQuestion struct {
	ID                 string              `json:"id" bson:"id"`
	Type               string              `json:"type" bson:"type"` // "classic", "true_false", "multi_select", "simple_text"
	Text               string              `json:"text" bson:"text"`
	Answers            []string            `json:"answers" bson:"answers"` // 4 answer options (or 2 for True/False)
	CorrectIndex       int                 `json:"correctIndex" bson:"correctIndex"`
	CorrectIndices     []int               `json:"correctIndices" bson:"correctIndices"` // For multi_select
	CorrectValue       float64             `json:"correctValue" bson:"correctValue"`     // For range
	RangeConfig        RangeConfig         `json:"rangeConfig" bson:"rangeConfig"`       // For range
	ScoringConfig      ScoringConfig       `json:"scoringConfig" bson:"scoringConfig"`   // For range + choice + simple_text points
	TimeLimit          int                 `json:"timeLimit" bson:"timeLimit"`           // seconds (0 = no limit for simple_text)
	ExplainTime        int                 `json:"explainTime" bson:"explainTime"`       // seconds for explanation before question starts
	ImageID            *string             `json:"imageId" bson:"imageId"`
	ImageAlt           string              `json:"imageAlt" bson:"imageAlt"`
	ImageOptions       []ImageOption       `json:"imageOptions" bson:"imageOptions"`             // For correct_picture
	WrongPictureConfig *WrongPictureConfig `json:"wrongPictureConfig" bson:"wrongPictureConfig"` // For wrong_pictures
	CorrectNumber      int                 `json:"correctNumber" bson:"correctNumber"`           // For number
	// Simple Quiz (simple_text) fields. Points reuse ScoringConfig.MaxPoints.
	JudgeNotes        string `json:"judgeNotes,omitempty" bson:"judgeNotes,omitempty"`               // Private marking notes, judge only
	CorrectAnswerText string `json:"correctAnswerText,omitempty" bson:"correctAnswerText,omitempty"` // Public correct answer (revealed if allowed)
}

// RangeConfig stores range-specific configuration
type RangeConfig struct {
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

// WrongPictureTarget represents a modification location on an image
type WrongPictureTarget struct {
	ID       string  `json:"id" bson:"id"`             // Unique target ID
	Label    string  `json:"label" bson:"label"`       // Description (e.g., "Modern sneakers")
	XPercent float64 `json:"xPercent" bson:"xPercent"` // X position as percentage (0-100)
	YPercent float64 `json:"yPercent" bson:"yPercent"` // Y position as percentage (0-100)
	Radius   float64 `json:"radius" bson:"radius"`     // Hit radius as percentage
}

// WrongPictureConfig stores wrong_pictures specific configuration
type WrongPictureConfig struct {
	Targets          []WrongPictureTarget `json:"targets" bson:"targets"`
	TolerancePercent float64              `json:"tolerancePercent" bson:"tolerancePercent"` // For scoring
	MaxPoints        int                  `json:"maxPoints" bson:"maxPoints"`
	ShowTargetCount  bool                 `json:"showTargetCount" bson:"showTargetCount"` // Show answer count to players
}

const (
	QuestionTypeClassic        = "classic"
	QuestionTypeTrueFalse      = "true_false"
	QuestionTypeMultiSelect    = "multi_select"
	QuestionTypeRange          = "range"
	QuestionTypeCorrectPicture = "correct_picture"
	QuestionTypeWrongPictures  = "wrong_pictures"
	QuestionTypeNumber         = "number"
	QuestionTypeSimpleText     = "simple_text"
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
	// Simple Quiz settings. Points per question reuse ScoringConfig.MaxPoints,
	// TimeLimit 0 means no limit. JudgeNotes/CorrectAnswerText live per question.
	AllowAnswerReveal bool   `json:"allowAnswerReveal,omitempty" bson:"allowAnswerReveal,omitempty"`
	JudgeToken        string `json:"judgeToken,omitempty" bson:"judgeToken,omitempty"`
}

// Player in quiz
type Player struct {
	ID       string `json:"id" bson:"id"`
	Nickname string `json:"nickname" bson:"nickname"`
	Score    int    `json:"score" bson:"score"`
}

// PlayerSignal represents a player's answer signal with metadata
type PlayerSignal struct {
	PlayerID      string        `json:"playerId" bson:"playerId"`
	QuestionIndex int           `json:"questionIndex" bson:"questionIndex"`
	AnswerIndex   int           `json:"answerIndex" bson:"answerIndex"`     // For single choice
	AnswerIndices []int         `json:"answerIndices" bson:"answerIndices"` // For multi_select
	AnswerValue   float64       `json:"answerValue" bson:"answerValue"`     // For range
	ScoreEarned   int           `json:"scoreEarned" bson:"scoreEarned"`
	TimeSpent     int           `json:"timeSpent" bson:"timeSpent"` // ms
	Timestamp     int64         `json:"timestamp" bson:"timestamp"`
	Status        string        `json:"status" bson:"status"`   // "on_time", "grace_period", "missed"
	Guesses       []PlayerGuess `json:"guesses" bson:"guesses"` // For wrong_pictures type
}

// PlayerGuess represents a player's guess marker
type PlayerGuess struct {
	TargetID  string  `json:"targetId" bson:"targetId"`   // Which target they're guessing
	XPercent  float64 `json:"xPercent" bson:"xPercent"`   // X position as percentage
	YPercent  float64 `json:"yPercent" bson:"yPercent"`   // Y position as percentage
	Distance  float64 `json:"distance" bson:"distance"`   // Distance from actual (calculated client-side)
	IsCorrect bool    `json:"isCorrect" bson:"isCorrect"` // Whether within tolerance
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
	EventID       string             `json:"eventId" bson:"eventId"`
	ModuleID      string             `json:"moduleId" bson:"moduleId"`
	CurrentIndex  int                `json:"currentIndex" bson:"currentIndex"`
	State         string             `json:"state" bson:"state"` // "lobby", "question", "results", "finished"
	Players       map[string]*Player `json:"players" bson:"players"`
	PlayerSignals []PlayerSignal     `json:"playerSignals" bson:"playerSignals"` // All signals received
	AnswerCounts  map[int]int        `json:"answerCounts" bson:"answerCounts"`   // answerIndex -> count
	// AnsweredPlayerIDs tracks unique accepted answers for the active question.
	// It is runtime-only; PlayerSignals remains the complete result history.
	AnsweredPlayerIDs map[string]struct{} `json:"-" bson:"-"`
	QuestionWindows   []QuestionWindow    `json:"questionWindows" bson:"questionWindows"`
	StartTime         int64               `json:"startTime" bson:"startTime"` // Unix timestamp for when the question started

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
	PlayerID      string        `json:"playerId" bson:"playerId"`
	QuestionIndex int           `json:"questionIndex" bson:"questionIndex"`
	AnswerValue   float64       `json:"answerValue" bson:"answerValue"` // For range questions
	ScoreEarned   int           `json:"scoreEarned" bson:"scoreEarned"`
	TimeSpent     int           `json:"timeSpent" bson:"timeSpent"` // milliseconds
	Guesses       []PlayerGuess `json:"guesses" bson:"guesses"`     // For wrong_pictures type
}
