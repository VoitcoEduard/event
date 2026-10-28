package services

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/utils"
)

var ErrPINUnavailable = errors.New("could not allocate a unique event PIN")

type EventService struct {
	Repo *database.EventRepository
}

func NewEventService(db *postgres.Store) *EventService {
	return &EventService{Repo: database.NewEventRepository(db)}
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
			if errors.Is(err, database.ErrConflict) {
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

func (s *EventService) GetByModule(ctx context.Context, moduleID string) (*models.Event, error) {
	return s.Repo.GetByModule(ctx, moduleID)
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

func (s *EventService) GetScreensaver(ctx context.Context, id string) (models.ScreensaverConfig, error) {
	event, err := s.Repo.Get(ctx, id)
	if err != nil {
		return models.ScreensaverConfig{}, err
	}
	if event.ScreensaverConfig == nil {
		return models.ScreensaverConfig{}, nil
	}
	return *event.ScreensaverConfig, nil
}

func (s *EventService) SetScreensaver(ctx context.Context, id string, config models.ScreensaverConfig) error {
	return s.Repo.SetScreensaver(ctx, id, config)
}

func (s *EventService) GetSlides(ctx context.Context, id string) ([]models.Slide, error) {
	event, err := s.Repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(event.Slides) > 0 {
		return event.Slides, nil
	}
	pin := event.Pin
	if pin == "" {
		pin = event.ID
	}
	baseURL := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	joinURL := baseURL + "/?pin=" + pin
	slides := []models.Slide{{
		ID: utils.NewID(), Type: "welcome", Name: "Welcome", BackgroundColor: "#1A1A1D",
		Elements: []models.CanvasElement{
			{ID: utils.NewID(), Type: "qr", X: 15, Y: 25, Width: 25, Height: 45, Content: joinURL},
			{ID: utils.NewID(), Type: "text", X: 45, Y: 25, Width: 45, Height: 20, Content: "Join the Game!", FontSize: 70, Color: "#FFFFFF", TextAlign: "left"},
			{ID: utils.NewID(), Type: "text", X: 45, Y: 50, Width: 45, Height: 10, Content: joinURL, FontSize: 45, Color: "#3b82f6", TextAlign: "left"},
			{ID: utils.NewID(), Type: "text", X: 45, Y: 65, Width: 45, Height: 10, Content: "PIN: " + pin, FontSize: 60, Color: "#FFD700", TextAlign: "left"},
		},
	}, {ID: utils.NewID(), Type: "screensaver", Name: "Screensaver", BackgroundColor: "#000000"}}
	if err := s.Repo.SetSlides(ctx, id, slides); err != nil {
		return nil, err
	}
	return slides, nil
}

func (s *EventService) SaveSlides(ctx context.Context, id string, slides []models.Slide) error {
	for i := range slides {
		if slides[i].ID == "" {
			slides[i].ID = utils.NewID()
		}
	}
	return s.Repo.SetSlides(ctx, id, slides)
}

func (s *EventService) ParticipantStatus(ctx context.Context, eventID, userID string) (*models.Participant, float64, error) {
	return s.Repo.ParticipantStatus(ctx, eventID, userID)
}

func (s *EventService) ActiveCards(ctx context.Context, eventID string) ([]models.DynamicCard, error) {
	return s.Repo.ActiveCards(ctx, eventID)
}
