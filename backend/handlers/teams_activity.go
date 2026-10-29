package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"github.com/user/go-expo-sample/backend/utils"
)

func teamsIDs(r *http.Request) (string, string) {
	return r.PathValue("eventID"), r.PathValue("moduleID")
}
func (h *Handler) teamsModule(r *http.Request) (*models.Module, error) {
	e, m := teamsIDs(r)
	module, err := h.deps.Modules.Get(r.Context(), e, m)
	if err == nil && module.Type != models.ModuleTypeTeams {
		return nil, services.ErrModuleNotFound
	}
	return module, err
}
func teamsFailure(w http.ResponseWriter, err error) {
	var validation *services.TeamsError
	code := http.StatusInternalServerError
	if errors.As(err, &validation) || errors.Is(err, services.ErrTeamsConflict) {
		code = http.StatusConflict
	}
	if errors.Is(err, database.ErrNotFound) || errors.Is(err, services.ErrModuleNotFound) {
		code = http.StatusNotFound
	}
	if code == http.StatusInternalServerError {
		writeRequestError(w, err, code)
		return
	}
	http.Error(w, err.Error(), code)
}
func (h *Handler) teamsService() *services.TeamsActivityService {
	return &services.TeamsActivityService{Store: database.NewTeamsActivityRepository(h.deps.DB)}
}
func (h *Handler) teamsChanged(a *models.TeamsActivity) {
	invalidateEventCache(a.EventID)
	notifyTeamsSubscribers(a.EventID, a.ModuleID, a.Revision)
	if h.deps.QuizHub != nil {
		h.deps.QuizHub.BroadcastToEvent(a.EventID, hub.ServerMessage{Type: "teams:changed", Payload: map[string]any{"eventId": a.EventID, "moduleId": a.ModuleID, "revision": a.Revision}})
	}
}

func (h *Handler) setTeamsModuleActive(ctx context.Context, eventID, moduleID string, active bool) error {
	found, err := h.deps.Modules.SetActive(ctx, eventID, moduleID, active)
	if err != nil {
		return err
	}
	if !found {
		return services.ErrModuleNotFound
	}
	invalidateEventCache(eventID)
	return nil
}

// Participants share the same event-scoped identity across all workflows.
func (h *Handler) teamsParticipants(ctx context.Context, eventID string) ([]models.TeamsEntrant, error) {
	people, err := h.deps.DB.Participants(ctx, eventID)
	if err != nil {
		return nil, err
	}
	out := make([]models.TeamsEntrant, 0, len(people))
	for _, p := range people {
		out = append(out, models.TeamsEntrant{ParticipantID: p.ID, Name: p.DisplayName, TeamID: p.TeamID})
	}
	return out, nil
}
func (h *Handler) resolveTeamsParticipant(ctx context.Context, eventID, userID, name string, create bool) (models.TeamsEntrant, error) {
	if userID == "" || len(userID) > 120 {
		return models.TeamsEntrant{}, &services.TeamsError{Message: "A device user ID is required"}
	}
	p, err := h.deps.DB.Participant(ctx, eventID, userID)
	if errors.Is(err, database.ErrNotFound) && create {
		name = strings.TrimSpace(name)
		if name == "" || len(name) > 120 {
			return models.TeamsEntrant{}, &services.TeamsError{Message: "Enter your name before joining"}
		}
		now := time.Now().UTC()
		p, err = h.deps.DB.EnsureParticipant(ctx, &models.Participant{ID: userID, EventID: eventID, DisplayName: name, NormalizedName: strings.ToLower(name), Status: models.ParticipantStatusJoined, CreatedAt: now, UpdatedAt: now})
	}
	if err != nil {
		return models.TeamsEntrant{}, err
	}
	return models.TeamsEntrant{ParticipantID: p.ID, Name: p.DisplayName, TeamID: p.TeamID}, nil
}

type publicTeamsMember struct {
	Name string `json:"name"`
}
type publicTeamsRow struct {
	models.ActivityTeam
	Score   int                 `json:"score"`
	Rank    int                 `json:"rank"`
	Members []publicTeamsMember `json:"members"`
	Full    bool                `json:"full"`
}
type publicTeamAward struct {
	ID         string    `json:"id"`
	TeamID     string    `json:"teamId"`
	Points     int       `json:"points"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"createdAt"`
	ReversesID string    `json:"reversesId,omitempty"`
	ForYou     bool      `json:"forYou"`
}

