package services

import (
	"context"
	"errors"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/utils"
)

var ErrModuleNotFound = errors.New("module not found")

type ModuleService struct {
	Repo *database.ModuleRepository
}

func NewModuleService(db *postgres.Store) *ModuleService {
	return &ModuleService{Repo: database.NewModuleRepository(db)}
}

func (s *ModuleService) Add(ctx context.Context, eventID string, module *models.Module) error {
	module.ID = utils.NewID()
	module.IsActive = false
	if module.Data == nil {
		module.Data = make(map[string]any)
	}
	return s.Repo.Add(ctx, eventID, *module)
}

func (s *ModuleService) SetActive(ctx context.Context, eventID, moduleID string, active bool) (bool, error) {
	return s.Repo.SetActive(ctx, eventID, moduleID, active)
}

func (s *ModuleService) Publish(ctx context.Context, eventID, moduleID string, data map[string]any) (bool, error) {
	return s.Repo.Publish(ctx, eventID, moduleID, data)
}

func (s *ModuleService) SubmitResponse(ctx context.Context, eventID, moduleID string, response *models.Response) error {
	response.EventID = eventID
	response.ModuleID = moduleID
	response.CreatedAt = time.Now()
	return s.Repo.InsertResponse(ctx, response)
}

func (s *ModuleService) Delete(ctx context.Context, eventID, moduleID string) (bool, error) {
	module, err := s.Get(ctx, eventID, moduleID)
	if err != nil {
		return false, err
	}
	_ = module
	found, err := s.Repo.Delete(ctx, eventID, moduleID)
	if errors.Is(err, database.ErrHasAwards) {
		return false, teamsError("This Teams activity contains awards; deactivate it instead of deleting it")
	}
	return found, err
}

func (s *ModuleService) UpdateData(ctx context.Context, eventID, moduleID string, data map[string]any) error {
	return s.Repo.UpdateData(ctx, eventID, moduleID, data)
}

func (s *ModuleService) Get(ctx context.Context, eventID, moduleID string) (*models.Module, error) {
	event, err := s.Repo.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	for _, module := range event.Modules {
		if module.ID == moduleID {
			return &module, nil
		}
	}
	return nil, ErrModuleNotFound
}

func (s *ModuleService) Responses(ctx context.Context, eventID, moduleID string) (any, error) {
	module, err := s.Get(ctx, eventID, moduleID)
	if err != nil && !errors.Is(err, ErrModuleNotFound) {
		return nil, err
	}
	if module != nil && module.Type == "teams" {
		activity, activityErr := database.NewTeamsActivityRepository(s.Repo.DB).Get(ctx, eventID, moduleID)
		if activityErr == nil {
			return []any{TeamsActivityResult(activity)}, nil
		}
		if !errors.Is(activityErr, database.ErrNotFound) {
			return nil, activityErr
		}
		return s.Repo.TeamResults(ctx, eventID, moduleID)
	}
	if module != nil && (module.Type == "quiz" || module.Type == "wrong_pictures") {
		return s.Repo.QuizResults(ctx, eventID, moduleID)
	}
	return s.Repo.FormResponses(ctx, eventID, moduleID)
}

func (s *ModuleService) UpdateResponse(ctx context.Context, eventID, moduleID, responseID string, data map[string]any) (bool, error) {
	return s.Repo.UpdateResponse(ctx, eventID, moduleID, responseID, data)
}

func (s *ModuleService) UpdateScore(ctx context.Context, eventID, moduleID, userID string, score int) (bool, error) {
	return s.Repo.UpdateScore(ctx, eventID, moduleID, userID, score)
}
