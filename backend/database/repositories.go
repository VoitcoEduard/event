// Package database exposes application repositories backed by PostgreSQL.
package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

var ErrNotFound = postgres.ErrNotFound
var ErrConflict = postgres.ErrConflict
var ErrInvalidReference = postgres.ErrInvalidReference
var ErrHasAwards = postgres.ErrHasAwards

type EventRepository struct{ DB *postgres.Store }

func NewEventRepository(db *postgres.Store) *EventRepository { return &EventRepository{DB: db} }
func (r *EventRepository) Create(ctx context.Context, e *models.Event) error {
	return r.DB.CreateEvent(ctx, e)
}
func (r *EventRepository) PINExists(ctx context.Context, pin string) (bool, error) {
	return r.DB.PINExists(ctx, pin)
}
func (r *EventRepository) FindByPIN(ctx context.Context, pin string) (*models.Event, error) {
	return r.DB.GetEventByPIN(ctx, pin)
}
func (r *EventRepository) Get(ctx context.Context, id string) (*models.Event, error) {
	return r.DB.GetEvent(ctx, id)
}
func (r *EventRepository) GetByModule(ctx context.Context, id string) (*models.Event, error) {
	return r.DB.GetEventByModule(ctx, id)
}
func (r *EventRepository) List(ctx context.Context) ([]models.Event, error) {
	return r.DB.ListEvents(ctx)
}
func (r *EventRepository) Update(ctx context.Context, id string, e *models.Event) error {
	return r.DB.UpdateEvent(ctx, id, e)
}
func (r *EventRepository) Delete(ctx context.Context, id string) error {
	return r.DB.DeleteEvent(ctx, id)
}
func (r *EventRepository) SetStatus(ctx context.Context, id string, active bool) error {
	return r.DB.SetEventStatus(ctx, id, active)
}
func (r *EventRepository) SetScreensaver(ctx context.Context, id string, c models.ScreensaverConfig) error {
	return r.DB.SetEventScreensaver(ctx, id, c)
}
func (r *EventRepository) SetSlides(ctx context.Context, id string, v []models.Slide) error {
	return r.DB.SetEventSlides(ctx, id, v)
}
func (r *EventRepository) ParticipantStatus(ctx context.Context, eventID, id string) (*models.Participant, float64, error) {
	p, err := r.DB.Participant(ctx, eventID, id)
	if err != nil {
		return nil, 0, err
	}
	scores, err := LoadEventScores(ctx, r.DB, eventID)
	if err != nil {
		return p, 0, err
	}
	return p, scores.PlayerTotal(id), nil
}
func (r *EventRepository) ActiveCards(ctx context.Context, eventID string) ([]models.DynamicCard, error) {
	return r.DB.ActiveCards(ctx, eventID)
}

type ModuleRepository struct{ DB *postgres.Store }

func NewModuleRepository(db *postgres.Store) *ModuleRepository { return &ModuleRepository{DB: db} }
func (r *ModuleRepository) Add(ctx context.Context, eventID string, m models.Module) error {
	return r.DB.AddModule(ctx, eventID, m)
}
func (r *ModuleRepository) SetActive(ctx context.Context, eventID, moduleID string, active bool) (bool, error) {
	return r.DB.SetModuleActive(ctx, eventID, moduleID, active)
}
func (r *ModuleRepository) Publish(ctx context.Context, eventID, moduleID string, data map[string]any) (bool, error) {
	return r.DB.PublishModule(ctx, eventID, moduleID, data)
}
func (r *ModuleRepository) Delete(ctx context.Context, eventID, moduleID string) (bool, error) {
	return r.DB.DeleteModule(ctx, eventID, moduleID)
}
func (r *ModuleRepository) UpdateData(ctx context.Context, eventID, moduleID string, data map[string]any) error {
	return r.DB.PatchModule(ctx, eventID, moduleID, data)
}
func (r *ModuleRepository) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	return r.DB.GetEvent(ctx, id)
}
func (r *ModuleRepository) InsertResponse(ctx context.Context, v *models.Response) error {
	return r.DB.InsertResponse(ctx, v)
}
func (r *ModuleRepository) TeamResults(ctx context.Context, eventID, moduleID string) ([]models.TeamResult, error) {
	return r.DB.TeamResults(ctx, eventID, moduleID)
}
func (r *ModuleRepository) QuizResults(ctx context.Context, eventID, moduleID string) ([]models.QuizResult, error) {
	return r.DB.QuizResults(ctx, eventID, moduleID)
}
func (r *ModuleRepository) FormResponses(ctx context.Context, eventID, moduleID string) ([]models.Response, error) {
	return r.DB.Responses(ctx, eventID, moduleID)
}
func (r *ModuleRepository) UpdateResponse(ctx context.Context, eventID, moduleID, responseID string, data map[string]any) (bool, error) {
	return r.DB.UpdateResponse(ctx, eventID, moduleID, responseID, data)
}
func (r *ModuleRepository) UpdateScore(ctx context.Context, eventID, moduleID, userID string, score int) (bool, error) {
	return r.DB.UpdateLatestScore(ctx, eventID, moduleID, userID, score)
}

