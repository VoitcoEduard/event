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
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestTeamEditAndDeleteRoutes(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	eventID, otherID, teamID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for i, id := range []string{eventID, otherID} {
		if err := db.CreateEvent(ctx, &models.Event{ID: id, Pin: []string{"111111", "222222"}[i], CreatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.CreateTeam(ctx, &models.Team{ID: teamID, EventID: eventID, Name: "Old", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	h := New(Dependencies{DB: db, Teams: database.NewTeamRepository(db)})
	mux := http.NewServeMux()
	mux.HandleFunc("PUT /events/{eventID}/teams/{teamID}", h.UpdateTeam)
	mux.HandleFunc("DELETE /events/{eventID}/teams/{teamID}", h.DeleteTeam)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/events/"+otherID+"/teams/"+teamID, nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-event deletion: %d", w.Code)
	}
	for _, tc := range []struct{ method, body string }{{"PUT", `{"name":"New"}`}, {"DELETE", ""}} {
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(tc.method, "/events/"+eventID+"/teams/"+teamID, strings.NewReader(tc.body)))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", tc.method, w.Code, w.Body.String())
		}
	}
}
