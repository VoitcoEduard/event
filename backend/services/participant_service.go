package services

import (
	"context"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/utils"
)

// ParticipantStore is the persistence boundary used by ParticipantService.
type ParticipantStore interface {
	Create(context.Context, *models.Participant) error
	GetByEventID(context.Context, string) ([]models.Participant, error)
}

type ParticipantService struct {
	Repo ParticipantStore
}

func NewParticipantService(db *postgres.Store) *ParticipantService {
	return &ParticipantService{
		Repo: database.NewParticipantRepository(db),
	}
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (s *ParticipantService) JoinEvent(ctx context.Context, eventID, joinKey, displayName string, profile map[string]any) (*models.Participant, error) {
	return s.JoinEventWithID(ctx, eventID, "", joinKey, displayName, profile)
}

func (s *ParticipantService) JoinEventWithID(ctx context.Context, eventID, userID, joinKey, displayName string, profile map[string]any) (*models.Participant, error) {
	participant := &models.Participant{
		ID:             userID,
		EventID:        eventID,
		JoinKey:        joinKey,
		DisplayName:    displayName,
		NormalizedName: normalizeName(displayName),
		Profile:        profile,
		Status:         models.ParticipantStatusJoined,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := s.Repo.Create(ctx, participant)
	if err != nil {
		return nil, err
	}

	if participant.ID == "" {
		participant.ID = utils.NewID()
	}
	return participant, nil
}

func (s *ParticipantService) GetParticipantsByEvent(ctx context.Context, eventID string) ([]models.Participant, error) {
	return s.Repo.GetByEventID(ctx, eventID)
}
