package handlers

import (
	"net/http"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/services"
)

// Dependencies are created at startup and shared by all HTTP handlers.
type Dependencies struct {
	Auth         *auth.Service
	QuizHub      *hub.QuizHub
	Events       *services.EventService
	Modules      *services.ModuleService
	Participants *services.ParticipantService
	Sessions     *services.EventSessionService
	Templates    *services.TemplateService
	Teams        *database.TeamRepository
}

type Handler struct {
	deps Dependencies
}

func New(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) ServeWs(w http.ResponseWriter, r *http.Request) {
	hub.ServeWs(h.deps.QuizHub, w, r)
}
