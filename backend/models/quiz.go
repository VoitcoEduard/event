package models

// Question model
type QuizQuestion struct {
	ID           string   `json:"id" bson:"id"`
	Text         string   `json:"text" bson:"text"`
	Answers      []string `json:"answers" bson:"answers"` // 4 answer options
	CorrectIndex int      `json:"correctIndex" bson:"correctIndex"`
	TimeLimit    int      `json:"timeLimit" bson:"timeLimit"` // seconds
}

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

// Quiz state - potentially transient or persisted separately
type QuizSession struct {
	EventID      string             `json:"eventId" bson:"eventId"`
	ModuleID     string             `json:"moduleId" bson:"moduleId"`
	CurrentIndex int                `json:"currentIndex" bson:"currentIndex"`
	State        string             `json:"state" bson:"state"` // "lobby", "question", "results", "finished"
	Players      map[string]*Player `json:"players" bson:"players"`
	Answers      map[string]int     `json:"answers" bson:"answers"`     // playerID -> answerIndex (int)
	StartTime    int64              `json:"startTime" bson:"startTime"` // Unix timestamp for when the question started

	// Server-side only (not exported to JSON generally, but maybe okay for now?
	// Actually, we shouldn't send answers to players.
	// But for simplicity of state sync, we might need to be careful.
	// Let's add them but maybe control JSON output or just trust clients not to cheat for this MVP.)
	// Better: Don't export Questions to JSON if we want to hide them.
	// But Host needs them.
	// We can use a separate struct or just be careful.
	// For MVP, sending questions to everyone is "okay" but ideally we filter.
	// Let's store them here for server logic.
	Questions []QuizQuestion `json:"-" bson:"-"`
	// Note: Plan said map[string]string for Answers, but key is playerID, value is answerIndex which is int?
	// Wait, plan said `Answers map[string]string`. Let's stick to plan but verify logic.
	// If answerIndex is int, map[string]int is better.
	// However, if we want to store arbitrary answers for other question types later, string might be better.
	// Given the constraints (4 options, index 0-3), int is cleaner.
	// BUT the plan explicitly says `Answers map[string]string // playerID -> answerIndex`.
	// Wait, "answerIndex" implies a number. Maybe string representation of index "0", "1"?
	// Or maybe the plan had a typo.
	// Let's use `map[string]int` for type safety if it's an index, or `map[string]string` if it might be text.
	// The plan says `CorrectIndex int`. So the answer is an index.
	// I will use `map[string]int` for better type safety, deviating slightly from plan for correctness unless I see a reason not to.
	// Actually, looking at `QuizQuestion.CorrectIndex int`, `int` makes strict sense.
	// Wait, I'll stick to `map[string]int` for `Answers`, assuming `playerID` -> `answerIndex`.
}
