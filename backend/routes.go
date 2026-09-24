package main

import (
	"net/http"
	"strings"

	"github.com/user/go-expo-sample/backend/handlers"
	"github.com/user/go-expo-sample/backend/hub"
)

// registerRoutes keeps the public API paths and authentication requirements in
// one place. Handlers still receive their original URL paths for compatibility.
func registerRoutes(mux *http.ServeMux, quizHub *hub.QuizHub) {
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) { hub.ServeWs(quizHub, w, r) })

	paths := make(map[string]bool)
	for _, route := range []struct {
		pattern string
		handler http.HandlerFunc
		admin   bool
	}{
		{"GET /events", handlers.GetEvents, false},
		{"POST /events", handlers.CreateEvent, true},
		{"POST /events/connect", handlers.ConnectToEvent, false},
		{"GET /events/{eventID}", handlers.GetEvent, false},
		{"PUT /events/{eventID}", handlers.UpdateEvent, true},
		{"DELETE /events/{eventID}", handlers.DeleteEvent, true},
		{"PUT /events/{eventID}/status", handlers.UpdateEventStatus, true},
		{"GET /events/{eventID}/screensaver", handlers.GetScreensaverConfig, false},
		{"PUT /events/{eventID}/screensaver", handlers.UpdateScreensaverConfig, true},
		{"GET /events/{eventID}/screen-state", handlers.GetScreenState, false},
		{"PUT /events/{eventID}/screen-state", handlers.UpdateScreenState, true},
		{"GET /events/{eventID}/screen-stream", handlers.ScreenStream, false},
		{"GET /events/{eventID}/slides", handlers.GetSlides, false},
		{"PUT /events/{eventID}/slides", handlers.SaveSlides, true},
		{"POST /events/{eventID}/sessions", handlers.CreateSession, true},
		{"GET /events/{eventID}/teams", handlers.GetTeams, true},
		{"POST /events/{eventID}/teams", handlers.CreateTeam, true},
		{"POST /events/{eventID}/teams/assign", handlers.AssignParticipant, true},
		{"POST /events/{eventID}/teams/auto-assign", handlers.AutoAssignParticipants, true},
		{"PUT /events/{eventID}/teams/{teamID}", handlers.UpdateTeam, true},
		{"DELETE /events/{eventID}/teams/{teamID}", handlers.DeleteTeam, true},
		{"GET /events/{eventID}/participants", handlers.GetParticipants, true},
		{"POST /events/{eventID}/participants/join", handlers.JoinEvent, false},
		{"POST /events/{eventID}/participants/{participantID}/name", handlers.UpdateParticipantName, false},
		{"POST /events/{eventID}/modules", handlers.AddModule, true},
		{"GET /events/{eventID}/modules/{moduleID}", handlers.GetModule, false},
		{"PUT /events/{eventID}/modules/{moduleID}", handlers.UpdateModule, true},
		{"DELETE /events/{eventID}/modules/{moduleID}", handlers.DeleteModule, true},
		{"PUT /events/{eventID}/modules/{moduleID}/status", handlers.UpdateModuleStatus, true},
		{"PUT /events/{eventID}/modules/{moduleID}/publish", handlers.PublishModule, true},
		{"POST /events/{eventID}/modules/{moduleID}/response", handlers.SubmitModuleResponse, false},
		{"GET /events/{eventID}/modules/{moduleID}/responses", handlers.GetModuleResponses, true},
		{"PUT /events/{eventID}/modules/{moduleID}/responses/{responseID}", handlers.UpdateModuleResponse, true},
		{"PUT /events/{eventID}/modules/{moduleID}/quiz-scores/{userID}", handlers.UpdateQuizScore, true},
		{"PUT /events/{eventID}/modules/{moduleID}/quiz/questions", handlers.UpdateQuizQuestions, true},
		{"GET /events/{eventID}/modules/{moduleID}/quiz/state", handlers.GetQuizState, false},
		{"POST /events/{eventID}/modules/{moduleID}/quiz/start", handlers.StartQuiz, true},
		{"POST /events/{eventID}/modules/{moduleID}/quiz/next", handlers.NextQuestion, true},
		{"GET /events/{eventID}/modules/{moduleID}/quiz/results", handlers.GetQuizResults, false},
		{"POST /events/{eventID}/modules/{moduleID}/quiz/reset", handlers.ResetQuiz, true},
		{"GET /events/{eventID}/modules/{moduleID}/wrongpictures/preload", handlers.PreloadWrongPictures, false},
		{"GET /sessions/{sessionID}", handlers.GetSession, true},
		{"PUT /sessions/{sessionID}/status", handlers.UpdateSessionStatus, true},
		{"PUT /sessions/{sessionID}/active_module", handlers.AdvanceSessionModule, true},
		{"GET /screensavers", handlers.GetScreensavers, false},
		{"POST /screensavers", handlers.CreateScreensaver, true},
		{"DELETE /screensavers/{screensaverID}", handlers.DeleteScreensaver, true},
		{"GET /tracks", handlers.GetTracks, false},
		{"POST /tracks", handlers.CreateTrack, true},
		{"DELETE /tracks/{trackID}", handlers.DeleteTrack, true},
		{"GET /templates", handlers.GetTemplates, false},
		{"POST /templates", handlers.SaveTemplate, true},
		{"PUT /templates/{templateID}", handlers.UpdateTemplate, true},
		{"DELETE /templates/{templateID}", handlers.DeleteTemplate, true},
		{"POST /media/upload", handlers.UploadMedia, true},
		{"GET /media/{mediaID}/file", handlers.ServeMedia, false},
		{"GET /media/{mediaID}", handlers.GetMedia, true},
		{"DELETE /media/{mediaID}", handlers.DeleteMedia, true},
	} {
		handler := route.handler
		if route.admin {
			handler = handlers.VerifyAuth(handler)
		}
		mux.HandleFunc(route.pattern, handler)
		paths[strings.SplitN(route.pattern, " ", 2)[1]] = true
	}

	for path := range paths {
		mux.HandleFunc("OPTIONS "+path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
}
