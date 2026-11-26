package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
)

func simpleQuizStateHub(t *testing.T) *hub.QuizHub {
	t.Helper()
	h := hub.NewQuizHub(hub.Dependencies{})
	t.Cleanup(h.Shutdown)
	h.SetSimpleQuiz(&models.SimpleQuizSession{
		ModuleID: "m", EventID: "e", Status: models.SimpleQuizStateQuestionOpen,
		Players: map[string]*models.SimpleQuizPlayer{"p": {ID: "p", Secret: "private"}},
	})
	return h
}

func doGetSimpleQuizState(t *testing.T, h *hub.QuizHub, query string, secret string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	handler := New(Dependencies{QuizHub: h})
	r := httptest.NewRequest("GET", "/events/e/modules/m/simple-quiz/state"+query, nil)
	if secret != "" {
		r.Header.Set("X-Player-Secret", secret)
	}
	r.SetPathValue("eventID", "e")
	r.SetPathValue("moduleID", "m")
	w := httptest.NewRecorder()
	handler.GetSimpleQuizState(w, r)
	var body map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &body) // plain-text error bodies stay nil
	}
	return w, body
}

// M3: a bare ?playerId=X request defaults to the player view BEFORE
// authorization, so it cannot bypass the player-secret check.
func TestSimpleQuizStatePlayerIDRequiresSecretWhenViewOmitted(t *testing.T) {
	h := simpleQuizStateHub(t)

	if w, _ := doGetSimpleQuizState(t, h, "?playerId=p", ""); w.Code != 403 {
		t.Fatalf("status without secret = %d, want 403 (body %s)", w.Code, w.Body.String())
	}
	if w, _ := doGetSimpleQuizState(t, h, "?playerId=p", "wrong"); w.Code != 403 {
		t.Fatalf("status with wrong secret = %d, want 403 (body %s)", w.Code, w.Body.String())
	}
	w, body := doGetSimpleQuizState(t, h, "?playerId=p", "private")
	if w.Code != 200 {
		t.Fatalf("status with valid secret = %d, want 200 (body %s)", w.Code, w.Body.String())
	}
	if _, ok := body["submitted"]; !ok {
		t.Fatalf("player view must include the submitted flag, got %s", w.Body.String())
	}

	w, body = doGetSimpleQuizState(t, h, "", "")
	if w.Code != 200 {
		t.Fatalf("status for anonymous project view = %d, want 200", w.Code)
	}
	if _, ok := body["submitted"]; ok {
		t.Fatalf("non-player view must not include submitted, got %s", w.Body.String())
	}
}

// M3 regression guard: explicitly non-player views must not leak player data to
// a playerId-carrying request without the secret.
func TestSimpleQuizStateExplicitHostViewStillRequiresAdmin(t *testing.T) {
	h := simpleQuizStateHub(t)
	if w, _ := doGetSimpleQuizState(t, h, "?view=host&playerId=p", "private"); w.Code == 200 && strings.Contains(w.Body.String(), "submitted") {
		t.Fatal("explicit host view with playerId unexpectedly returned player data")
	}
	// Explicit player view keeps requiring the secret.
	if w, _ := doGetSimpleQuizState(t, h, "?view=player&playerId=p", ""); w.Code != 403 {
		t.Fatalf("explicit player view without secret = %d, want 403", w.Code)
	}
}

// H1: blank module names must 400 on both update and publish.
func TestModuleRenameBlankNameRejected(t *testing.T) {
	h := New(Dependencies{})
	for _, tc := range []struct {
		name   string
		method string
		body   string
	}{
		{"update spaces", "PUT", `{"name":"   ","data":{"timeline":[]}}`},
		{"update only name", "PUT", `{"name":"   "}`},
		{"publish spaces", "PUT", `{"name":"   ","data":{"timeline":[]}}`},
		{"publish only name", "PUT", `{"name":"   "}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/events/e/modules/m", strings.NewReader(tc.body))
			r.SetPathValue("eventID", "e")
			r.SetPathValue("moduleID", "m")
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			if strings.Contains(tc.body, "timeline") && strings.HasPrefix(tc.name, "publish") {
				h.PublishModule(w, r)
			} else {
				h.UpdateModule(w, r)
			}
			if w.Code != 400 {
				t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "Module name cannot be empty") {
				t.Fatalf("unexpected error body: %s", w.Body.String())
			}
		})
	}
}
