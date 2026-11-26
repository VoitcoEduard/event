package models

func PublicQuizQuestion(q QuizQuestion) map[string]any {
	out := map[string]any{
		"id": q.ID, "type": q.Type, "text": q.Text,
		"answers": q.Answers, "timeLimit": q.TimeLimit,
		"explainTime": q.ExplainTime, "imageId": q.ImageID,
		"imageAlt": q.ImageAlt, "imageOptions": q.ImageOptions,
		"rangeConfig": q.RangeConfig, "scoringConfig": q.ScoringConfig,
	}
	if q.WrongPictureConfig != nil {
		out["wrongPictureConfig"] = map[string]any{
			"showTargetCount": q.WrongPictureConfig.ShowTargetCount,
			"targetCount":     len(q.WrongPictureConfig.Targets),
			"maxPoints":       q.WrongPictureConfig.MaxPoints,
		}
	}
	return out
}
