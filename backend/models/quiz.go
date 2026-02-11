package models

// Question model
type QuizQuestion struct {
	ID           string   `json:"id" bson:"id"`
	Type         string   `json:"type" bson:"type"` // "multiple_choice" or "true_false"
	Text         string   `json:"text" bson:"text"`
	Answers      []string `json:"answers" bson:"answers"` // 4 answer options (or 2 for True/False)
	CorrectIndex int      `json:"correctIndex" bson:"correctIndex"`
	TimeLimit    int      `json:"timeLimit" bson:"timeLimit"` // seconds
}

const (
	QuestionTypeMultipleChoice = "multiple_choice"
	QuestionTypeTrueFalse      = "true_false"
)

// Quiz module data - stored in Module.Data
type QuizModuleData struct {
	Questions []QuizQuestion `json:"questions" bson:"questions"`
}

// Player in quiz
type Player struct {
	ID       string `json:"id" bson:"id"`
	Nickname string `json:"nickname" bson:"nickname"`
	Score    int    `json:"score" bson:"score"`
}

// PlayerSignal represents a player's answer signal with metadata
type PlayerSignal struct {
	PlayerID      string `json:"playerId" bson:"playerId"`
	QuestionIndex int    `json:"questionIndex" bson:"questionIndex"`
	AnswerIndex   int    `json:"answerIndex" bson:"answerIndex"`
	ScoreEarned   int    `json:"scoreEarned" bson:"scoreEarned"`
	TimeSpent     int    `json:"timeSpent" bson:"timeSpent"` // ms
	Timestamp     int64  `json:"timestamp" bson:"timestamp"`
	Status        string `json:"status" bson:"status"` // "on_time", "grace_period", "missed"
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
