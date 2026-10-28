package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestPublicEventFiltersQuizSecretsWithoutChangingOriginal(t *testing.T) {
	quiz := models.Module{Type: models.ModuleTypeSimpleQuiz, Data: map[string]any{
		"judgeToken": "private-token", "allowAnswerReveal": true,
		"questions": []models.QuizQuestion{{Text: "secret question", JudgeNotes: "private note", CorrectAnswerText: "private answer"}},
	}}
	regular := models.Module{Type: models.ModuleTypeQuiz, Data: map[string]any{
		"judgeToken": "other-token", "questions": []models.QuizQuestion{{Text: "regular", CorrectIndex: 2, JudgeNotes: "private note", CorrectAnswerText: "private answer"}},
	}}
	event := models.Event{Modules: []models.Module{quiz, regular}}
	public, err := json.Marshal(publicEvent(event))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-token", "other-token", "secret question", "private note", "private answer"} {
		if strings.Contains(string(public), secret) {
			t.Fatalf("public event contains %q: %s", secret, public)
		}
	}
	if !strings.Contains(string(public), `"correctIndex":2`) {
		t.Fatalf("regular quiz client scoring data missing: %s", public)
	}
	original, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"private-token", "secret question", "private note", "private answer"} {
		if !strings.Contains(string(original), value) {
			t.Fatalf("projection changed administrator data: %s", original)
		}
	}
}

func TestPublicQuizResponsesAndAdminCacheIsolation(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	mt.Run("public and admin views", func(mt *mtest.T) {
		mt.Setenv("MONGO_DB_NAME", "eventdb")
		previous := db.Client
		db.Client = mt.Client
		defer func() { db.Client = previous; invalidateEventCache("privacy-event") }()
		service := &auth.Service{Config: auth.Config{BaseURL: "http://localhost"}, Store: auth.Store{DB: db.Database()}}
		h := New(Dependencies{Auth: service, Events: services.NewEventService(), Modules: services.NewModuleService(), Templates: services.NewTemplateService()})
		mux := http.NewServeMux()
		mux.HandleFunc("GET /events/{eventID}", h.GetEvent)
		mux.HandleFunc("POST /events/connect", h.ConnectToEvent)
		mux.HandleFunc("GET /events/{eventID}/modules/{moduleID}", h.GetModule)
		mux.HandleFunc("GET /templates", h.GetTemplates)
		secretData := bson.D{{Key: "judgeToken", Value: "private-token"}, {Key: "questions", Value: bson.A{bson.D{{Key: "text", Value: "private-question"}, {Key: "correctAnswerText", Value: "private-answer"}, {Key: "judgeNotes", Value: "private-notes"}}}}}
		event := bson.D{{Key: "_id", Value: "privacy-event"}, {Key: "pin", Value: "1234"}, {Key: "isActive", Value: true}, {Key: "modules", Value: bson.A{bson.D{{Key: "id", Value: "m"}, {Key: "type", Value: "simple_quiz"}, {Key: "data", Value: secretData}}}}}
		cursor := func(collection string, docs ...bson.D) bson.D {
			return mtest.CreateCursorResponse(0, "eventdb."+collection, mtest.FirstBatch, docs...)
		}
		request := func(method, path, body string, admin bool) *httptest.ResponseRecorder {
			r := httptest.NewRequest(method, path, strings.NewReader(body))
			if admin {
				r.AddCookie(&http.Cookie{Name: "admin_session_dev", Value: "token"})
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != 200 {
				mt.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
			}
			return w
		}
		assertPublic := func(body string) {
			for _, secret := range []string{"private-token", "private-question", "private-answer", "private-notes"} {
				if strings.Contains(body, secret) {
					mt.Fatalf("public response contains %s: %s", secret, body)
				}
			}
		}
		invalidateEventCache("privacy-event")
		mt.AddMockResponses(cursor("events", event), cursor("dynamic_cards"))
		assertPublic(request("GET", "/events/privacy-event", "", false).Body.String())
		mt.AddMockResponses(cursor("auth_sessions", bson.D{{Key: "userId", Value: "admin"}}), cursor("auth_users", bson.D{{Key: "_id", Value: "admin"}, {Key: "status", Value: "active"}, {Key: "role", Value: "admin"}}), cursor("events", event), cursor("dynamic_cards"))
		admin := request("GET", "/events/privacy-event", "", true).Body.String()
		if !strings.Contains(admin, "private-token") || !strings.Contains(admin, "private-answer") {
			mt.Fatalf("admin data missing: %s", admin)
		}
		assertPublic(request("GET", "/events/privacy-event", "", false).Body.String())
		mt.AddMockResponses(cursor("events", event))
		assertPublic(request("GET", "/events/privacy-event/modules/m", "", false).Body.String())
		mt.AddMockResponses(cursor("events", event))
		assertPublic(request("POST", "/events/connect", `{"pin":"1234"}`, false).Body.String())
		template := bson.D{{Key: "_id", Value: "template"}, {Key: "moduleType", Value: "simple_quiz"}, {Key: "data", Value: secretData}}
		mt.AddMockResponses(cursor("templates", template))
		assertPublic(request("GET", "/templates", "", false).Body.String())
	})
}
