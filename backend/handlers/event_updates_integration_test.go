//go:build integration

package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestEventUpdatesDistinguishMissingRecords(t *testing.T) {
	db := testutils.SetupPostgres(t)
	eventID := uuid.NewString()
	if err := db.CreateEvent(context.Background(), &models.Event{ID: eventID, Pin: "987654", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	h := New(Dependencies{DB: db, Events: services.NewEventService(db)})
	for _, tc := range []struct {
		path, body string
		handler    http.HandlerFunc
	}{
		{"status", `{"isActive":true}`, h.UpdateEventStatus},
		{"screensaver", `{}`, h.UpdateScreensaverConfig},
		{"slides", `[]`, h.SaveSlides},
		{"screen-state", `{"currentView":"screensaver"}`, h.UpdateScreenState},
	} {
		t.Run(tc.path, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("PUT /events/{eventID}/"+tc.path, tc.handler)
			for _, id := range []string{uuid.NewString(), eventID} {
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/events/"+id+"/"+tc.path, strings.NewReader(tc.body)))
				want := http.StatusNotFound
				if id == eventID {
					want = http.StatusOK
				}
				if w.Code != want {
					t.Fatalf("event %s: %d %s, want %d", id, w.Code, w.Body.String(), want)
				}
			}
		})
	}
}
