package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestResponseResolveOtherAnswersFromJSON(t *testing.T) {
	payload := `{
		"userId": "player-1",
		"data": {
			"q_radio": "__other__",
			"q_dropdown": "__other__",
			"q_checkbox": ["pizza", "__other__"],
			"q_switched": "pasta",
			"q_text": "hello"
		},
		"otherAnswers": {
			"q_radio": "  Custom Radio  ",
			"q_dropdown": "Custom Dropdown",
			"q_checkbox": "  Sushi  ",
			"q_switched": "Should not overwrite predefined option"
		}
	}`

	var resp Response
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	resp.ResolveOtherAnswers()
	resp.ResolveOtherAnswers()

	if got := resp.Data["q_radio"]; got != "Custom Radio" {
		t.Fatalf("q_radio = %#v, want %q", got, "Custom Radio")
	}
	if got := resp.Data["q_dropdown"]; got != "Custom Dropdown" {
		t.Fatalf("q_dropdown = %#v, want %q", got, "Custom Dropdown")
	}
	wantCheckbox := []any{"pizza", "Sushi"}
	if got := resp.Data["q_checkbox"]; !reflect.DeepEqual(got, wantCheckbox) {
		t.Fatalf("q_checkbox = %#v, want %#v", got, wantCheckbox)
	}
	if got := resp.Data["q_switched"]; got != "pasta" {
		t.Fatalf("q_switched = %#v, want %q", got, "pasta")
	}
	if got := resp.Data["q_text"]; got != "hello" {
		t.Fatalf("q_text = %#v, want %q", got, "hello")
	}
}

func TestResponseResolveOtherAnswersEdgeCases(t *testing.T) {
	resp := &Response{
		Data: map[string]any{
			"q_strings": []string{"a", "__other__"},
			"q_blank":   "__other__",
			"q_missing": "__other__",
		},
		OtherAnswers: map[string]string{
			"q_strings": "custom",
			"q_blank":   "   ",
		},
	}

	resp.ResolveOtherAnswers()

	wantStrings := []string{"a", "custom"}
	if got := resp.Data["q_strings"]; !reflect.DeepEqual(got, wantStrings) {
		t.Fatalf("q_strings = %#v, want %#v", got, wantStrings)
	}
	if got := resp.Data["q_blank"]; got != "__other__" {
		t.Fatalf("q_blank = %#v, want %q", got, "__other__")
	}
	if got := resp.Data["q_missing"]; got != "__other__" {
		t.Fatalf("q_missing = %#v, want %q", got, "__other__")
	}
}
