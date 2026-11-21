package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

func screenSubscriberMapSize() int {
	screenSubMutex.Lock()
	defer screenSubMutex.Unlock()
	return len(screenSubscribers)
}

func screenSubscriberKeyExists(eventID string) bool {
	screenSubMutex.Lock()
	defer screenSubMutex.Unlock()
	_, ok := screenSubscribers[eventID]
	return ok
}

func TestScreenStreamRejectsInvalidAndMissingEventsWithoutLeaking(t *testing.T) {
	missingID := uuid.NewString()
	errID := uuid.NewString()

	h := New(Dependencies{})
	h.getEvent = func(_ context.Context, id string) (*models.Event, error) {
		if id == missingID {
			return nil, postgres.ErrNotFound
		}
		if id == errID {
			return nil, errors.New("db connection dropped")
		}
		return &models.Event{ID: id}, nil
	}

	cases := []struct {
		name       string
		path       string
		eventID    string
		wantStatus int
	}{
		{"invalid_url", "/events/only", "", http.StatusBadRequest},
		{"invalid_uuid", "/events/not-a-uuid/screen-stream", "not-a-uuid", http.StatusBadRequest},
		{"missing_event", "/events/" + missingID + "/screen-stream", missingID, http.StatusNotFound},
		{"db_error", "/events/" + errID + "/screen-stream", errID, http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.eventID != "" {
				req.SetPathValue("eventID", tc.eventID)
			}
			rec := httptest.NewRecorder()
			h.ScreenStream(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body=%q)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.eventID != "" && screenSubscriberKeyExists(tc.eventID) {
				t.Fatalf("screenSubscribers leaked key %q after %d response", tc.eventID, tc.wantStatus)
			}
		})
	}
}

func TestScreenStreamRemovesMapEntryWhenLastSubscriberDisconnects(t *testing.T) {
	eventID := uuid.NewString()
	h := New(Dependencies{})
	h.getEvent = func(_ context.Context, id string) (*models.Event, error) {
		if id == eventID {
			return &models.Event{
				ID:          id,
				ScreenState: &models.ScreenState{CurrentView: "slides"},
			}, nil
		}
		return nil, postgres.ErrNotFound
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	req1 := httptest.NewRequest(http.MethodGet, "/events/"+eventID+"/screen-stream", nil).WithContext(ctx1)
	req1.SetPathValue("eventID", eventID)
	rec1 := httptest.NewRecorder()
	done1 := make(chan struct{})
	go func() {
		defer close(done1)
		h.ScreenStream(rec1, req1)
	}()

	// Wait for first subscriber to register.
	deadline := time.Now().Add(2 * time.Second)
	for !screenSubscriberKeyExists(eventID) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !screenSubscriberKeyExists(eventID) {
		cancel1()
		t.Fatal("first subscriber did not register")
	}

	// Register a second subscriber for the same event.
	ch2 := addScreenSubscriber(eventID)

	// Push an update and disconnect the first subscriber.
	notifyScreenSubscribers(eventID, models.ScreenState{CurrentView: "quiz"})
	select {
	case got := <-ch2:
		if got.CurrentView != "quiz" {
			t.Fatalf("second subscriber got %q, want quiz", got.CurrentView)
		}
	case <-time.After(time.Second):
		t.Fatal("second subscriber did not receive update")
	}

	cancel1()
	select {
	case <-done1:
	case <-time.After(2 * time.Second):
		t.Fatal("ScreenStream did not exit after context cancellation")
	}

	if !strings.Contains(rec1.Body.String(), `"currentView":"slides"`) {
		t.Fatalf("first subscriber did not receive initial state: %q", rec1.Body.String())
	}
	// Second subscriber is still active, so the key should remain.
	if !screenSubscriberKeyExists(eventID) {
		t.Fatal("event key deleted while second subscriber was still connected")
	}

	// Removing the last subscriber must delete the map entry.
	removeScreenSubscriber(eventID, ch2)
	if screenSubscriberKeyExists(eventID) {
		t.Fatalf("event key %q leaked after last subscriber disconnected (map size=%d)", eventID, screenSubscriberMapSize())
	}
}
