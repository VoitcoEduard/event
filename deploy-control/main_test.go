package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestController(t *testing.T, handler http.HandlerFunc, initialPrevious []string) *controller {
	t.Helper()
	statePath := filepath.Join(t.TempDir(), "state.json")
	cfg := config{
		Token:           strings.Repeat("a", 32),
		PortainerURL:    "http://portainer.test",
		PortainerAPIKey: "test-key",
		StatePath:       statePath,
		Environments: map[string]environment{
			"dev": {
				Name:            "dev",
				StackName:       "app-dev",
				EndpointID:      1,
				ImageVariable:   "APP_IMAGE",
				ImageRepository: "ghcr.io/org/app",
			},
		},
	}
	c, err := newController(cfg)
	if err != nil {
		t.Fatal(err)
	}
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, r)
			return rec.Result(), nil
		}),
	}
	if len(initialPrevious) > 0 {
		c.environmentState("dev").Previous = append([]string(nil), initialPrevious...)
	}
	return c
}

func TestRollbackSkipsRedundantTopEntriesAndUnsticks(t *testing.T) {
	currentImage := "ghcr.io/org/app:sha-2222222222222222222222222222222222222222"
	olderImage := "ghcr.io/org/app:sha-1111111111111111111111111111111111111111"
	redeployedTo := ""

	portainer := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/stacks":
			_ = json.NewEncoder(w).Encode([]stack{{
				ID:         7,
				Name:       "app-dev",
				EndpointID: 1,
				Env:        []envVar{{Name: "APP_IMAGE", Value: currentImage}},
			}})
		case r.Method == http.MethodPut && r.URL.Path == "/api/stacks/7/git/redeploy":
			var body struct {
				Env []envVar `json:"env"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			redeployedTo, _ = imageValue(body.Env, "APP_IMAGE")
			currentImage = redeployedTo
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}

	// Top of Previous matches running image (twice); rollback must pop them and roll back to olderImage.
	c := newTestController(t, portainer, []string{olderImage, currentImage, currentImage})

	req := httptest.NewRequest(http.MethodPost, "/v1/environments/dev/rollback", nil)
	req.SetPathValue("environment", "dev")
	w := httptest.NewRecorder()
	c.rollback(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("rollback status = %d: %s", w.Code, w.Body.String())
	}
	if redeployedTo != olderImage {
		t.Fatalf("redeployed to %q, want %q", redeployedTo, olderImage)
	}
	if prev := c.environmentState("dev").Previous; len(prev) != 0 {
		t.Fatalf("remaining Previous = %v, want empty", prev)
	}

	// If Previous only contains entries matching currentImage, rollback cleans them and returns 409 Conflict.
	c2 := newTestController(t, portainer, []string{currentImage})
	req2 := httptest.NewRequest(http.MethodPost, "/v1/environments/dev/rollback", nil)
	req2.SetPathValue("environment", "dev")
	w2 := httptest.NewRecorder()
	c2.rollback(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("empty-after-pop rollback status = %d, want 409: %s", w2.Code, w2.Body.String())
	}
	if prev := c2.environmentState("dev").Previous; len(prev) != 0 {
		t.Fatalf("redundant Previous entry was not popped: %v", prev)
	}
}
