package models

import "time"

// Question model
type QuizQuestion struct {
	ID                 string              `json:"id"`
	Type               string              `json:"type"` // "classic", "true_false", "multi_select", "simple_text"
	Text               string              `json:"text"`
	Answers            []string            `json:"answers"` // 4 answer options (or 2 for True/False)
	CorrectIndex       int                 `json:"correctIndex"`
	CorrectIndices     []int               `json:"correctIndices"` // For multi_select
	CorrectValue       float64             `json:"correctValue"`   // For range
	RangeConfig        RangeConfig         `json:"rangeConfig"`    // For range
	ScoringConfig      ScoringConfig       `json:"scoringConfig"`  // For range + choice + simple_text points
	TimeLimit          int                 `json:"timeLimit"`      // seconds (0 = no limit for simple_text)
	ExplainTime        int                 `json:"explainTime"`    // seconds for explanation before question starts
	ImageID            *string             `json:"imageId"`
	ImageAlt           string              `json:"imageAlt"`
	ImageOptions       []ImageOption       `json:"imageOptions"`       // For correct_picture
	WrongPictureConfig *WrongPictureConfig `json:"wrongPictureConfig"` // For wrong_pictures
	CorrectNumber      int                 `json:"correctNumber"`      // For number
	// Simple Quiz (simple_text) fields. Points reuse ScoringConfig.MaxPoints.
	JudgeNotes        string `json:"judgeNotes,omitempty"`        // Private marking notes, judge only
	CorrectAnswerText string `json:"correctAnswerText,omitempty"` // Public correct answer (revealed if allowed)
}

// RangeConfig stores range-specific configuration
type RangeConfig struct {
	MinValue      float64 `json:"minValue"`
	MaxValue      float64 `json:"maxValue"`
	Step          float64 `json:"step"`
	DecimalPlaces int     `json:"decimalPlaces"`
	Unit          string  `json:"unit"`
}

// ScoringConfig stores scoring rules
type ScoringConfig struct {
	Mode           string  `json:"mode"`           // "symmetric" or "asymmetric"
	MaxPoints      int     `json:"maxPoints"`      // Points for exact answer
	Tolerance      float64 `json:"tolerance"`      // Distance for partial credit
	PartialPercent float64 `json:"partialPercent"` // % of max points at tolerance edge

}

// WrongPictureTarget represents a modification location on an image
type WrongPictureTarget struct {
	ID       string  `json:"id"`       // Unique target ID
	Label    string  `json:"label"`    // Description (e.g., "Modern sneakers")
	XPercent float64 `json:"xPercent"` // X position as percentage (0-100)
	YPercent float64 `json:"yPercent"` // Y position as percentage (0-100)
	Radius   float64 `json:"radius"`   // Hit radius as percentage
}

// WrongPictureConfig stores wrong_pictures specific configuration
type WrongPictureConfig struct {
	Targets          []WrongPictureTarget `json:"targets"`
	TolerancePercent float64              `json:"tolerancePercent"` // For scoring
	MaxPoints        int                  `json:"maxPoints"`
	ShowTargetCount  bool                 `json:"showTargetCount"` // Show answer count to players
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
	ID      string `json:"id"`      // Unique option ID
	ImageID string `json:"imageId"` // Reference to media library
	Label   string `json:"label"`   // Optional label for admin
}

// Quiz module data - stored in Module.Data
type QuizModuleData struct {
	Questions          []QuizQuestion `json:"questions"`
	BackgroundMusicURL string         `json:"backgroundMusicUrl"`
	BackgroundMusicID  string         `json:"backgroundMusicId"`
	AudioOutput        string         `json:"audioOutput"` // "admin", "bigscreen", "both"
	// Simple Quiz settings. Points per question reuse ScoringConfig.MaxPoints,
	// TimeLimit 0 means no limit. JudgeNotes/CorrectAnswerText live per question.
	AllowAnswerReveal bool   `json:"allowAnswerReveal,omitempty"`
	JudgeToken        string `json:"judgeToken,omitempty"`
}

// Player in quiz
type Player struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Score    int    `json:"score"`
}

// PlayerSignal represents a player's answer signal with metadata
type PlayerSignal struct {
	PlayerID      string        `json:"playerId"`
	QuestionIndex int           `json:"questionIndex"`
	AnswerIndex   int           `json:"answerIndex"`   // For single choice
	AnswerIndices []int         `json:"answerIndices"` // For multi_select
	AnswerValue   float64       `json:"answerValue"`   // For range
	ScoreEarned   int           `json:"scoreEarned"`
	TimeSpent     int           `json:"timeSpent"` // ms
	Timestamp     int64         `json:"timestamp"`
	Status        string        `json:"status"`  // "on_time", "grace_period", "missed"
	Guesses       []PlayerGuess `json:"guesses"` // For wrong_pictures type
}

// PlayerGuess represents a player's guess marker
type PlayerGuess struct {
	TargetID  string  `json:"targetId"`  // Which target they're guessing
	XPercent  float64 `json:"xPercent"`  // X position as percentage
	YPercent  float64 `json:"yPercent"`  // Y position as percentage
	Distance  float64 `json:"distance"`  // Distance from actual (calculated client-side)
	IsCorrect bool    `json:"isCorrect"` // Whether within tolerance
}

// QuestionWindow tracks the timing and acceptance window for a question
type QuestionWindow struct {
	QuestionIndex int64  `json:"questionIndex"`
	StartTime     int64  `json:"startTime"`    // Unix timestamp
	EndTime       int64  `json:"endTime"`      // When timer expires
	GraceEndTime  int64  `json:"graceEndTime"` // EndTime + GracePeriod
	State         string `json:"state"`        // "active", "grace", "closed"
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
	RunID         string             `json:"-"`
	EventID       string             `json:"eventId"`
	ModuleID      string             `json:"moduleId"`
	CurrentIndex  int                `json:"currentIndex"`
	State         string             `json:"state"` // "lobby", "question", "results", "finished"
	Players       map[string]*Player `json:"players"`
	PlayerSignals []PlayerSignal     `json:"playerSignals"` // All signals received
	AnswerCounts  map[int]int        `json:"answerCounts"`  // answerIndex -> count
	// AnsweredPlayerIDs tracks unique accepted answers for the active question.
	// It is runtime-only; PlayerSignals remains the complete result history.
	AnsweredPlayerIDs map[string]struct{} `json:"-"`
	QuestionWindows   []QuestionWindow    `json:"questionWindows"`
	StartTime         int64               `json:"startTime"` // Unix timestamp for when the question started

	// Batch management
	Batches []QuestionBatch `json:"-"`

	// Server-side only (not exported to JSON generally)
	Questions []QuizQuestion `json:"-"`
}

// QuizResult document (new "quiz_results" collection)
type QuizResult struct {
	RunID         string         `json:"-"`
	EventID       string         `json:"eventId"`
	ModuleID      string         `json:"moduleId"`
	FinishedAt    time.Time      `json:"finishedAt"`
	Players       []PlayerResult `json:"players"`
	PlayerSignals []SignalResult `json:"playerSignals"`
}

type PlayerResult struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Score    int    `json:"score"`
}

type SignalResult struct {
	PlayerID      string        `json:"playerId"`
	QuestionIndex int           `json:"questionIndex"`
	AnswerValue   float64       `json:"answerValue"` // For range questions
	ScoreEarned   int           `json:"scoreEarned"`
	TimeSpent     int           `json:"timeSpent"` // milliseconds
	Guesses       []PlayerGuess `json:"guesses"`   // For wrong_pictures type
}
