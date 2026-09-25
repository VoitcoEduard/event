package models

import (
	"testing"
	"time"
)

func TestSimpleQuizWorkingResults(t *testing.T) {
	s := &SimpleQuizSession{
		Questions: []SimpleQuizQuestionSnapshot{
			{ID: "q1", Text: "Capital?", Points: 100},
			{ID: "q2", Text: "2+2?", Points: 50},
		},
		Players: map[string]*SimpleQuizPlayer{
			"p1": {ID: "p1", Nickname: "Ana"},
			"p2": {ID: "p2", Nickname: "Bob"},
		},
		Answers: []SimpleQuizAnswer{
			{ID: "a1", PlayerID: "p1", QuestionIndex: 0, Text: "Paris"},
			{ID: "a2", PlayerID: "p2", QuestionIndex: 0, Text: "London"},
			{ID: "a3", PlayerID: "p1", QuestionIndex: 1, Text: "4"},
		},
		Grades: map[string]*SimpleQuizGrade{
			"a1": {AnswerID: "a1", Verdict: SimpleQuizVerdictRight, UpdatedAt: time.Now()},
			"a2": {AnswerID: "a2", Verdict: SimpleQuizVerdictWrong, UpdatedAt: time.Now()},
			"a3": {AnswerID: "a3", Verdict: SimpleQuizVerdictReviewLater, UpdatedAt: time.Now()},
		},
		Revealed: map[int]bool{},
	}
	if s.GradingComplete() {
		t.Fatal("review_later must block publication")
	}
	rows := s.ComputeWorkingResults()
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Ana: Q1 right (100), Q2 review_later (unresolved). Bob: Q1 wrong, Q2 missing.
	var ana, bob *SimpleQuizPlayerResult
	for i := range rows {
		if rows[i].PlayerID == "p1" {
			ana = &rows[i]
		}
		if rows[i].PlayerID == "p2" {
			bob = &rows[i]
		}
	}
	if ana == nil || bob == nil {
		t.Fatal("missing rows")
	}
	if ana.TotalScore != 100 {
		t.Fatalf("ana total = %d, want 100", ana.TotalScore)
	}
	if ana.PerQuestion[1].Verdict != SimpleQuizVerdictReviewLater {
		t.Fatalf("ana Q2 = %s, want review_later", ana.PerQuestion[1].Verdict)
	}
	if bob.PerQuestion[1].Verdict != SimpleQuizVerdictMissing {
		t.Fatalf("bob Q2 = %s, want missing", bob.PerQuestion[1].Verdict)
	}
	p := s.ProgressFor(0)
	if p.Received != 2 || p.Graded != 2 || p.ReviewLater != 0 {
		t.Fatalf("Q1 progress = %+v", p)
	}
	p1 := s.ProgressFor(1)
	if p1.Received != 1 || p1.Graded != 0 || p1.ReviewLater != 1 {
		t.Fatalf("Q2 progress = %+v", p1)
	}
	// Finalize Q2 -> complete.
	s.Grades["a3"].Verdict = SimpleQuizVerdictRight
	if !s.GradingComplete() {
		t.Fatal("expected grading complete")
	}
	rows = s.ComputeWorkingResults()
	for i := range rows {
		if rows[i].PlayerID == "p1" && rows[i].TotalScore != 150 {
			t.Fatalf("ana total after finalize = %d, want 150", rows[i].TotalScore)
		}
	}
	// Leaderboard order: Ana first.
	if rows[0].PlayerID != "p1" {
		t.Fatalf("leaderboard order wrong: %+v", rows)
	}
}

func TestSnapshotSimpleQuizQuestion(t *testing.T) {
	q := QuizQuestion{
		ID: "q1", Type: QuestionTypeSimpleText, Text: "Q?",
		TimeLimit: 30, JudgeNotes: "note", CorrectAnswerText: "ans",
		ScoringConfig: ScoringConfig{MaxPoints: 75},
	}
	snap := SnapshotSimpleQuizQuestion(q)
	if snap.Points != 75 || snap.TimeLimit != 30 || snap.JudgeNotes != "note" || snap.CorrectAnswerText != "ans" {
		t.Fatalf("snapshot = %+v", snap)
	}
}
