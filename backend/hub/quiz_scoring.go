package hub

import (
	"math"
	"sort"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

// scoreQuizAnswer uses only the stored question and the server's question
// window. The payload supplies an answer, never a point value or a clock.
func scoreQuizAnswer(q models.QuizQuestion, data map[string]interface{}, window models.QuestionWindow, now time.Time, streak int, aspect float64) (int, []int, float64, []models.PlayerGuess, bool) {
	limit := float64(EffectiveTimeLimit(q))
	spent := math.Max(0, now.Sub(time.Unix(window.StartTime, 0)).Seconds())
	maxPoints := q.ScoringConfig.MaxPoints
	if maxPoints <= 0 {
		maxPoints = 50
	}
	var indices []int
	var value float64
	var guesses []models.PlayerGuess
	correct := false
	switch q.Type {
	case models.QuestionTypeRange, models.QuestionTypeNumber:
		v, ok := data["answerValue"].(float64)
		if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, nil, 0, nil, false
		}
		value = v
		if q.Type == models.QuestionTypeNumber {
			if v == float64(q.CorrectNumber) {
				return maxPoints, nil, value, nil, true
			}
			return 0, nil, value, nil, true
		}
		if v < q.RangeConfig.MinValue || v > q.RangeConfig.MaxValue ||
			math.IsNaN(q.CorrectValue) || math.IsInf(q.CorrectValue, 0) {
			return 0, nil, 0, nil, false
		}
		tolerance := q.ScoringConfig.Tolerance
		distance := math.Abs(v - q.CorrectValue)
		if q.ScoringConfig.Mode == "asymmetric" && v > q.CorrectValue {
			return 0, nil, value, nil, true
		}
		if distance == 0 {
			return maxPoints, nil, value, nil, true
		}
		if tolerance <= 0 || distance > tolerance+0.0001 {
			return 0, nil, value, nil, true
		}
		return int(math.Round(float64(maxPoints) * math.Max(0, 1-distance/tolerance))), nil, value, nil, true
	case models.QuestionTypeWrongPictures:
		items, ok := data["guesses"].([]interface{})
		if !ok || len(items) > 100 || q.WrongPictureConfig == nil || !finitePositive(aspect) {
			return 0, nil, 0, nil, false
		}
		config := q.WrongPictureConfig
		maxGuesses := len(config.Targets) + 3
		if config.ShowTargetCount {
			maxGuesses = len(config.Targets)
		}
		if len(items) > maxGuesses {
			return 0, nil, 0, nil, false
		}
		if len(config.Targets) == 0 {
			return 0, nil, 0, nil, true
		}
		max := config.MaxPoints
		if max < 0 {
			return 0, nil, 0, nil, false
		}
		base, remainder := max/len(config.Targets), max%len(config.Targets)
		maxPoints = 0
		used := make(map[string]bool, len(config.Targets))
		for _, item := range items {
			entry, ok := item.(map[string]interface{})
			if !ok {
				return 0, nil, 0, nil, false
			}
			x, xOK := entry["xPercent"].(float64)
			y, yOK := entry["yPercent"].(float64)
			if !xOK || !yOK || math.IsNaN(x) || math.IsNaN(y) || x < 0 || x > 100 || y < 0 || y > 100 {
				return 0, nil, 0, nil, false
			}
			guess := models.PlayerGuess{XPercent: x, YPercent: y}
			best := math.Inf(1)
			matched := -1
			for i, target := range config.Targets {
				d := math.Hypot(x-target.XPercent, (y-target.YPercent)/aspect)
				if d < best {
					best = d
				}
				if !used[target.ID] && d <= target.Radius+config.TolerancePercent &&
					(matched < 0 || d < guess.Distance) {
					matched = i
					guess.Distance = d
				}
			}
			if matched >= 0 {
				target := config.Targets[matched]
				used[target.ID] = true
				guess.TargetID = target.ID
				guess.IsCorrect = true
				points := base
				if remainder > 0 {
					points++
					remainder--
				}
				maxPoints += points
			}
			guess.Distance = best
			guesses = append(guesses, guess)
		}
		return maxPoints, nil, 0, guesses, true
	case models.QuestionTypeMultiSelect:
		raw, ok := data["answerIndices"].([]interface{})
		if !ok || len(raw) == 0 || len(raw) > len(q.Answers) {
			return 0, nil, 0, nil, false
		}
		seen := make(map[int]bool, len(raw))
		for _, item := range raw {
			v, ok := exactIndex(item, len(q.Answers))
			if !ok || seen[v] {
				return 0, nil, 0, nil, false
			}
			seen[v] = true
			indices = append(indices, v)
		}
		sort.Ints(indices)
		wanted := append([]int(nil), q.CorrectIndices...)
		sort.Ints(wanted)
		correct = len(indices) == len(wanted)
		for i := range indices {
			if correct && indices[i] != wanted[i] {
				correct = false
			}
		}
	case models.QuestionTypeClassic, models.QuestionTypeTrueFalse, models.QuestionTypeCorrectPicture, "":
		count := len(q.Answers)
		if q.Type == models.QuestionTypeCorrectPicture {
			count = len(q.ImageOptions)
		}
		if q.Type == models.QuestionTypeTrueFalse && count == 0 {
			count = 2
		}
		v, ok := exactIndex(data["answerIndex"], count)
		if !ok {
			return 0, nil, 0, nil, false
		}
		indices = []int{v}
		correct = v == q.CorrectIndex
	default:
		return 0, nil, 0, nil, false
	}
	if !correct {
		return 0, indices, value, guesses, true
	}
	floorScore := math.Floor(float64(maxPoints) * 0.25)
	base := math.Max(floorScore, float64(maxPoints)-spent*(float64(maxPoints)*0.75/limit))
	bonus := min(max(0, streak)*(maxPoints/10), maxPoints/2)
	return int(math.Floor(base)) + bonus, indices, value, guesses, true
}

func finitePositive(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

func exactIndex(raw interface{}, count int) (int, bool) {
	v, ok := raw.(float64)
	return int(v), ok && v >= 0 && v < float64(count) && v == math.Trunc(v)
}