func publicTeamsState(a *models.TeamsActivity, userID string, total float64) map[string]any {
	teams := []publicTeamsRow{}
	for _, row := range a.Standings() {
		members := []publicTeamsMember{}
		for _, p := range row.Members {
			members = append(members, publicTeamsMember{p.Name})
		}
		cap := row.Capacity
		if cap == 0 {
			cap = a.Config.Capacity
		}
		teams = append(teams, publicTeamsRow{row.ActivityTeam, row.Score, row.Rank, members, cap > 0 && len(members) >= cap})
	}
	awards := []publicTeamAward{}
	for _, award := range a.Awards {
		forYou := false
		for _, id := range award.Recipients {
			if userID != "" && id == userID {
				forYou = true
			}
		}
		awards = append(awards, publicTeamAward{award.ID, award.TeamID, award.Points, award.Reason, award.CreatedAt, award.ReversesID, forYou})
	}
	_, scores := a.Scores()
	joined := false
	teamID := ""
	for _, p := range a.Entrants {
		if p.ParticipantID == userID {
			joined = true
			teamID = p.TeamID
		}
	}
	unassigned := []publicTeamsMember{}
	for _, p := range a.Entrants {
		if p.TeamID == "" {
			unassigned = append(unassigned, publicTeamsMember{p.Name})
		}
	}
	return map[string]any{"initialized": true, "eventId": a.EventID, "moduleId": a.ModuleID, "name": a.Name, "status": a.Status, "revision": a.Revision, "method": a.Config.Method, "teams": teams, "awards": awards, "unassigned": unassigned, "joined": joined, "myTeamId": teamID, "myPoints": scores[userID], "totalPoints": total, "entrantCount": len(a.Entrants), "finishedAt": a.FinishedAt}
}
func (h *Handler) GetTeamsActivity(w http.ResponseWriter, r *http.Request) {
	module, err := h.teamsModule(r)
	if err != nil {
		teamsFailure(w, err)
		return
	}
	e, m := teamsIDs(r)
	a, err := database.NewTeamsActivityRepository(h.deps.DB).Get(r.Context(), e, m)
	if errors.Is(err, database.ErrNotFound) {
		writeJSON(w, map[string]any{"initialized": false, "name": module.Name, "teams": []any{}, "awards": []any{}})
		return
	}
	if err != nil {
		teamsFailure(w, err)
		return
	}
	userID := r.Header.Get("X-Teams-Player")
	total := 0.0
	if userID != "" {
		scores, err := database.LoadEventScores(r.Context(), h.deps.DB, e)
		if err != nil {
			teamsFailure(w, err)
			return
		}
		total = scores.PlayerTotal(userID)
	}
	writeJSON(w, publicTeamsState(a, userID, total))
}
func (h *Handler) GetTeamsActivityAdmin(w http.ResponseWriter, r *http.Request) {
	module, err := h.teamsModule(r)
	if err != nil {
		teamsFailure(w, err)
		return
	}
	e, m := teamsIDs(r)
	a, err := database.NewTeamsActivityRepository(h.deps.DB).Get(r.Context(), e, m)
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		teamsFailure(w, err)
		return
	}
	people, err := h.teamsParticipants(r.Context(), e)
	if err != nil {
		teamsFailure(w, err)
		return
	}
	legacy, err := h.deps.Teams.GetByEventID(r.Context(), e)
	if err != nil {
		teamsFailure(w, err)
		return
	}
	if a.ID == "" {
		a = nil
	}
	writeJSON(w, map[string]any{"activity": a, "module": module, "participants": people, "legacyTeams": legacy})
}
func (h *Handler) JoinTeamsActivity(w http.ResponseWriter, r *http.Request) {
	if _, err := h.teamsModule(r); err != nil {
		teamsFailure(w, err)
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	e, m := teamsIDs(r)
	// Validate the activity before creating a live participant.
	existing, err := database.NewTeamsActivityRepository(h.deps.DB).Get(r.Context(), e, m)
	if err != nil {
		teamsFailure(w, err)
		return
	}
	id := r.Header.Get("X-Teams-Player")
	if existing.Status == "completed" {
		found := false
		for _, p := range existing.Entrants {
			if p.ParticipantID == id {
				found = true
			}
		}
		if !found {
			teamsFailure(w, &services.TeamsError{Message: "This activity has completed"})
			return
		}
	}
	p, err := h.resolveTeamsParticipant(r.Context(), e, id, input.Name, true)
	if err != nil {
		teamsFailure(w, err)
		return
	}
	a, err := h.teamsService().Mutate(r.Context(), e, m, func(a *models.TeamsActivity) error { return services.AddTeamsEntrant(a, p) })
	if err != nil {
		teamsFailure(w, err)
		return
	}
	h.teamsChanged(a)
	h.GetTeamsActivity(w, r)
}
func (h *Handler) ChooseTeamsActivity(w http.ResponseWriter, r *http.Request) {
	if _, err := h.teamsModule(r); err != nil {
		teamsFailure(w, err)
		return
	}
	var input struct {
		TeamID string `json:"teamId"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	e, m := teamsIDs(r)
	id := r.Header.Get("X-Teams-Player")
	if id == "" {
		http.Error(w, "Device identity required", 403)
		return
	}
	a, err := h.teamsService().Mutate(r.Context(), e, m, func(a *models.TeamsActivity) error { return services.AssignTeamsEntrant(a, id, input.TeamID, true) })
	if err != nil {
		teamsFailure(w, err)
		return
	}
	h.teamsChanged(a)
	h.GetTeamsActivity(w, r)
}

type teamsAction struct {
	Action        string              `json:"action"`
	Config        *models.TeamsConfig `json:"config"`
	ImportLegacy  bool                `json:"importLegacy"`
	Team          models.ActivityTeam `json:"team"`
	TeamID        string              `json:"teamId"`
	ParticipantID string              `json:"participantId"`
	Reshuffle     bool                `json:"reshuffle"`
	Points        *float64            `json:"points"`
	Reason        string              `json:"reason"`
	RequestID     string              `json:"requestId"`
	AwardID       string              `json:"awardId"`
}

func (h *Handler) TeamsActivityAction(w http.ResponseWriter, r *http.Request) {
	module, err := h.teamsModule(r)
	if err != nil {
		teamsFailure(w, err)
		return
	}
	var input teamsAction
	if err := decodeJSON(w, r, &input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	e, m := teamsIDs(r)
	actorID := ""
	if h.deps.Auth == nil {
		http.Error(w, "Authentication unavailable", 503)
		return
	}
	_, actor, ok := h.deps.Auth.Authorize(w, r, false)
	if !ok {
		return
	}
	actorID = actor.ID
	if input.Action == "initialize" {
		h.initializeTeams(w, r, module, input)
		return
	}
	var entrant models.TeamsEntrant
	if input.Action == "add_participant" {
		entrant, err = h.resolveTeamsParticipant(r.Context(), e, input.ParticipantID, "", false)
		if err != nil {
			teamsFailure(w, err)
			return
		}
	}
	if input.Team.ID == "" {
		input.Team.ID = utils.NewID()
	}
	awardID := utils.NewID()
	a, err := h.teamsService().Mutate(r.Context(), e, m, func(a *models.TeamsActivity) error {
		switch input.Action {
		case "configure":
			if a.Status != "setup" {
				return &services.TeamsError{Message: "Configuration can change only during setup"}
			}
			if input.Config == nil {
				return &services.TeamsError{Message: "Configuration is required"}
			}
			if err := services.ValidateTeamsConfig(input.Config); err != nil {
				return err
			}
			if input.Config.SourceModuleID != a.Config.SourceModuleID {
				return &services.TeamsError{Message: "Choose the team source when initializing the activity"}
			}
			a.Config = *input.Config
			for _, t := range a.Teams {
				if err := services.SaveActivityTeam(a, t); err != nil {
					return err
				}
			}
		case "save_team":
			return services.SaveActivityTeam(a, input.Team)
		case "delete_team":
			return services.DeleteActivityTeam(a, input.TeamID)
		case "add_participant":
			return services.AddTeamsEntrant(a, entrant)
		case "assign":
			return services.AssignTeamsEntrant(a, input.ParticipantID, input.TeamID, false)
		case "distribute":
			return services.DistributeTeams(a, input.Reshuffle)
		case "start", "complete", "reopen":
			return services.TransitionTeams(a, input.Action)
		case "award":
			points, err := services.WholeTeamsPoints(input.Points)
			if err != nil {
				return err
			}
			return services.AwardTeam(a, models.TeamAward{ID: awardID, RequestID: input.RequestID, TeamID: input.TeamID, Points: points, Reason: input.Reason, ActorID: actorID})
		case "undo":
			return services.UndoTeamAward(a, input.AwardID, input.RequestID, actorID, awardID)
		default:
			return &services.TeamsError{Message: "Unknown teams action"}
		}
		return nil
	})
	if err != nil {
		teamsFailure(w, err)
		return
	}
	if input.Action == "start" || input.Action == "reopen" || input.Action == "complete" {
		active := input.Action != "complete"
		if err := h.setTeamsModuleActive(r.Context(), e, m, active); err != nil {
			teamsFailure(w, err)
			return
		}
	}
	h.teamsChanged(a)
	writeJSON(w, a)
}
func (h *Handler) initializeTeams(w http.ResponseWriter, r *http.Request, module *models.Module, input teamsAction) {
	e, m := teamsIDs(r)
	repo := database.NewTeamsActivityRepository(h.deps.DB)
	existing, err := repo.Get(r.Context(), e, m)
	if err == nil {
		if err = h.setTeamsModuleActive(r.Context(), e, m, true); err != nil {
			teamsFailure(w, err)
			return
		}
		writeJSON(w, existing)
		return
	}
	if !errors.Is(err, database.ErrNotFound) {
		teamsFailure(w, err)
		return
	}
	config := models.TeamsConfig{}
	if input.Config != nil {
		config = *input.Config
	} else if raw, ok := module.Data["teamsConfig"]; ok {
		data, _ := json.Marshal(raw)
		if err = json.Unmarshal(data, &config); err != nil {
			http.Error(w, "Invalid teams configuration", 400)
			return
		}
	}
	if err = services.ValidateTeamsConfig(&config); err != nil {
		teamsFailure(w, err)
		return
	}
	a := services.NewTeamsActivity(e, m, module.Name, config)
	if config.SourceModuleID != "" {
		event, err := h.deps.Events.Get(r.Context(), e)
		if err != nil {
			teamsFailure(w, err)
			return
		}
		currentIndex, sourceIndex := -1, -1
		for i, mod := range event.Modules {
			if mod.ID == m {
				currentIndex = i
			}
			if mod.ID == config.SourceModuleID && mod.Type == models.ModuleTypeTeams {
				sourceIndex = i
			}
		}
		if sourceIndex < 0 || sourceIndex >= currentIndex {
			http.Error(w, "Reuse an earlier Teams activity in this event", 400)
			return
		}
		source, err := repo.Get(r.Context(), e, config.SourceModuleID)
		if err != nil {
			teamsFailure(w, err)
			return
		}
		a.Teams = append(a.Teams, source.Teams...)
		a.SourceMembers = append(a.SourceMembers, source.Entrants...)
	}
	if input.ImportLegacy {
		if config.SourceModuleID != "" {
			http.Error(w, "Choose either reuse or legacy import", 400)
			return
		}
		teams, err := h.deps.Teams.GetByEventID(r.Context(), e)
		if err != nil {
			teamsFailure(w, err)
			return
		}
		for _, team := range teams {
			a.Teams = append(a.Teams, models.ActivityTeam{ID: team.ID, Name: team.Name, Color: team.Color, Capacity: config.Capacity})
		}
		people, err := h.teamsParticipants(r.Context(), e)
		if err != nil {
			teamsFailure(w, err)
			return
		}
		for _, p := range people {
			if p.TeamID != "" && services.TeamIndex(a, p.TeamID) >= 0 {
				a.SourceMembers = append(a.SourceMembers, p)
			}
		}
	}
	if err = repo.Create(r.Context(), a); err != nil {
		if errors.Is(err, database.ErrConflict) {
			existing, err = repo.Get(r.Context(), e, m)
			if err == nil {
				if err = h.setTeamsModuleActive(r.Context(), e, m, true); err != nil {
					teamsFailure(w, err)
					return
				}
				writeJSON(w, existing)
				return
			}
		}
		teamsFailure(w, err)
		return
	}
	if err = h.setTeamsModuleActive(r.Context(), e, m, true); err != nil {
		teamsFailure(w, err)
		return
	}
	h.teamsChanged(a)
	writeJSON(w, a)
}
