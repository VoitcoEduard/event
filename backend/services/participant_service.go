package services

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
)

type ParticipantService struct {
	Repo *database.ParticipantRepository
}

func NewParticipantService() *ParticipantService {
	return &ParticipantService{
		Repo: database.NewParticipantRepository(),
	}
}

func generateID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%06d", r.Intn(1000000))
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (s *ParticipantService) JoinEvent(ctx context.Context, eventID, joinKey, displayName string, profile map[string]any) (*models.Participant, error) {
	participant := &models.Participant{
		ID:             generateID(),
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

	return participant, nil
}

func (s *ParticipantService) GetParticipantsByEvent(ctx context.Context, eventID string) ([]models.Participant, error) {
	return s.Repo.GetByEventID(ctx, eventID)
}
