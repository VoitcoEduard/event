package models

import "time"

type QuizQuestion struct {
	ID                 string              `json:"id"`
	Type               string              `json:"type"`
	Text               string              `json:"text"`
	Answers            []string            `json:"answers"`
	CorrectIndex       int                 `json:"correctIndex"`
	CorrectIndices     []int               `json:"correctIndices"`
	CorrectValue       float64             `json:"correctValue"`
	RangeConfig        RangeConfig         `json:"rangeConfig"`
	ScoringConfig      ScoringConfig       `json:"scoringConfig"`
	TimeLimit          int                 `json:"timeLimit"`
	ExplainTime        int                 `json:"explainTime"`
	ImageID            *string             `json:"imageId"`
	ImageAlt           string              `json:"imageAlt"`
	ImageOptions       []ImageOption       `json:"imageOptions"`
	WrongPictureConfig *WrongPictureConfig `json:"wrongPictureConfig"`
	CorrectNumber      int                 `json:"correctNumber"`
	JudgeNotes        string `json:"judgeNotes,omitempty"`
	CorrectAnswerText string `json:"correctAnswerText,omitempty"`
}

type RangeConfig struct {
	MinValue      float64 `json:"minValue"`
	MaxValue      float64 `json:"maxValue"`
	Step          float64 `json:"step"`
	DecimalPlaces int     `json:"decimalPlaces"`
	Unit          string  `json:"unit"`
}

type ScoringConfig struct {
	Mode           string  `json:"mode"`
	MaxPoints      int     `json:"maxPoints"`
	Tolerance      float64 `json:"tolerance"`
	PartialPercent float64 `json:"partialPercent"`

}

type WrongPictureTarget struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	XPercent float64 `json:"xPercent"`
	YPercent float64 `json:"yPercent"`
	Radius   float64 `json:"radius"`
}

type WrongPictureConfig struct {
	Targets          []WrongPictureTarget `json:"targets"`
	TolerancePercent float64              `json:"tolerancePercent"`
	MaxPoints        int                  `json:"maxPoints"`
	ShowTargetCount  bool                 `json:"showTargetCount"`
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

type ImageOption struct {
	ID      string `json:"id"`
	ImageID string `json:"imageId"`
	Label   string `json:"label"`
}

type QuizModuleData struct {
	Questions          []QuizQuestion `json:"questions"`
	BackgroundMusicURL string         `json:"backgroundMusicUrl"`
	BackgroundMusicID  string         `json:"backgroundMusicId"`
	AudioOutput        string         `json:"audioOutput"`
	AllowAnswerReveal bool   `json:"allowAnswerReveal,omitempty"`
	JudgeToken        string `json:"judgeToken,omitempty"`
}

type Player struct {
	ID                  string `json:"id"`
	Nickname            string `json:"nickname"`
	Score               int    `json:"score"`
	JoinedQuestionIndex int    `json:"joinedQuestionIndex,omitempty"`
}

type PlayerSignal struct {
	PlayerID      string        `json:"playerId"`
	QuestionIndex int           `json:"questionIndex"`
	AnswerIndex   int           `json:"answerIndex"`
	AnswerIndices []int         `json:"answerIndices"`
	AnswerValue   float64       `json:"answerValue"`
	ScoreEarned   int           `json:"scoreEarned"`
	TimeSpent     int           `json:"timeSpent"`
	Timestamp     int64         `json:"timestamp"`
	Status        string        `json:"status"`
	Guesses       []PlayerGuess `json:"guesses"`
}

type PlayerGuess struct {
	TargetID  string  `json:"targetId"`
	XPercent  float64 `json:"xPercent"`
	YPercent  float64 `json:"yPercent"`
	Distance  float64 `json:"distance"`
	IsCorrect bool    `json:"isCorrect"`
}

type QuestionWindow struct {
	QuestionIndex int64  `json:"questionIndex"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	GraceEndTime  int64  `json:"graceEndTime"`
	State         string `json:"state"`
}

type QuestionBatch struct {
	BatchIndex   int            `json:"batchIndex"`
	TotalBatches int            `json:"totalBatches"`
	Questions    []QuizQuestion `json:"questions"`
	NextBatchAt  int            `json:"nextBatchAt"`
	StartIndex   int            `json:"startIndex"`
	EndIndex     int            `json:"endIndex"`
}

type QuizSession struct {
	RunID         string             `json:"-"`
	EventID       string             `json:"eventId"`
	ModuleID      string             `json:"moduleId"`
	CurrentIndex  int                `json:"currentIndex"`
	State         string             `json:"state"`
	Players       map[string]*Player `json:"players"`
	PlayerSignals []PlayerSignal     `json:"playerSignals"`
	AnswerCounts  map[int]int        `json:"answerCounts"`
	AnsweredPlayerIDs   map[string]struct{} `json:"-"`
	Streaks             map[string]int      `json:"-"`
	WrongPictureAspects map[int]float64     `json:"-"`
	QuestionWindows     []QuestionWindow    `json:"questionWindows"`
	StartTime           int64               `json:"startTime"`

	Batches []QuestionBatch `json:"-"`

	Questions []QuizQuestion `json:"-"`
}

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
	AnswerValue   float64       `json:"answerValue"`
	ScoreEarned   int           `json:"scoreEarned"`
	TimeSpent     int           `json:"timeSpent"`
	Guesses       []PlayerGuess `json:"guesses"`
}
