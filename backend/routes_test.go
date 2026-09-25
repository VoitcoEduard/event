package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/go-expo-sample/backend/handlers"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
)

func TestRoutePatterns(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux, handlers.New(handlers.Dependencies{}))

	tests := []struct {
		method, path, pattern string
	}{
		{"POST", "/events/connect", "POST /events/connect"},
		{"GET", "/events/e1", "GET /events/{eventID}"},
		{"PUT", "/events/e1/modules/m1/responses/r1", "PUT /events/{eventID}/modules/{moduleID}/responses/{responseID}"},
		{"POST", "/events/e1/modules/m1/quiz/start", "POST /events/{eventID}/modules/{moduleID}/quiz/start"},
		{"PUT", "/events/e1/teams/t1", "PUT /events/{eventID}/teams/{teamID}"},
		{"GET", "/media/m1/file", "GET /media/{mediaID}/file"},
		{"DELETE", "/templates/t1", "DELETE /templates/{templateID}"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			_, pattern := mux.Handler(httptest.NewRequest(tt.method, tt.path, nil))
			if pattern != tt.pattern {
				t.Fatalf("matched %q, want %q", pattern, tt.pattern)
			}
		})
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("PATCH", "/events/e1/modules/m1", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unsupported method status = %d, want 405", w.Code)
	}
}

func TestRoutesUseTheirInjectedQuizHub(t *testing.T) {
	firstHub := hub.NewQuizHub(hub.Dependencies{})
	firstHub.CreateSession("quiz-1", "event-1")
	secondHub := hub.NewQuizHub(hub.Dependencies{})

	for _, tt := range []struct {
		name string
		hub  *hub.QuizHub
		want string
	}{
		{name: "first", hub: firstHub, want: "lobby"},
		{name: "second", hub: secondHub, want: "inactive"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			registerRoutes(mux, handlers.New(handlers.Dependencies{QuizHub: tt.hub}))
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/events/event-1/modules/quiz-1/quiz/state", nil))
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			var session models.QuizSession
			if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil {
				t.Fatal(err)
			}
			if session.State != tt.want {
				t.Fatalf("state = %q, want %q", session.State, tt.want)
			}
		})
	}
}

func TestAppHandlerKeepsAPIErrorsOutOfSPA(t *testing.T) {
	api := http.NewServeMux()
	api.HandleFunc("GET /events", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	spa := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	app := appHandler(api, spa)
	for _, tt := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/events", http.StatusNoContent},
		{http.MethodPost, "/events", http.StatusMethodNotAllowed},
		{http.MethodGet, "/events/missing", http.StatusNotFound},
		{http.MethodGet, "/admin", http.StatusOK},
	} {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
		if w.Code != tt.want {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, w.Code, tt.want)
		}
	}
}
