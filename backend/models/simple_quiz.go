package models

import "time"

// Simple Quiz: live, manually graded text-answer quiz.
// The template lives in Module.Data (Questions + AllowAnswerReveal + JudgeToken).
// On launch the server snapshots questions into SimpleQuizSession so later
// template edits do not affect the running session.

const (
	SimpleQuizStateLobby             = "lobby"
	SimpleQuizStateQuestionOpen      = "question_open"
	SimpleQuizStateAnswersLocked     = "answers_locked"
	SimpleQuizStateBetweenQuestions  = "between_questions"
	SimpleQuizStateAnsweringFinished = "answering_finished"
	SimpleQuizStateWaitingGrading    = "waiting_grading"
	SimpleQuizStateResultsPublished  = "results_published"
	SimpleQuizStateCompleted         = "completed"
	SimpleQuizStatePaused            = "paused"
)

const (
	SimpleQuizVerdictRight       = "right"
	SimpleQuizVerdictWrong       = "wrong"
	SimpleQuizVerdictReviewLater = "review_later"
	SimpleQuizVerdictMissing     = "missing"
)

const SimpleQuizMaxAnswerChars = 2000

// SimpleQuizQuestionSnapshot is a copy of the template question at launch time.
type SimpleQuizQuestionSnapshot struct {
	ID                string  `json:"id" bson:"id"`
	Text              string  `json:"text" bson:"text"`
	ImageID           *string `json:"imageId,omitempty" bson:"imageId,omitempty"`
	ImageAlt          string  `json:"imageAlt,omitempty" bson:"imageAlt,omitempty"`
	Points            int     `json:"points" bson:"points"`
	TimeLimit         int     `json:"timeLimit" bson:"timeLimit"` // 0 = no limit
	JudgeNotes        string  `json:"judgeNotes,omitempty" bson:"judgeNotes,omitempty"`
	CorrectAnswerText string  `json:"correctAnswerText,omitempty" bson:"correctAnswerText,omitempty"`
}

// SimpleQuizPlayer joined with a nickname. Secret is the private reconnect identity.
type SimpleQuizPlayer struct {
	ID        string    `json:"id" bson:"id"`
	Nickname  string    `json:"nickname" bson:"nickname"`
	Secret    string    `json:"secret,omitempty" bson:"secret,omitempty"`
	Score     int       `json:"score" bson:"score"`
	JoinedAt  time.Time `json:"joinedAt" bson:"joinedAt"`
	Connected bool      `json:"connected" bson:"connected"`
}

// SimpleQuizAnswer is one locked text answer per player per question.
type SimpleQuizAnswer struct {
	ID            string    `json:"id" bson:"id"` // client-generated UUID for idempotency
	PlayerID      string    `json:"playerId" bson:"playerId"`
	QuestionIndex int       `json:"questionIndex" bson:"questionIndex"`
	QuestionID    string    `json:"questionId" bson:"questionId"`
	Text          string    `json:"text" bson:"text"`
	CreatedAt     time.Time `json:"createdAt" bson:"createdAt"`
}

// SimpleQuizGradeHistoryEntry tracks every grading change for undo/audit.
type SimpleQuizGradeHistoryEntry struct {
	Verdict string    `json:"verdict" bson:"verdict"`
	By      string    `json:"by" bson:"by"` // judge session label, e.g. "judge"
	At      time.Time `json:"at" bson:"at"`
}

