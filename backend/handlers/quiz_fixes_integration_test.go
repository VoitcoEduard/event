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
	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestQuizStateAuthorizationAndEventCacheKeys(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC()
	if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: "410000", CreatedAt: now, Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeQuiz}}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { invalidateEventCache(eventID) })
	qh := hub.NewQuizHub(hub.Dependencies{DB: db})
	t.Cleanup(qh.Shutdown)
	qh.CreateSession(moduleID, eventID)
	qh.UpdateSession(moduleID, func(s *models.QuizSession) {
		s.Players["private-id"] = &models.Player{ID: "private-id", Nickname: "private-name"}
		s.PlayerSignals = []models.PlayerSignal{{PlayerID: "private-id", AnswerValue: 12345}}
	})
	store := auth.Store{DB: db}
	owner, err := store.EnsureDevUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.NewSession(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	h := New(Dependencies{DB: db, QuizHub: qh, Events: services.NewEventService(db), Auth: &auth.Service{
		Config: auth.Config{BaseURL: "http://localhost"}, Store: store,
	}})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/{eventID}", h.GetEvent)
	mux.HandleFunc("GET /events/{eventID}/modules/{moduleID}/quiz/state", h.GetQuizState)
	request := func(path string, admin bool) string {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		if admin {
			r.AddCookie(&http.Cookie{Name: "admin_session_dev", Value: token})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	statePath := "/events/" + eventID + "/modules/" + moduleID + "/quiz/state"
	if body := request(statePath, true); !strings.Contains(body, "private-name") || !strings.Contains(body, "12345") {
		t.Fatalf("administrator session data missing: %s", body)
	}
	if body := request(statePath, false); strings.Contains(body, "private-name") || strings.Contains(body, "12345") {
		t.Fatalf("public session leaks private data: %s", body)
	}
	request("/events/"+eventID, false)
	for range 20 {
		userID := uuid.NewString()
		request("/events/"+eventID+"?userId="+userID, false)
		if _, ok := eventResponseCache.Get(eventCacheKey(eventID, userID)); ok {
			t.Fatal("unknown user created a distinct cache entry")
		}
	}
	if _, ok := eventResponseCache.Get(eventCacheKey(eventID, "")); !ok {
		t.Fatal("anonymous event response was not cached")
	}
	const legacyID = "legacy-user-id"
	if _, err := db.EnsureParticipant(ctx, &models.Participant{
		ID: legacyID, EventID: eventID, JoinKey: "legacy-device", DisplayName: "Known participant",
		NormalizedName: "known participant", Status: models.ParticipantStatusJoined, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if body := request("/events/"+eventID+"?userId="+legacyID, false); !strings.Contains(body, "Known participant") {
		t.Fatalf("valid legacy participant ID rejected or status lost: %s", body)
	}
	if _, ok := eventResponseCache.Get(eventCacheKey(eventID, legacyID)); !ok {
		t.Fatal("known participant response was not cached")
	}
}

func TestDeletingModulesAndEventsEvictsQuizSessions(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	for i, kind := range []string{"module", "event"} {
		t.Run(kind, func(t *testing.T) {
			eventID, moduleID := uuid.NewString(), uuid.NewString()
			now := time.Now().UTC()
			if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: []string{"410001", "410002"}[i], CreatedAt: now, Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeSimpleQuiz}}}); err != nil {
				t.Fatal(err)
			}
			qh := hub.NewQuizHub(hub.Dependencies{DB: db})
			t.Cleanup(qh.Shutdown)
			qh.CreateSession(moduleID, eventID)
			if err := qh.TrySetSimpleQuiz(&models.SimpleQuizSession{
				ModuleID: moduleID, EventID: eventID, Status: models.SimpleQuizStateLobby, CreatedAt: now, UpdatedAt: now,
			}); err != nil {
				t.Fatal(err)
			}
			h := New(Dependencies{DB: db, QuizHub: qh, Events: services.NewEventService(db), Modules: services.NewModuleService(db)})
			mux := http.NewServeMux()
			mux.HandleFunc("DELETE /events/{eventID}", h.DeleteEvent)
			mux.HandleFunc("DELETE /events/{eventID}/modules/{moduleID}", h.DeleteModule)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/events/"+uuid.NewString()+"/modules/"+moduleID, nil))
			if w.Code != http.StatusNotFound || qh.GetSession(moduleID) == nil || qh.GetSimpleQuiz(moduleID) == nil {
				t.Fatalf("failed deletion evicted a session: %d %s", w.Code, w.Body.String())
			}
			path := "/events/" + eventID
			if kind == "module" {
				path += "/modules/" + moduleID
			}
			w = httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest("DELETE", path, nil))
			if w.Code != http.StatusOK {
				t.Fatalf("DELETE %s: %d %s", path, w.Code, w.Body.String())
			}
			if qh.GetSession(moduleID) != nil || qh.GetSimpleQuiz(moduleID) != nil {
				t.Fatal("deleted module still has a cached quiz session")
			}
		})
	}
}
