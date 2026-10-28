package models

import (
	"fmt"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func simpleBenchmarkFixture(players, questions int) *SimpleQuizSession {
	s := &SimpleQuizSession{ModuleID: "bench", Players: map[string]*SimpleQuizPlayer{}, Grades: map[string]*SimpleQuizGrade{}, Questions: make([]SimpleQuizQuestionSnapshot, questions)}
	for q := range s.Questions {
		s.Questions[q] = SimpleQuizQuestionSnapshot{ID: fmt.Sprint(q), Text: "Example question", Points: 100}
	}
	for p := 0; p < players; p++ {
		id := fmt.Sprintf("p-%03d", p)
		s.Players[id] = &SimpleQuizPlayer{ID: id, Nickname: id}
		for q := range s.Questions {
			answerID := fmt.Sprintf("%s-%d", id, q)
			s.Answers = append(s.Answers, SimpleQuizAnswer{ID: answerID, PlayerID: id, QuestionIndex: q, Text: strings.Repeat("answer ", 10)})
			s.Grades[answerID] = &SimpleQuizGrade{AnswerID: answerID, Verdict: SimpleQuizVerdictRight}
		}
	}
	return s
}

func BenchmarkSimpleQuizWorkingResults(b *testing.B) {
	for _, players := range []int{50, 150, 300} {
		b.Run(fmt.Sprintf("players=%d/questions=50", players), func(b *testing.B) {
			s := simpleBenchmarkFixture(players, 50)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				rows := s.ComputeWorkingResults()
				if len(rows) != players || rows[0].TotalScore != 5000 {
					b.Fatal("invalid working results")
				}
			}
		})
	}
}

// The application replaces this whole BSON document on each submission/grade.
// This benchmark isolates serialization only; it excludes MongoDB and disk I/O.
func BenchmarkSimpleQuizPersistenceBSON(b *testing.B) {
	for _, questions := range []int{1, 25, 100} {
		b.Run(fmt.Sprintf("players=150/questions=%d", questions), func(b *testing.B) {
			s := simpleBenchmarkFixture(150, questions)
			data, err := bson.Marshal(s)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := bson.Marshal(s); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(len(data)), "document-B")
		})
	}
}

func BenchmarkTeamsScoreProjection(b *testing.B) {
	for _, awards := range []int{50, 500, 2000} {
		b.Run(fmt.Sprintf("awards=%d/recipients=15", awards), func(b *testing.B) {
			a := &TeamsActivity{ID: "bench", EventID: "bench", ModuleID: "bench"}
			for i := 0; i < awards; i++ {
				recipients := make([]string, 15)
				for p := range recipients {
					recipients[p] = fmt.Sprintf("team-%d-person-%d", i%10, p)
				}
				a.Awards = append(a.Awards, TeamAward{ID: fmt.Sprint(i), TeamID: fmt.Sprintf("team-%d", i%10), Points: 20, Recipients: recipients})
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				entries := a.ScoreEntries()
				if len(entries) != awards*16 {
					b.Fatal("invalid projection")
				}
			}
		})
	}
}