// SimpleQuizGrade is the current verdict for one answer.
type SimpleQuizGrade struct {
	AnswerID  string                        `json:"answerId" bson:"answerId"`
	Verdict   string                        `json:"verdict" bson:"verdict"` // right | wrong | review_later
	UpdatedAt time.Time                     `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy string                        `json:"updatedBy" bson:"updatedBy"`
	History   []SimpleQuizGradeHistoryEntry `json:"history" bson:"history"`
}

// SimpleQuizPerQuestionResult is one cell of working/published results.
type SimpleQuizPerQuestionResult struct {
	QuestionIndex int    `json:"questionIndex" bson:"questionIndex"`
	Verdict       string `json:"verdict" bson:"verdict"` // right | wrong | missing
	PointsEarned  int    `json:"pointsEarned" bson:"pointsEarned"`
}

// SimpleQuizPlayerResult is a leaderboard row.
type SimpleQuizPlayerResult struct {
	PlayerID    string                        `json:"playerId" bson:"playerId"`
	Nickname    string                        `json:"nickname" bson:"nickname"`
	TotalScore  int                           `json:"totalScore" bson:"totalScore"`
	PerQuestion []SimpleQuizPerQuestionResult `json:"perQuestion" bson:"perQuestion"`
}

// SimpleQuizPublishedSnapshot is the stable version visible to players/projector.
type SimpleQuizPublishedSnapshot struct {
	Version     int                      `json:"version" bson:"version"`
	PublishedAt time.Time                `json:"publishedAt" bson:"publishedAt"`
	Players     []SimpleQuizPlayerResult `json:"players" bson:"players"`
}

// SimpleQuizSession is the live session. Persisted in simple_quiz_sessions,
// cached in-memory in the hub. Questions is the launch-time snapshot.
type SimpleQuizSession struct {
	EventID       string                       `json:"eventId" bson:"eventId"`
	ModuleID      string                       `json:"moduleId" bson:"moduleId"`
	Title         string                       `json:"title,omitempty" bson:"title,omitempty"`
	Status        string                       `json:"status" bson:"status"`
	CurrentIndex  int                          `json:"currentIndex" bson:"currentIndex"`
	Questions     []SimpleQuizQuestionSnapshot `json:"questions" bson:"questions"`
	Players       map[string]*SimpleQuizPlayer `json:"players" bson:"players"`
	Answers       []SimpleQuizAnswer           `json:"answers" bson:"answers"`
	Grades        map[string]*SimpleQuizGrade  `json:"grades" bson:"grades"`
	Revealed      map[int]bool                 `json:"revealed" bson:"revealed"` // questionIndex -> revealed
	Published     *SimpleQuizPublishedSnapshot `json:"published,omitempty" bson:"published,omitempty"`
	PublishedVer  int                          `json:"publishedVersion" bson:"publishedVersion"`
	AllowReveal   bool                         `json:"allowReveal" bson:"allowReveal"`
	DeadlineUnix  int64                        `json:"deadlineUnix,omitempty" bson:"deadlineUnix,omitempty"` // 0 = no limit
	QuestionOpenedAt time.Time                 `json:"questionOpenedAt,omitempty" bson:"questionOpenedAt,omitempty"`
	PausedFrom    string                       `json:"pausedFrom,omitempty" bson:"pausedFrom,omitempty"`
	CreatedAt     time.Time                    `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time                    `json:"updatedAt" bson:"updatedAt"`
}

// SnapshotSimpleQuizQuestion copies template fields into a snapshot.
// Points reuse ScoringConfig.MaxPoints for consistency with other types.
func SnapshotSimpleQuizQuestion(q QuizQuestion) SimpleQuizQuestionSnapshot {
	points := q.ScoringConfig.MaxPoints
	return SimpleQuizQuestionSnapshot{
		ID:                q.ID,
		Text:              q.Text,
		ImageID:           q.ImageID,
		ImageAlt:          q.ImageAlt,
		Points:            points,
		TimeLimit:         q.TimeLimit,
		JudgeNotes:        q.JudgeNotes,
		CorrectAnswerText: q.CorrectAnswerText,
	}
}

// EffectiveSimpleQuizPoints returns the point value for a snapshot question.
func EffectiveSimpleQuizPoints(q SimpleQuizQuestionSnapshot) int {
	return q.Points
}