type ParticipantRepository struct{ DB *postgres.Store }

func NewParticipantRepository(db *postgres.Store) *ParticipantRepository {
	return &ParticipantRepository{DB: db}
}
func (r *ParticipantRepository) Create(ctx context.Context, p *models.Participant) error {
	v, err := r.DB.EnsureParticipant(ctx, p)
	if err == nil {
		*p = *v
	}
	return err
}
func (r *ParticipantRepository) GetByID(ctx context.Context, eventID, id string) (*models.Participant, error) {
	return r.DB.Participant(ctx, eventID, id)
}
func (r *ParticipantRepository) GetByEventID(ctx context.Context, eventID string) ([]models.Participant, error) {
	return r.DB.Participants(ctx, eventID)
}
func (r *ParticipantRepository) Update(ctx context.Context, p *models.Participant) error {
	return r.DB.UpdateParticipant(ctx, p)
}

type TeamRepository struct{ DB *postgres.Store }

func NewTeamRepository(db *postgres.Store) *TeamRepository { return &TeamRepository{DB: db} }
func (r *TeamRepository) Create(ctx context.Context, t *models.Team) error {
	return r.DB.CreateTeam(ctx, t)
}
func (r *TeamRepository) GetByID(ctx context.Context, id string) (*models.Team, error) {
	return r.DB.Team(ctx, id)
}
func (r *TeamRepository) GetByEventID(ctx context.Context, id string) ([]models.Team, error) {
	return r.DB.Teams(ctx, id)
}
func (r *TeamRepository) Update(ctx context.Context, t *models.Team) error {
	return r.DB.UpdateTeam(ctx, t)
}
func (r *TeamRepository) Delete(ctx context.Context, id string) error {
	return r.DB.DeleteTeam(ctx, id)
}
func (r *TeamRepository) DeleteAllForEvent(ctx context.Context, eventID string) error {
	return r.DB.DeleteTeamsForEvent(ctx, eventID)
}

type EventSessionRepository struct{ DB *postgres.Store }

func NewEventSessionRepository(db *postgres.Store) *EventSessionRepository {
	return &EventSessionRepository{DB: db}
}
func (r *EventSessionRepository) Create(ctx context.Context, v *models.EventSession) error {
	return r.DB.CreateEventSession(ctx, v)
}
func (r *EventSessionRepository) GetByID(ctx context.Context, id string) (*models.EventSession, error) {
	return r.DB.EventSession(ctx, id)
}
func (r *EventSessionRepository) GetByEventID(ctx context.Context, id string) ([]models.EventSession, error) {
	return r.DB.EventSessions(ctx, id)
}
func (r *EventSessionRepository) Update(ctx context.Context, v *models.EventSession) error {
	return r.DB.UpdateEventSession(ctx, v)
}

// Used to preserve and verify the existing partial-update semantics.
func buildEventSessionUpdate(v *models.EventSession) map[string]any {
	m := map[string]any{"status": v.Status, "updatedAt": v.UpdatedAt}
	if v.ActiveModuleID != "" {
		m["activeModuleId"] = v.ActiveModuleID
	}
	if v.ModuleOrder != nil {
		m["moduleOrder"] = v.ModuleOrder
	}
	if v.ModuleStates != nil {
		m["moduleStates"] = v.ModuleStates
	}
	if v.StartedAt != nil {
		m["startedAt"] = v.StartedAt
	}
	if v.EndedAt != nil {
		m["endedAt"] = v.EndedAt
	}
	return m
}

type ModuleAttemptRepository struct{ DB *postgres.Store }

func NewModuleAttemptRepository(db *postgres.Store) *ModuleAttemptRepository {
	return &ModuleAttemptRepository{DB: db}
}
func (r *ModuleAttemptRepository) Create(ctx context.Context, v *models.ModuleAttempt) error {
	return r.DB.CreateAttempt(ctx, v)
}
func (r *ModuleAttemptRepository) GetByID(ctx context.Context, id string) (*models.ModuleAttempt, error) {
	return r.DB.Attempt(ctx, id)
}
func (r *ModuleAttemptRepository) GetByParticipantID(ctx context.Context, eventID, id string) ([]models.ModuleAttempt, error) {
	return r.DB.AttemptsForParticipant(ctx, eventID, id)
}
func (r *ModuleAttemptRepository) Update(ctx context.Context, v *models.ModuleAttempt) error {
	return r.DB.UpdateAttempt(ctx, v)
}

type SharedStateRepository struct{ DB *postgres.Store }

func NewSharedStateRepository(db *postgres.Store) *SharedStateRepository {
	return &SharedStateRepository{DB: db}
}
func (r *SharedStateRepository) Create(ctx context.Context, v *models.SharedStateValue) error {
	return r.DB.CreateSharedState(ctx, v)
}
func (r *SharedStateRepository) GetByKey(ctx context.Context, sessionID, key string) (*models.SharedStateValue, error) {
	return r.DB.SharedStateByKey(ctx, sessionID, key)
}
func (r *SharedStateRepository) GetBySessionID(ctx context.Context, id string) ([]models.SharedStateValue, error) {
	return r.DB.SharedStateBySession(ctx, id)
}
func (r *SharedStateRepository) Update(ctx context.Context, v *models.SharedStateValue) error {
	return r.DB.UpdateSharedState(ctx, v)
}

