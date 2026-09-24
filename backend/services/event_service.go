package services

import (
	"context"
	"errors"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrPINUnavailable = errors.New("could not allocate a unique event PIN")

type EventService struct {
	Repo *database.EventRepository
}

func NewEventService() *EventService {
	return &EventService{Repo: database.NewEventRepository()}
}

func (s *EventService) Create(ctx context.Context, event *models.Event) error {
	event.ID = utils.NewID()
	event.IsActive = false
	event.CreatedAt = time.Now()
	welcomeLabel := "What is your name?"
	switch event.Language {
	case "ru":
		welcomeLabel = "Как вас зовут?"
	case "ro":
		welcomeLabel = "Cum te numești?"
	}
	event.Modules = []models.Module{{
		ID: utils.NewID(), Type: "form", Name: "Welcome", IsActive: true,
		Data: map[string]any{
			"mandatoryOnEntry": true, "hideAfterCompletion": true,
			"timeline": []map[string]any{{
				"id": "step_welcome_name", "type": "field",
				"data": map[string]any{"label": welcomeLabel, "fieldType": "text", "compactLabel": "name"},
			}},
		},
	}}
	for i := 0; i < 10; i++ {
		pin, err := utils.NewEventPIN()
		if err != nil {
			return err
		}
		exists, err := s.Repo.PINExists(ctx, pin)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		event.Pin = pin
		if err := s.Repo.Create(ctx, event); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return err
		}
		return nil
	}
	return ErrPINUnavailable
}

func (s *EventService) List(ctx context.Context) ([]models.Event, error) {
	return s.Repo.List(ctx)
}

func (s *EventService) Get(ctx context.Context, id string) (*models.Event, error) {
	return s.Repo.Get(ctx, id)
}

func (s *EventService) FindByPIN(ctx context.Context, pin string) (*models.Event, error) {
	return s.Repo.FindByPIN(ctx, pin)
}

func (s *EventService) Update(ctx context.Context, id string, event *models.Event) error {
	return s.Repo.Update(ctx, id, event)
}

func (s *EventService) Delete(ctx context.Context, id string) error {
	return s.Repo.Delete(ctx, id)
}

func (s *EventService) SetStatus(ctx context.Context, id string, active bool) error {
	return s.Repo.SetStatus(ctx, id, active)
}
