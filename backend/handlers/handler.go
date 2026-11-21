package handlers

import (
	"context"
	"net/http"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/filestore"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/services"
)

// Dependencies are created at startup and shared by all HTTP handlers.
type Dependencies struct {
	DB           *postgres.Store
	Files        *filestore.Store
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
	deps                  Dependencies
	checkParticipantProof func(ctx context.Context, eventID, participantID, secret string) (bool, bool, error)
	getEvent              func(ctx context.Context, id string) (*models.Event, error)
}

func New(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) ServeWs(w http.ResponseWriter, r *http.Request) {
	hub.ServeWs(h.deps.QuizHub, w, r)
}
