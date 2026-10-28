package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestPublicEventFiltersQuizSecretsWithoutChangingOriginal(t *testing.T) {
	quiz := models.Module{Type: models.ModuleTypeSimpleQuiz, Data: map[string]any{
		"judgeToken": "private-token", "allowAnswerReveal": true,
		"questions": []models.QuizQuestion{{Text: "secret question", JudgeNotes: "private note", CorrectAnswerText: "private answer"}},
	}}
	regular := models.Module{Type: models.ModuleTypeQuiz, Data: map[string]any{
		"judgeToken": "other-token", "questions": []models.QuizQuestion{{Text: "regular", CorrectIndex: 2, JudgeNotes: "private note", CorrectAnswerText: "private answer"}},
	}}
	event := models.Event{Modules: []models.Module{quiz, regular}}
	public, err := json.Marshal(publicEvent(event))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-token", "other-token", "secret question", "private note", "private answer"} {
		if strings.Contains(string(public), secret) {
			t.Fatalf("public event contains %q: %s", secret, public)
		}
	}
	if !strings.Contains(string(public), `"correctIndex":2`) {
		t.Fatalf("regular quiz client scoring data missing: %s", public)
	}
	original, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"private-token", "secret question", "private note", "private answer"} {
		if !strings.Contains(string(original), value) {
			t.Fatalf("projection changed administrator data: %s", original)
		}
	}
}