type TemplateRepository struct{ DB *postgres.Store }

func NewTemplateRepository(db *postgres.Store) *TemplateRepository {
	return &TemplateRepository{DB: db}
}
func (r *TemplateRepository) Create(ctx context.Context, v *models.Template) error {
	return r.DB.CreateTemplate(ctx, v)
}
func (r *TemplateRepository) List(ctx context.Context) ([]models.Template, error) {
	return r.DB.Templates(ctx)
}
func (r *TemplateRepository) Delete(ctx context.Context, id string) error {
	return r.DB.DeleteTemplate(ctx, id)
}
func (r *TemplateRepository) Update(ctx context.Context, id string, v *models.Template) error {
	return r.DB.UpdateTemplate(ctx, id, v)
}

type QuizResultRepository struct{ DB *postgres.Store }

func NewQuizResultRepository(db *postgres.Store) *QuizResultRepository {
	return &QuizResultRepository{DB: db}
}
func (r *QuizResultRepository) EventForModule(ctx context.Context, id string) (*models.Event, error) {
	return r.DB.GetEventByModule(ctx, id)
}
func (r *QuizResultRepository) ParticipantsForEvent(ctx context.Context, eventID string) ([]models.Participant, error) {
	return r.DB.Participants(ctx, eventID)
}
func (r *QuizResultRepository) Create(ctx context.Context, v *models.QuizResult) error {
	if v.RunID == "" {
		v.RunID = uuid.NewString()
	}
	return r.DB.SaveQuizResult(ctx, v.RunID, v)
}

type TeamResultRepository struct{ DB *postgres.Store }

func NewTeamResultRepository(db *postgres.Store) *TeamResultRepository {
	return &TeamResultRepository{DB: db}
}
func (r *TeamResultRepository) Create(ctx context.Context, v *models.TeamResult) error {
	v.FinishedAt = time.Now().UTC()
	return r.DB.SaveTeamResult(ctx, v)
}
func (r *TeamResultRepository) GetByEventModule(ctx context.Context, eventID, moduleID string) (*models.TeamResult, error) {
	v, err := r.DB.TeamResults(ctx, eventID, moduleID)
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return nil, ErrNotFound
	}
	return &v[0], nil
}
func (r *TeamResultRepository) GetByEventID(ctx context.Context, eventID string) ([]models.TeamResult, error) {
	return r.DB.TeamResults(ctx, eventID, "")
}
func (r *TeamResultRepository) DeleteByModuleID(ctx context.Context, id string) error {
	return r.DB.DeleteTeamResults(ctx, id)
}

type ScoreEntryRepository struct{ DB *postgres.Store }

func NewScoreEntryRepository(db *postgres.Store) *ScoreEntryRepository {
	return &ScoreEntryRepository{DB: db}
}
func (r *ScoreEntryRepository) Create(ctx context.Context, v *models.ScoreEntry) error {
	return r.DB.SaveScore(ctx, v, false)
}
func (r *ScoreEntryRepository) GetByEventID(ctx context.Context, id string) ([]models.ScoreEntry, error) {
	return r.DB.ScoreEntries(ctx, id)
}
func (r *ScoreEntryRepository) GetByParticipantID(ctx context.Context, eventID, id string) ([]models.ScoreEntry, error) {
	scores, err := r.DB.ScoreEntries(ctx, eventID)
	if err != nil {
		return nil, err
	}
	out := []models.ScoreEntry{}
	for _, v := range scores {
		if v.ParticipantID == id {
			out = append(out, v)
		}
	}
	return out, nil
}
func (r *ScoreEntryRepository) SetManual(ctx context.Context, eventID, moduleID, target, targetID string, points float64) error {
	return r.DB.SetManualScore(ctx, eventID, moduleID, target, targetID, points)
}

type TeamsActivityRepository struct{ DB *postgres.Store }

func NewTeamsActivityRepository(db *postgres.Store) *TeamsActivityRepository {
	return &TeamsActivityRepository{DB: db}
}
func (r *TeamsActivityRepository) Get(ctx context.Context, eventID, moduleID string) (*models.TeamsActivity, error) {
	return r.DB.TeamsActivity(ctx, eventID, moduleID)
}
func (r *TeamsActivityRepository) Create(ctx context.Context, v *models.TeamsActivity) error {
	return r.DB.CreateTeamsActivity(ctx, v)
}
func (r *TeamsActivityRepository) CompareAndSwap(ctx context.Context, v *models.TeamsActivity, revision int64) (bool, error) {
	return r.DB.CompareAndSwapTeamsActivity(ctx, v, revision)
}
func (r *TeamsActivityRepository) ByEvent(ctx context.Context, eventID string) ([]models.TeamsActivity, error) {
	return r.DB.TeamsActivities(ctx, eventID)
}

// EncodeJSON is shared by repository clients that handle polymorphic values.
func EncodeJSON(v any) ([]byte, error) { return json.Marshal(v) }
