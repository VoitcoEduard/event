package database

import (
	"fmt"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

// Isolates the in-memory total scan used after LoadEventScores. Database
// aggregation and cache misses must be measured with the deployment load test.
func BenchmarkEventPlayerTotal(b *testing.B) {
	for _, modules := range []int{5, 25, 100} {
		b.Run(fmt.Sprintf("modules=%d/players=150", modules), func(b *testing.B) {
			s := &EventScores{}
			for m := 0; m < modules; m++ {
				game := models.QuizResult{ModuleID: fmt.Sprint(m)}
				for p := 0; p < 150; p++ {
					id := fmt.Sprintf("p-%d", p)
					game.Players = append(game.Players, models.PlayerResult{ID: id, Score: 1000})
					s.Entries = append(s.Entries, models.ScoreEntry{ParticipantID: id, ModuleID: game.ModuleID, Points: 20})
				}
				s.Games = append(s.Games, game)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if s.PlayerTotal("p-149") != float64(modules*1020) {
					b.Fatal("incorrect total")
				}
			}
		})
	}
}
