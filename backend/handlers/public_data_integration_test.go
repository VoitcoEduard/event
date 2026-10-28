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
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestPublicQuizResponsesAndAdminCacheIsolation(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	secret := map[string]any{"judgeToken": "private-token", "questions": []map[string]any{{"text": "private-question", "correctAnswerText": "private-answer", "judgeNotes": "private-notes"}}}
	event := &models.Event{ID: eventID, Pin: "123456", Name: "Event", IsActive: true, CreatedAt: time.Now().UTC(), Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeSimpleQuiz, Name: "Quiz", Data: secret}}}
	if err := db.CreateEvent(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateTemplate(ctx, &models.Template{ID: uuid.NewString(), Name: "Template", ModuleType: models.ModuleTypeSimpleQuiz, Data: secret, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	store := auth.Store{DB: db}
	owner, err := store.EnsureDevUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.NewSession(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Config: auth.Config{BaseURL: "http://localhost"}, Store: store}
	h := New(Dependencies{DB: db, Auth: authService, Events: services.NewEventService(db), Modules: services.NewModuleService(db), Templates: services.NewTemplateService(db)})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/{eventID}", h.GetEvent)
	mux.HandleFunc("POST /events/connect", h.ConnectToEvent)
	mux.HandleFunc("GET /events/{eventID}/modules/{moduleID}", h.GetModule)
	mux.HandleFunc("GET /templates", h.GetTemplates)
	request := func(method, path, body string, admin bool) string {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if admin {
			r.AddCookie(&http.Cookie{Name: "admin_session_dev", Value: token})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	assertPublic := func(body string) {
		t.Helper()
		for _, value := range []string{"private-token", "private-question", "private-answer", "private-notes"} {
			if strings.Contains(body, value) {
				t.Fatalf("public data leaked %s: %s", value, body)
			}
		}
	}
	t.Cleanup(func() { invalidateEventCache(eventID) })
	assertPublic(request("GET", "/events/"+eventID, "", false))
	admin := request("GET", "/events/"+eventID, "", true)
	if !strings.Contains(admin, "private-token") || !strings.Contains(admin, "private-answer") {
		t.Fatalf("admin data missing: %s", admin)
	}
	assertPublic(request("GET", "/events/"+eventID, "", false))
	assertPublic(request("GET", "/events/"+eventID+"/modules/"+moduleID, "", false))
	assertPublic(request("POST", "/events/connect", `{"pin":"123456"}`, false))
	assertPublic(request("GET", "/templates", "", false))
}