// GradingProgress counts received/graded/ungraded/review-later per question.
type SimpleQuizGradingProgress struct {
	QuestionIndex int `json:"questionIndex"`
	Received      int `json:"received"`
	Graded        int `json:"graded"`
	Ungraded      int `json:"ungraded"`
	ReviewLater   int `json:"reviewLater"`
}

// ComputeWorkingResults builds working results from current answers+grades.
func (s *SimpleQuizSession) ComputeWorkingResults() []SimpleQuizPlayerResult {
	byPlayer := make(map[string]*SimpleQuizPlayerResult)
	for _, p := range s.Players {
		byPlayer[p.ID] = &SimpleQuizPlayerResult{
			PlayerID:    p.ID,
			Nickname:    p.Nickname,
			PerQuestion: make([]SimpleQuizPerQuestionResult, len(s.Questions)),
		}
		for i := range s.Questions {
			byPlayer[p.ID].PerQuestion[i] = SimpleQuizPerQuestionResult{
				QuestionIndex: i,
				Verdict:       SimpleQuizVerdictMissing,
			}
		}
	}
	answerByKey := make(map[string]*SimpleQuizAnswer)
	for i := range s.Answers {
		a := &s.Answers[i]
		answerByKey[a.PlayerID+"/"+itoa(a.QuestionIndex)] = a
	}
	for pid, row := range byPlayer {
		total := 0
		for qi, q := range s.Questions {
			key := pid + "/" + itoa(qi)
			ans, ok := answerByKey[key]
			if !ok || ans == nil {
				row.PerQuestion[qi] = SimpleQuizPerQuestionResult{QuestionIndex: qi, Verdict: SimpleQuizVerdictMissing}
				continue
			}
			g, ok := s.Grades[ans.ID]
			if !ok || g.Verdict == SimpleQuizVerdictReviewLater {
				row.PerQuestion[qi] = SimpleQuizPerQuestionResult{QuestionIndex: qi, Verdict: SimpleQuizVerdictReviewLater}
				continue
			}
			if g.Verdict == SimpleQuizVerdictRight {
				row.PerQuestion[qi] = SimpleQuizPerQuestionResult{QuestionIndex: qi, Verdict: SimpleQuizVerdictRight, PointsEarned: q.Points}
				total += q.Points
			} else {
				row.PerQuestion[qi] = SimpleQuizPerQuestionResult{QuestionIndex: qi, Verdict: SimpleQuizVerdictWrong}
			}
		}
		row.TotalScore = total
	}
	out := make([]SimpleQuizPlayerResult, 0, len(byPlayer))
	for _, r := range byPlayer {
		out = append(out, *r)
	}
	// Sort by total desc, nickname asc for stable leaderboard.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].TotalScore > out[i].TotalScore ||
				(out[j].TotalScore == out[i].TotalScore && out[j].Nickname < out[i].Nickname) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// GradingComplete reports whether every submitted answer is finalized right/wrong.
func (s *SimpleQuizSession) GradingComplete() bool {
	for _, a := range s.Answers {
		g, ok := s.Grades[a.ID]
		if !ok {
			return false
		}
		if g.Verdict != SimpleQuizVerdictRight && g.Verdict != SimpleQuizVerdictWrong {
			return false
		}
	}
	return true
}

// ProgressFor returns counts for one question.
func (s *SimpleQuizSession) ProgressFor(questionIndex int) SimpleQuizGradingProgress {
	p := SimpleQuizGradingProgress{QuestionIndex: questionIndex}
	for _, a := range s.Answers {
		if a.QuestionIndex != questionIndex {
			continue
		}
		p.Received++
		g, ok := s.Grades[a.ID]
		if !ok {
			p.Ungraded++
			continue
		}
		switch g.Verdict {
		case SimpleQuizVerdictRight, SimpleQuizVerdictWrong:
			p.Graded++
		case SimpleQuizVerdictReviewLater:
			p.ReviewLater++
			p.Ungraded++
		default:
			p.Ungraded++
		}
	}
	return p
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
