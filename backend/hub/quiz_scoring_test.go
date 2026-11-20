package hub

import (
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func TestServerScoresQuizAnswers(t *testing.T) {
	now := time.Unix(100, 0)
	window := models.QuestionWindow{StartTime: 90, EndTime: 110}
	tests := []struct {
		name   string
		q      models.QuizQuestion
		answer map[string]interface{}
		want   int
		valid  bool
	}{
		{"choice", models.QuizQuestion{Type: "classic", Answers: []string{"A", "B"}, CorrectIndex: 1, TimeLimit: 20}, map[string]interface{}{"answerIndex": float64(1), "scoreEarned": float64(999999)}, 31, true},
		{"wrong choice", models.QuizQuestion{Type: "classic", Answers: []string{"A", "B"}, CorrectIndex: 1}, map[string]interface{}{"answerIndex": float64(0), "scoreEarned": float64(999999)}, 0, true},
		{"multi", models.QuizQuestion{Type: "multi_select", Answers: []string{"A", "B", "C"}, CorrectIndices: []int{0, 2}}, map[string]interface{}{"answerIndices": []interface{}{float64(2), float64(0)}}, 31, true},
		{"duplicate choice", models.QuizQuestion{Type: "multi_select", Answers: []string{"A", "B"}, CorrectIndices: []int{0}}, map[string]interface{}{"answerIndices": []interface{}{float64(0), float64(0)}}, 0, false},
		{"empty multi answer", models.QuizQuestion{Type: "multi_select", Answers: []string{"A", "B"}}, map[string]interface{}{"answerIndices": []interface{}{}}, 0, false},
		{"number zero", models.QuizQuestion{Type: "number", CorrectNumber: 0}, map[string]interface{}{"answerValue": float64(0)}, 50, true},
		{"range exact zero", models.QuizQuestion{Type: "range", CorrectValue: 0, RangeConfig: models.RangeConfig{MinValue: -10, MaxValue: 10}}, map[string]interface{}{"answerValue": float64(0)}, 50, true},
		{"range out of bounds", models.QuizQuestion{Type: "range", RangeConfig: models.RangeConfig{MinValue: 0, MaxValue: 10}}, map[string]interface{}{"answerValue": float64(11)}, 0, false},
		{"picture", models.QuizQuestion{Type: "correct_picture", CorrectIndex: 1, ImageOptions: []models.ImageOption{{}, {}}}, map[string]interface{}{"answerIndex": float64(1)}, 31, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, _, _, valid := scoreQuizAnswer(tc.q, tc.answer, window, now, 0, 1)
			if got != tc.want || valid != tc.valid {
				t.Fatalf("score=%d valid=%v, want %d %v", got, valid, tc.want, tc.valid)
			}
		})
	}
}

func TestWrongPicturesUsesTrustedAspectAndIgnoresClaimedTarget(t *testing.T) {
	q := models.QuizQuestion{Type: "wrong_pictures", WrongPictureConfig: &models.WrongPictureConfig{
		MaxPoints: 101, Targets: []models.WrongPictureTarget{{ID: "target", XPercent: 50, YPercent: 50, Radius: 5}},
	}}
	answer := map[string]interface{}{"guesses": []interface{}{map[string]interface{}{
		"xPercent": float64(50), "yPercent": float64(58), "targetId": "forged", "isCorrect": true,
	}}}
	window := models.QuestionWindow{StartTime: 1}
	score, _, _, guesses, valid := scoreQuizAnswer(q, answer, window, time.Unix(2, 0), 0, 2)
	if !valid || score != 101 || guesses[0].TargetID != "target" {
		t.Fatalf("trusted aspect score=%d guesses=%+v valid=%v", score, guesses, valid)
	}
	score, _, _, _, _ = scoreQuizAnswer(q, answer, window, time.Unix(2, 0), 0, 1)
	if score != 0 {
		t.Fatalf("square image unexpectedly scored %d", score)
	}
}
