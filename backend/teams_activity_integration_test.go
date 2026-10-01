//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/handlers"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"github.com/user/go-expo-sample/backend/testutils"
	"go.mongodb.org/mongo-driver/bson"
)

func TestTeamsActivityHTTPAndPersistence(t *testing.T) {
	container := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, container)
	previous := db.Client
	db.Client = container.Client
	defer func() { db.Client = previous }()
	t.Setenv("MONGO_DB_NAME", "teams_integration")
	ctx := context.Background()
	db.EnsureIndexes()
	store := auth.Store{DB: db.Database()}
	if err := store.EnsureIndexes(ctx); err != nil {
		t.Fatal(err)
	}
	actor, err := store.UpsertGoogleUser(ctx, "test", "admin", "admin@example.test", "Admin")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Users().UpdateByID(ctx, actor.ID, bson.M{"$set": bson.M{"status": "active", "role": "admin"}})
	if err != nil {
		t.Fatal(err)
	}
	token, session, err := store.NewSession(ctx, actor.ID)
	if err != nil {
		t.Fatal(err)
	}
	authService := &auth.Service{Config: auth.Config{Mode: "dev", Environment: "development", BaseURL: "http://localhost:3000"}, Store: store}
	handlerModules := services.NewModuleService()
	handler := handlers.New(handlers.Dependencies{Auth: authService, Events: services.NewEventService(), Modules: handlerModules, Teams: database.NewTeamRepository(), Participants: services.NewParticipantService()})
	mux := http.NewServeMux()
	registerRoutes(mux, handler)
	modules := []models.Module{{ID: "relay", Type: models.ModuleTypeTeams, Name: "Relay"}, {ID: "build", Type: models.ModuleTypeTeams, Name: "Building"}, {ID: "reuse", Type: models.ModuleTypeTeams, Name: "Reuse"}, {ID: "legacy", Type: models.ModuleTypeTeams, Name: "Legacy"}, {ID: "quiz", Type: models.ModuleTypeQuiz, Name: "Quiz"}}
	_, err = db.GetCollection("events").InsertOne(ctx, models.Event{ID: "event", Modules: modules})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.GetCollection("quiz_results").InsertOne(ctx, models.QuizResult{EventID: "event", ModuleID: "quiz", FinishedAt: time.Now(), Players: []models.PlayerResult{{ID: "anna", Nickname: "Anna", Score: 7}}})
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body, userID string, admin bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", authService.Config.BaseURL)
		if userID != "" {
			r.Header.Set("X-Teams-Player", userID)
		}
		if admin {
			r.AddCookie(&http.Cookie{Name: "admin_session_dev", Value: token})
			r.Header.Set("X-CSRF-Token", session.CSRF)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	must := func(w *httptest.ResponseRecorder) *httptest.ResponseRecorder {
		t.Helper()
		if w.Code != 200 && w.Code != 204 {
			t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
		}
		return w
	}
	base := func(m string) string { return "/events/event/modules/" + m + "/teams" }
	action := func(m, body string) *httptest.ResponseRecorder {
		return must(request("POST", base(m)+"/actions", body, "", true))
	}
	assertModuleActive := func(moduleID string, want bool) {
		t.Helper()
		module, err := handlerModules.Get(ctx, "event", moduleID)
		if err != nil {
			t.Fatal(err)
		}
		if module.IsActive != want {
			t.Fatalf("module %s active = %v, want %v", moduleID, module.IsActive, want)
		}
	}
	for _, path := range []string{base("relay") + "/admin", base("relay") + "/actions"} {
		method := "GET"
		if strings.HasSuffix(path, "actions") {
			method = "POST"
		}
		if w := request(method, path, `{"action":"initialize"}`, "anna", false); w.Code != 401 {
			t.Fatalf("player gained admin access: %d", w.Code)
		}
	}
	action("relay", `{"action":"initialize","config":{"method":"choice","presets":[1,5,20]}}`)
	assertModuleActive("relay", true)
	action("relay", `{"action":"save_team","team":{"id":"blue","name":"Blue","color":"#3B82F6"}}`)
	action("relay", `{"action":"save_team","team":{"id":"red","name":"Red","color":"#EF4444"}}`)
	for _, id := range []string{"anna", "bob"} {
		must(request("POST", base("relay")+"/join", fmt.Sprintf(`{"name":%q}`, strings.Title(id)), id, false))
		must(request("POST", base("relay")+"/choose", `{"teamId":"blue"}`, id, false))
	}
	// Public choices cannot operate on an arbitrary body participant ID.
	w := request("POST", base("relay")+"/choose", `{"teamId":"red","participantId":"bob"}`, "unknown", false)
	if w.Code == 200 {
		t.Fatal("changed another person's assignment")
	}
	action("relay", `{"action":"start"}`)
	award := `{"action":"award","teamId":"blue","points":20,"reason":"Relay win","requestId":"relay-win"}`
	action("relay", award)
	action("relay", award)
	repo := database.NewTeamsActivityRepository()
	a, err := repo.Get(ctx, "event", "relay")
	if err != nil || len(a.Awards) != 1 {
		t.Fatalf("persisted retry %+v %v", a, err)
	}
	if len(a.Awards[0].Recipients) != 2 || a.Awards[0].ActorID != actor.ID {
		t.Fatal("recipient or actor snapshot missing")
	}
	if w := request("POST", base("relay")+"/actions", `{"action":"award","teamId":"blue","points":21,"requestId":"relay-win"}`, "", true); w.Code != 409 {
		t.Fatal("conflicting retry accepted")
	}
	if w := request("POST", base("relay")+"/choose", `{"teamId":"red"}`, "anna", false); w.Code != 409 {
		t.Fatal("player changed teams while live")
	}
	if w := request("DELETE", "/events/event/modules/relay", "", "", true); w.Code != 409 {
		t.Fatal("deleted module with scores")
	}
	action("relay", `{"action":"complete"}`)
	action("relay", `{"action":"complete"}`)
	assertModuleActive("relay", false)
	// Reusing a roster copies neither entrants nor points until people enter.
	action("reuse", `{"action":"initialize","config":{"sourceModuleId":"relay","method":"manual"}}`)
	must(request("POST", base("reuse")+"/join", `{"name":"Anna"}`, "anna", false))
	reused, err := repo.Get(ctx, "event", "reuse")
	if err != nil {
		t.Fatal(err)
	}
	if len(reused.Entrants) != 1 || reused.Entrants[0].TeamID != "blue" || len(reused.Awards) != 0 {
		t.Fatal("reuse copied entrants/awards")
	}
	action("reuse", `{"action":"assign","participantId":"anna","teamId":"red"}`)
	old, _ := repo.Get(ctx, "event", "relay")
	if old.Entrants[0].TeamID != "blue" {
		t.Fatal("modified source roster")
	}
	action("build", `{"action":"initialize","config":{"method":"manual"}}`)
	action("build", `{"action":"save_team","team":{"id":"red","name":"Red","color":"#EF4444"}}`)
	action("build", `{"action":"add_participant","participantId":"anna"}`)
	action("build", `{"action":"assign","participantId":"anna","teamId":"red"}`)
	action("build", `{"action":"start"}`)
	action("build", `{"action":"award","teamId":"red","points":15,"reason":"Building win","requestId":"build-win"}`)
	action("build", `{"action":"complete"}`)
	scores, err := database.LoadEventScores(ctx, "event")
	if err != nil || scores.PlayerTotal("anna") != 42 || scores.PlayerTotal("bob") != 20 {
		t.Fatalf("expected Anna 35+7: %+v %v", scores, err)
	}
	var view map[string]any
	w = must(request("GET", base("build")+"/state", "", "anna", false))
	if err = json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view["myPoints"] != float64(15) || view["totalPoints"] != float64(42) {
		t.Fatalf("wrong player view %v", view)
	}
	public := must(request("GET", base("relay")+"/projector", "", "", false)).Body.String()
	for _, secret := range []string{`"participantId"`, `"actorId"`, `"recipients"`, `"requestId"`} {
		if strings.Contains(public, secret) {
			t.Fatalf("public state leaked %s", secret)
		}
	}
	results := must(request("GET", "/events/event/modules/build/responses", "", "", true)).Body.String()
	if !strings.Contains(results, `"points":15`) || !strings.Contains(results, `"score":15`) {
		t.Fatalf("results missing scores: %s", results)
	}
	// Reopening and reversing uses the old recipients even after a member moves.
	action("relay", `{"action":"reopen"}`)
	assertModuleActive("relay", true)
	action("relay", `{"action":"assign","participantId":"anna","teamId":"red"}`)
	action("relay", fmt.Sprintf(`{"action":"undo","awardId":%q,"requestId":"undo-relay"}`, a.Awards[0].ID))
	action("relay", `{"action":"complete"}`)
	scores, err = database.LoadEventScores(ctx, "event")
	if err != nil || scores.PlayerTotal("anna") != 22 || scores.PlayerTotal("bob") != 0 {
		t.Fatal("reversal used current membership")
	}
	// Legacy manual team points remain team-only after importing the roster.
	if err = database.NewTeamRepository().Create(ctx, &models.Team{ID: "oldteam", EventID: "event", Name: "Old", Color: "#22C55E"}); err != nil {
		t.Fatal(err)
	}
	_, err = db.GetCollection("participants").UpdateByID(ctx, "anna", bson.M{"$set": bson.M{"teamId": "oldteam"}})
	if err != nil {
		t.Fatal(err)
	}
	if err = database.NewScoreEntryRepository().SetManual(ctx, "event", "legacy", "teams", "oldteam", 100); err != nil {
		t.Fatal(err)
	}
	action("legacy", `{"action":"initialize","importLegacy":true,"config":{"method":"choice"}}`)
	must(request("POST", base("legacy")+"/join", `{"name":"Anna"}`, "anna", false))
	legacy, err := repo.Get(ctx, "event", "legacy")
	if err != nil || len(legacy.Awards) != 0 || legacy.Entrants[0].TeamID != "oldteam" {
		t.Fatal("legacy import changed scoring or membership")
	}
	scores, err = database.LoadEventScores(ctx, "event")
	if err != nil || scores.PlayerTotal("anna") != 22 {
		t.Fatal("legacy team points distributed retrospectively")
	}
	// A new service/repository instance reads the same durable state.
	reloaded, err := (&services.TeamsActivityService{Store: database.NewTeamsActivityRepository()}).Store.Get(ctx, "event", "build")
	if err != nil || len(reloaded.Awards) != 1 || reloaded.Status != "completed" {
		t.Fatal("runtime did not survive reload")
	}
	if w := request("POST", "/events/other/modules/build/teams/actions", `{"action":"award","teamId":"red","points":9,"requestId":"cross-event"}`, "", true); w.Code != 404 {
		t.Fatal("cross-event module access")
	}
	if w := request("POST", "/events/event/modules/quiz/teams/actions", `{"action":"initialize"}`, "", true); w.Code != 404 {
		t.Fatal("non-teams module accepted")
	}
}
