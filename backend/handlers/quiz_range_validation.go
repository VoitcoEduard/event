package handlers

import (
	"encoding/json"
	"errors"
	"math"

	"github.com/user/go-expo-sample/backend/models"
)

var errInvalidQuizRange = errors.New("Range minimum, maximum, step and target must be finite; minimum must not exceed maximum")

func validateQuizRanges(questions []models.QuizQuestion) error {
	for _, q := range questions {
		if q.Type != models.QuestionTypeRange {
			continue
		}
		c := q.RangeConfig
		if !finite(c.MinValue) || !finite(c.MaxValue) || !finite(c.Step) || !finite(q.CorrectValue) ||
			!finite(q.ScoringConfig.Tolerance) || c.MinValue > c.MaxValue || c.Step <= 0 ||
			q.CorrectValue < c.MinValue || q.CorrectValue > c.MaxValue || q.ScoringConfig.Tolerance < 0 {
			return errInvalidQuizRange
		}
	}
	return nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func validateQuizRangesData(data map[string]any) error {
	if data == nil || data["questions"] == nil {
		return nil
	}
	raw, err := json.Marshal(data["questions"])
	if err != nil {
		return errInvalidQuizRange
	}
	var questions []models.QuizQuestion
	if err := json.Unmarshal(raw, &questions); err != nil {
		return errInvalidQuizRange
	}
	return validateQuizRanges(questions)
}
