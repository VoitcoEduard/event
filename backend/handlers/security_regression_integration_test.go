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

func TestResultsAndParticipantRenameAuthorization(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	event, other, module := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for i, id := range []string{event, other} {
		e := &models.Event{ID: id, Pin: []string{"438100", "438101"}[i], CreatedAt: time.Now(), IsActive: true}
		if id == event {
			e.Modules = []models.Module{{ID: module, Type: models.ModuleTypeQuiz}}
		}
		if err := db.CreateEvent(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	p, err := db.JoinEventParticipant(ctx, event, "player", "", "Original", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := db.JoinEventParticipant(ctx, other, "player", "", "Other", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	authStore := auth.Store{DB: db}
	owner, err := authStore.EnsureDevUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	token, session, err := authStore.NewSession(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	qh := hub.NewQuizHub(hub.Dependencies{DB: db})
	defer qh.Shutdown()
	qh.CreateSession(module, event)
	qh.UpdateSession(module, func(s *models.QuizSession) {
		s.Players["player"] = &models.Player{ID: "player", Nickname: "Alex", Score: 10}
		s.PlayerSignals = []models.PlayerSignal{{PlayerID: "player", AnswerValue: 12345}}
	})
	h := New(Dependencies{DB: db, Modules: services.NewModuleService(db), QuizHub: qh, Auth: &auth.Service{Config: auth.Config{BaseURL: "http://localhost"}, Store: authStore}})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/{eventID}/modules/{moduleID}/quiz/results", h.GetQuizResults)
	mux.HandleFunc("POST /events/{eventID}/participants/{participantID}/name", h.UpdateParticipantName)
	resultPath := "/events/" + event + "/modules/" + module + "/quiz/results"
	for _, state := range []string{"lobby", "question-active", "results", "finished"} {
		qh.UpdateSession(module, func(s *models.QuizSession) { s.State = state })
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", resultPath, nil))
		body := w.Body.String()
		if w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || strings.Contains(body, "playerSignals") || strings.Contains(body, "12345") || strings.Contains(body, "\"player\"") {
			t.Fatalf("public %s: %d %s", state, w.Code, body)
		}
		if strings.Contains(body, "leaderboard") != (state == "finished") {
			t.Fatalf("leaderboard in %s: %s", state, body)
		}
	}
	r := httptest.NewRequest("GET", resultPath, nil)
	r.AddCookie(&http.Cookie{Name: "admin_session_dev", Value: token})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "12345") {
		t.Fatalf("admin results: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/events/"+other+"/modules/"+module+"/quiz/results", nil))
	if w.Code != 404 {
		t.Fatalf("mismatched event: %d", w.Code)
	}
	namePath := "/events/" + event + "/participants/player/name"
	for _, tc := range []struct {
		secret      string
		admin, csrf bool
		want        int
	}{{"", false, false, 403}, {"wrong", false, false, 403}, {foreign.JoinSecret, false, false, 403}, {p.JoinSecret, false, false, 200}, {"", true, false, 403}, {"", true, true, 200}} {
		r := httptest.NewRequest("POST", namePath, strings.NewReader(`{"displayName":"Updated","joinSecret":"`+tc.secret+`"}`))
		if tc.admin {
			r.AddCookie(&http.Cookie{Name: "admin_session_dev", Value: token})
		}
		if tc.csrf {
			r.Header.Set("Origin", "http://localhost")
			r.Header.Set("X-CSRF-Token", session.CSRF)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("rename admin=%v csrf=%v: %d %s", tc.admin, tc.csrf, w.Code, w.Body.String())
		}
	}
}
