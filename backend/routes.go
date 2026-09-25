package main

import (
	"net/http"
	"strings"

	"github.com/user/go-expo-sample/backend/handlers"
)

// registerRoutes keeps the public API paths and authentication requirements in
// one place. Handlers still receive their original URL paths for compatibility.
func registerRoutes(mux *http.ServeMux, h *handlers.Handler) {
	mux.HandleFunc("GET /ws", h.ServeWs)

	paths := make(map[string]bool)
	for _, route := range []struct {
		pattern string
		handler http.HandlerFunc
		admin   bool
	}{
		{"GET /events", h.GetEvents, true},
		{"POST /events", h.CreateEvent, true},
		{"POST /events/connect", h.ConnectToEvent, false},
		{"GET /events/{eventID}", h.GetEvent, false},
		{"PUT /events/{eventID}", h.UpdateEvent, true},
		{"DELETE /events/{eventID}", h.DeleteEvent, true},
		{"PUT /events/{eventID}/status", h.UpdateEventStatus, true},
		{"GET /events/{eventID}/screensaver", h.GetScreensaverConfig, false},
		{"PUT /events/{eventID}/screensaver", h.UpdateScreensaverConfig, true},
		{"GET /events/{eventID}/screen-state", h.GetScreenState, false},
		{"PUT /events/{eventID}/screen-state", h.UpdateScreenState, true},
		{"GET /events/{eventID}/screen-stream", h.ScreenStream, false},
		{"GET /events/{eventID}/slides", h.GetSlides, false},
		{"PUT /events/{eventID}/slides", h.SaveSlides, true},
		{"POST /events/{eventID}/sessions", h.CreateSession, true},
		{"GET /events/{eventID}/teams", h.GetTeams, true},
		{"POST /events/{eventID}/teams", h.CreateTeam, true},
		{"POST /events/{eventID}/teams/assign", h.AssignParticipant, true},
		{"POST /events/{eventID}/teams/auto-assign", h.AutoAssignParticipants, true},
		{"PUT /events/{eventID}/teams/{teamID}", h.UpdateTeam, true},
		{"DELETE /events/{eventID}/teams/{teamID}", h.DeleteTeam, true},
		{"GET /events/{eventID}/scores", h.GetEventScores, true},
		{"PUT /events/{eventID}/modules/{moduleID}/manual-score-target", h.SetManualScoreTarget, true},
		{"PUT /events/{eventID}/modules/{moduleID}/manual-scores/{target}/{targetID}", h.SetManualScore, true},
		{"GET /events/{eventID}/participants", h.GetParticipants, true},
		{"POST /events/{eventID}/participants/join", h.JoinEvent, false},
		{"POST /events/{eventID}/participants/{participantID}/name", h.UpdateParticipantName, false},
		{"POST /events/{eventID}/modules", h.AddModule, true},
		{"GET /events/{eventID}/modules/{moduleID}", h.GetModule, false},
		{"PUT /events/{eventID}/modules/{moduleID}", h.UpdateModule, true},
		{"DELETE /events/{eventID}/modules/{moduleID}", h.DeleteModule, true},
		{"PUT /events/{eventID}/modules/{moduleID}/status", h.UpdateModuleStatus, true},
		{"PUT /events/{eventID}/modules/{moduleID}/publish", h.PublishModule, true},
		{"POST /events/{eventID}/modules/{moduleID}/response", h.SubmitModuleResponse, false},
		{"GET /events/{eventID}/modules/{moduleID}/responses", h.GetModuleResponses, true},
		{"PUT /events/{eventID}/modules/{moduleID}/responses/{responseID}", h.UpdateModuleResponse, true},
		{"PUT /events/{eventID}/modules/{moduleID}/quiz-scores/{userID}", h.UpdateQuizScore, true},
		{"PUT /events/{eventID}/modules/{moduleID}/quiz/questions", h.UpdateQuizQuestions, true},
		{"GET /events/{eventID}/modules/{moduleID}/quiz/state", h.GetQuizState, false},
		{"POST /events/{eventID}/modules/{moduleID}/quiz/start", h.StartQuiz, true},
		{"POST /events/{eventID}/modules/{moduleID}/quiz/next", h.NextQuestion, true},
		{"GET /events/{eventID}/modules/{moduleID}/quiz/results", h.GetQuizResults, false},
		{"POST /events/{eventID}/modules/{moduleID}/quiz/reset", h.ResetQuiz, true},
		{"GET /events/{eventID}/modules/{moduleID}/wrongpictures/preload", handlers.PreloadWrongPictures, false},
		{"GET /sessions/{sessionID}", h.GetSession, true},
		{"PUT /sessions/{sessionID}/status", h.UpdateSessionStatus, true},
		{"PUT /sessions/{sessionID}/active_module", h.AdvanceSessionModule, true},
		{"GET /screensavers", handlers.GetScreensavers, false},
		{"POST /screensavers", handlers.CreateScreensaver, true},
		{"DELETE /screensavers/{screensaverID}", handlers.DeleteScreensaver, true},
		{"GET /tracks", handlers.GetTracks, false},
		{"POST /tracks", handlers.CreateTrack, true},
		{"DELETE /tracks/{trackID}", handlers.DeleteTrack, true},
		{"GET /templates", h.GetTemplates, false},
		{"POST /templates", h.SaveTemplate, true},
		{"PUT /templates/{templateID}", h.UpdateTemplate, true},
		{"DELETE /templates/{templateID}", h.DeleteTemplate, true},
		{"POST /media/upload", handlers.UploadMedia, true},
		{"GET /media/{mediaID}/file", handlers.ServeMedia, false},
		{"GET /media/{mediaID}", handlers.GetMedia, true},
		{"DELETE /media/{mediaID}", handlers.DeleteMedia, true},
	} {
		handler := route.handler
		if route.admin {
			handler = h.VerifyAuth(handler)
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
