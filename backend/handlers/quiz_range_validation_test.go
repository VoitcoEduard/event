package handlers

import (
	"math"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestRangeValidationAcceptsFixedValueAndRejectsInvalidBounds(t *testing.T) {
	q := models.QuizQuestion{Type: models.QuestionTypeRange, CorrectValue: 5,
		RangeConfig: models.RangeConfig{MinValue: 5, MaxValue: 5, Step: 1}}
	if err := validateQuizRanges([]models.QuizQuestion{q}); err != nil {
		t.Fatalf("fixed range rejected: %v", err)
	}
	q.RangeConfig.MaxValue = 4
	if err := validateQuizRanges([]models.QuizQuestion{q}); err == nil {
		t.Fatal("reversed range accepted")
	}
	q.RangeConfig.MaxValue = math.Inf(1)
	if err := validateQuizRanges([]models.QuizQuestion{q}); err == nil {
		t.Fatal("nonfinite range accepted")
	}
}
