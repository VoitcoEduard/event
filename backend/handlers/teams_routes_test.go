package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestTeamEditAndDeleteRoutes(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("valid event team paths", func(mt *mtest.T) {
		previous := db.Client
		db.Client = mt.Client
		defer func() { db.Client = previous }()
		h := New(Dependencies{Teams: database.NewTeamRepository()})
		mux := http.NewServeMux()
		mux.HandleFunc("PUT /events/{eventID}/teams/{teamID}", h.UpdateTeam)
		mux.HandleFunc("DELETE /events/{eventID}/teams/{teamID}", h.DeleteTeam)
		team := bson.D{{Key: "_id", Value: "t"}, {Key: "eventId", Value: "e"}, {Key: "name", Value: "Old"}}
		for _, tc := range []struct{ method, body string }{{"PUT", `{"name":"New"}`}, {"DELETE", ""}} {
			mt.AddMockResponses(mtest.CreateCursorResponse(0, "eventdb.event_teams", mtest.FirstBatch, team), mtest.CreateSuccessResponse(), mtest.CreateSuccessResponse())
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(tc.method, "/events/e/teams/t", strings.NewReader(tc.body)))
			if w.Code != http.StatusOK {
				mt.Fatalf("%s: status %d, body %s", tc.method, w.Code, w.Body.String())
			}
			mt.ClearMockResponses()
		}
		mt.AddMockResponses(mtest.CreateCursorResponse(0, "eventdb.event_teams", mtest.FirstBatch, team))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/events/other/teams/t", nil))
		if w.Code != http.StatusNotFound {
			mt.Fatalf("cross-event deletion status = %d, want 404", w.Code)
		}
	})
}
