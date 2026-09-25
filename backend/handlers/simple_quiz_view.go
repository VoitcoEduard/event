package handlers

import (
	"strings"

	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
)

func armSimpleQuizTimer(h *hub.QuizHub, moduleID string, deadlineUnix int64) {
	if h == nil || deadlineUnix <= 0 {
		return
	}
	h.ArmSimpleQuizTimer(moduleID, deadlineUnix)
}

// publicSimpleQuizState filters the session per role.
// host: full except player secrets/answer texts? Host sees counts + progress + questions without notes.
// player: current question (no notes; correct answer only if revealed+allowed), own submitted flag, published leaderboard.
// judge: never called here (queue endpoint is separate); state gives progress only.
// projector: lobby/question/counts/revealed/published leaderboard.
func publicSimpleQuizState(s *models.SimpleQuizSession, view, playerID string) map[string]any {
	if s == nil {
		return map[string]any{"status": "inactive"}
	}
	answered := 0
	for _, a := range s.Answers {
		if a.QuestionIndex == s.CurrentIndex {
			answered++
		}
	}
	base := map[string]any{
		"eventId":          s.EventID,
		"moduleId":         s.ModuleID,
		"title":            s.Title,
		"status":           s.Status,
		"currentIndex":     s.CurrentIndex,
		"total":            len(s.Questions),
		"players":          len(s.Players),
		"answeredCount":    answered,
		"revealed":         s.Revealed,
		"deadlineUnix":     s.DeadlineUnix,
		"allowReveal":      s.AllowReveal,
		"publishedVersion": s.PublishedVer,
		"published":        s.Published,
		"updatedAt":        s.UpdatedAt,
	}
	progress := make([]models.SimpleQuizGradingProgress, 0, len(s.Questions))
	for i := range s.Questions {
		progress = append(progress, s.ProgressFor(i))
	}
	base["progress"] = progress

	switch view {
	case "host":
		qs := make([]map[string]any, 0, len(s.Questions))
		for i, q := range s.Questions {
			qs = append(qs, map[string]any{
				"id": i, "questionId": q.ID, "text": q.Text,
				"imageId": q.ImageID, "imageAlt": q.ImageAlt,
				"points": q.Points, "timeLimit": q.TimeLimit,
				"hasCorrectAnswer": strings.TrimSpace(q.CorrectAnswerText) != "",
				"revealed":         s.Revealed[i],
			})
		}
		base["questions"] = qs
		names := make([]map[string]any, 0, len(s.Players))
		for _, p := range s.Players {
			names = append(names, map[string]any{"playerId": p.ID, "nickname": p.Nickname})
		}
		base["playerList"] = names
		base["gradingComplete"] = s.GradingComplete()
		base["working"] = s.ComputeWorkingResults()
		return base
	case "player":
		// Current question without notes/correct answer unless revealed.
		var cur any
		if s.CurrentIndex >= 0 && s.CurrentIndex < len(s.Questions) {
			q := s.Questions[s.CurrentIndex]
			cq := map[string]any{
				"text": q.Text, "imageId": q.ImageID, "imageAlt": q.ImageAlt,
				"points": q.Points, "timeLimit": q.TimeLimit,
				"revealed": s.Revealed[s.CurrentIndex],
			}
			if s.AllowReveal && s.Revealed[s.CurrentIndex] && strings.TrimSpace(q.CorrectAnswerText) != "" {
				cq["correctAnswer"] = q.CorrectAnswerText
			}
			cur = cq
		}
		base["question"] = cur
		if playerID != "" {
			submitted := false
			for _, a := range s.Answers {
				if a.PlayerID == playerID && a.QuestionIndex == s.CurrentIndex {
					submitted = true
					break
				}
			}
			base["submitted"] = submitted
		}
		// Published personal + leaderboard is delivered via results endpoint;
		// include leaderboard here for convenience once published.
		return base
	default: // projector, judge-state, anonymous
		qs := make([]map[string]any, 0, len(s.Questions))
		for i, q := range s.Questions {
			item := map[string]any{
				"text": q.Text, "imageId": q.ImageID, "imageAlt": q.ImageAlt,
				"points": q.Points, "timeLimit": q.TimeLimit, "revealed": s.Revealed[i],
			}
			if s.AllowReveal && s.Revealed[i] && strings.TrimSpace(q.CorrectAnswerText) != "" {
				item["correctAnswer"] = q.CorrectAnswerText
			}
			qs = append(qs, item)
		}
		base["questions"] = qs
		return base
	}
}
