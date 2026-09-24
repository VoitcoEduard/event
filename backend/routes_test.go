package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutePatterns(t *testing.T) {
	mux := http.NewServeMux()
	registerRoutes(mux, nil)

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
