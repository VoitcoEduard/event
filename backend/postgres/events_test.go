package postgres

import "testing"

func TestMergePreservedModuleData(t *testing.T) {
	existing := map[string]any{
		"manualScoreTarget": "teams",
		"judgeToken":        "secret-judge-token",
		"oldField":          "should-not-carry-over",
	}
	next := map[string]any{
		"timeline":         []any{map[string]any{"id": "step-1"}},
		"mandatoryOnEntry": true,
	}

	merged := mergePreservedModuleData(existing, next)
	if merged["manualScoreTarget"] != "teams" {
		t.Fatalf("expected manualScoreTarget=teams to be preserved, got %v", merged["manualScoreTarget"])
	}
	if merged["judgeToken"] != "secret-judge-token" {
		t.Fatalf("expected judgeToken to be preserved, got %v", merged["judgeToken"])
	}
	if _, exists := merged["oldField"]; exists {
		t.Fatalf("expected non-preserved key oldField to be dropped, got %v", merged["oldField"])
	}
	if merged["mandatoryOnEntry"] != true {
		t.Fatalf("expected mandatoryOnEntry=true, got %v", merged["mandatoryOnEntry"])
	}

	override := mergePreservedModuleData(existing, map[string]any{
		"manualScoreTarget": "players",
	})
	if override["manualScoreTarget"] != "players" {
		t.Fatalf("expected explicit manualScoreTarget=players to win, got %v", override["manualScoreTarget"])
	}
	if override["judgeToken"] != "secret-judge-token" {
		t.Fatalf("expected judgeToken to still be preserved, got %v", override["judgeToken"])
	}
}
