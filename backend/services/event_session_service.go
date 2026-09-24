package services

import (
	"context"
	"errors"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
)

type EventSessionService struct {
	SessionRepo *database.EventSessionRepository
}

func NewEventSessionService() *EventSessionService {
	return &EventSessionService{
		SessionRepo: database.NewEventSessionRepository(),
	}
}

func (s *EventSessionService) CreateSession(ctx context.Context, eventID string, moduleOrder []string) (*models.EventSession, error) {
	if len(moduleOrder) == 0 {
		return nil, errors.New("moduleOrder cannot be empty")
	}

	session := &models.EventSession{
		ID:             utils.NewID(),
		EventID:        eventID,
		Status:         models.EventSessionStatusWaiting,
		ModuleOrder:    moduleOrder,
		ActiveModuleID: moduleOrder[0],
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := s.SessionRepo.Create(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *EventSessionService) GetSession(ctx context.Context, sessionID string) (*models.EventSession, error) {
	return s.SessionRepo.GetByID(ctx, sessionID)
}

func (s *EventSessionService) UpdateSessionStatus(ctx context.Context, sessionID string, status models.EventSessionStatus) error {
	session, err := s.SessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	now := time.Now()
	if status == models.EventSessionStatusLive && session.StartedAt == nil {
		session.StartedAt = &now
	}
	if status == models.EventSessionStatusCompleted {
		session.EndedAt = &now
	}

	return s.SessionRepo.Update(ctx, session)
}

func (s *EventSessionService) AdvanceSessionModule(ctx context.Context, sessionID string, activeModuleID string) error {
	session, err := s.SessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ActiveModuleID = activeModuleID
	session.UpdatedAt = time.Now()

	return s.SessionRepo.Update(ctx, session)
}
