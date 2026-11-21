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

type EventStore interface {
	Create(ctx context.Context, e *models.Event) error
	PINExists(ctx context.Context, pin string) (bool, error)
	FindByPIN(ctx context.Context, pin string) (*models.Event, error)
	Get(ctx context.Context, id string) (*models.Event, error)
	GetByModule(ctx context.Context, id string) (*models.Event, error)
	List(ctx context.Context) ([]models.Event, error)
	Update(ctx context.Context, id string, e *models.Event) error
	Delete(ctx context.Context, id string) error
	SetStatus(ctx context.Context, id string, active bool) error
	SetScreensaver(ctx context.Context, id string, c models.ScreensaverConfig) error
	SetSlides(ctx context.Context, id string, v []models.Slide) error
	ParticipantStatus(ctx context.Context, eventID, id string) (*models.Participant, float64, error)
	ActiveCards(ctx context.Context, eventID string) ([]models.DynamicCard, error)
}

type EventService struct {
	Repo  *database.EventRepository
	store EventStore
}

func NewEventService(db *postgres.Store) *EventService {
	repo := database.NewEventRepository(db)
	return &EventService{Repo: repo, store: repo}
}

func (s *EventService) db() EventStore {
	if s.store != nil {
		return s.store
	}
	return s.Repo
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
		exists, err := s.db().PINExists(ctx, pin)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		event.Pin = pin
		if event.Slides == nil {
			event.Slides = defaultSlidesForEvent(event)
		}
		if err := s.db().Create(ctx, event); err != nil {
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
	return s.db().List(ctx)
}

func (s *EventService) Get(ctx context.Context, id string) (*models.Event, error) {
	return s.db().Get(ctx, id)
}

func (s *EventService) GetByModule(ctx context.Context, moduleID string) (*models.Event, error) {
	return s.db().GetByModule(ctx, moduleID)
}

func (s *EventService) FindByPIN(ctx context.Context, pin string) (*models.Event, error) {
	return s.db().FindByPIN(ctx, pin)
}

func (s *EventService) Update(ctx context.Context, id string, event *models.Event) error {
	return s.db().Update(ctx, id, event)
}

func (s *EventService) Delete(ctx context.Context, id string) error {
	return s.db().Delete(ctx, id)
}

func (s *EventService) SetStatus(ctx context.Context, id string, active bool) error {
	return s.db().SetStatus(ctx, id, active)
}

func (s *EventService) GetScreensaver(ctx context.Context, id string) (models.ScreensaverConfig, error) {
	event, err := s.db().Get(ctx, id)
	if err != nil {
		return models.ScreensaverConfig{}, err
	}
	if event.ScreensaverConfig == nil {
		return models.ScreensaverConfig{}, nil
	}
	return *event.ScreensaverConfig, nil
}

func (s *EventService) SetScreensaver(ctx context.Context, id string, config models.ScreensaverConfig) error {
	return s.db().SetScreensaver(ctx, id, config)
}

func defaultSlidesForEvent(event *models.Event) []models.Slide {
	pin := event.Pin
	if pin == "" {
		pin = event.ID
	}
	baseURL := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	joinURL := baseURL + "/?pin=" + pin
	return []models.Slide{{
		ID: "default-welcome-" + event.ID, Type: "welcome", Name: "Welcome", BackgroundColor: "#1A1A1D",
		Elements: []models.CanvasElement{
			{ID: "default-welcome-qr-" + event.ID, Type: "qr", X: 15, Y: 25, Width: 25, Height: 45, Content: joinURL},
			{ID: "default-welcome-title-" + event.ID, Type: "text", X: 45, Y: 25, Width: 45, Height: 20, Content: "Join the Game!", FontSize: 70, Color: "#FFFFFF", TextAlign: "left"},
			{ID: "default-welcome-url-" + event.ID, Type: "text", X: 45, Y: 50, Width: 45, Height: 10, Content: joinURL, FontSize: 45, Color: "#3b82f6", TextAlign: "left"},
			{ID: "default-welcome-pin-" + event.ID, Type: "text", X: 45, Y: 65, Width: 45, Height: 10, Content: "PIN: " + pin, FontSize: 60, Color: "#FFD700", TextAlign: "left"},
		},
	}, {ID: "default-screensaver-" + event.ID, Type: "screensaver", Name: "Screensaver", BackgroundColor: "#000000"}}
}

// GetSlides is strictly read-only: if slides have never been saved (nil), it
// computes default slides in memory without writing to the database; if an
// admin explicitly saved slides (including an empty slice []), it returns them as-is.
func (s *EventService) GetSlides(ctx context.Context, id string) ([]models.Slide, error) {
	event, err := s.db().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if event.Slides != nil {
		return event.Slides, nil
	}
	return defaultSlidesForEvent(event), nil
}

func (s *EventService) SaveSlides(ctx context.Context, id string, slides []models.Slide) error {
	if slides == nil {
		slides = []models.Slide{}
	}
	for i := range slides {
		if slides[i].ID == "" {
			slides[i].ID = utils.NewID()
		}
	}
	return s.db().SetSlides(ctx, id, slides)
}

func (s *EventService) ParticipantStatus(ctx context.Context, eventID, userID string) (*models.Participant, float64, error) {
	return s.db().ParticipantStatus(ctx, eventID, userID)
}

func (s *EventService) ActiveCards(ctx context.Context, eventID string) ([]models.DynamicCard, error) {
	return s.db().ActiveCards(ctx, eventID)
}
