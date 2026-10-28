//go:build integration

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/handlers"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestSimpleQuizRoutesEnforceEventOwnership(t *testing.T) {
	db := testutils.SetupPostgres(t)
	event1 := &models.Event{ID: uuid.NewString(), Pin: "423801", CreatedAt: time.Now(), Modules: []models.Module{{ID: uuid.NewString(), Type: models.ModuleTypeSimpleQuiz}}}
	event2 := &models.Event{ID: uuid.NewString(), Pin: "423802", CreatedAt: time.Now()}
	for _, e := range []*models.Event{event1, event2} {
		if err := db.CreateEvent(context.Background(), e); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	registerRoutes(mux, handlers.New(handlers.Dependencies{DB: db, QuizHub: hub.NewQuizHub(hub.Dependencies{DB: db})}))
	req := httptest.NewRequest(http.MethodGet, "/events/"+event2.ID+"/modules/"+event1.Modules[0].ID+"/simple-quiz/state", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-event request returned %d, want 404", rec.Code)
	}
}
