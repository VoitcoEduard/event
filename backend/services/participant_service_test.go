package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
)

// =============================================================================
// Test data
// =============================================================================

var testCtx = context.Background()

// =============================================================================
// Mock repository for testing
// =============================================================================

// MockParticipantRepository is a test double for ParticipantRepository
type MockParticipantRepository struct {
	CreateFunc          func(ctx context.Context, participant *models.Participant) error
	GetByEventIDFunc    func(ctx context.Context, eventID string) ([]models.Participant, error)
	CreatedParticipants []*models.Participant
	GetByEventIDResults map[string][]models.Participant
}

func NewMockRepo() *MockParticipantRepository {
	return &MockParticipantRepository{
		CreatedParticipants: make([]*models.Participant, 0),
		GetByEventIDResults: make(map[string][]models.Participant),
	}
}

func (m *MockParticipantRepository) Create(ctx context.Context, participant *models.Participant) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, participant)
	}
	if participant == nil {
		return errors.New("participant is nil")
	}
	m.CreatedParticipants = append(m.CreatedParticipants, participant)
	return nil
}

func (m *MockParticipantRepository) GetByEventID(ctx context.Context, eventID string) ([]models.Participant, error) {
	if m.GetByEventIDFunc != nil {
		return m.GetByEventIDFunc(ctx, eventID)
	}
	// If explicit results are set, use those (for pre-populated test data)
	if results, ok := m.GetByEventIDResults[eventID]; ok {
		return results, nil
	}
	// Otherwise auto-populate from created participants
	var result []models.Participant
	for _, p := range m.CreatedParticipants {
		if p.EventID == eventID {
			result = append(result, *p)
		}
	}
	return result, nil
}

// =============================================================================
// normalizeName tests
// =============================================================================

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase",
			input:    "JohnDoe",
			expected: "johndoe",
		},
		{
			name:     "with spaces",
			input:    "  John Doe  ",
			expected: "john doe",
		},
		{
			name:     "mixed case with spaces",
			input:    "  JANE SMITH  ",
			expected: "jane smith",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: "",
		},
		{
			name:     "unicode characters",
			input:    "  José García  ",
			expected: "josé garcía",
		},
		{
			name:     "already normalized",
			input:    "alice",
			expected: "alice",
		},
		{
			name:     "with tabs and newlines",
			input:    "\tJohn\nDoe\t",
			expected: "john\ndoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// ID generation tests
// =============================================================================

func TestNewID(t *testing.T) {
	t.Run("returns UUID", func(t *testing.T) {
		id := utils.NewID()
		parsed, err := uuid.Parse(id)
		if err != nil || parsed.Version() != 4 {
			t.Errorf("NewID() = %q, want version 4 UUID", id)
		}
	})

	t.Run("generates different IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := utils.NewID()
			if ids[id] {
				t.Fatalf("duplicate ID generated: %s", id)
			}
			ids[id] = true
		}
	})
}

// =============================================================================
// JoinEvent tests
// =============================================================================

func TestJoinEvent(t *testing.T) {
	tests := []struct {
		name        string
		eventID     string
		joinKey     string
		displayName string
		profile     map[string]any
		setupMock   func(*MockParticipantRepository)
		checkResult func(*testing.T, *models.Participant, error)
	}{
		{
			name:        "successful join",
			eventID:     "event-123",
			joinKey:     "key-abc",
			displayName: "Alice Smith",
			profile:     map[string]any{"role": "player"},
			setupMock: func(m *MockParticipantRepository) {
				// No-op - Create succeeds by default
			},
			checkResult: func(t *testing.T, p *models.Participant, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if p == nil {
					t.Fatal("expected participant, got nil")
				}
				if _, parseErr := uuid.Parse(p.ID); parseErr != nil {
					t.Errorf("ID = %q, want UUID", p.ID)
				}
				if p.EventID != "event-123" {
					t.Errorf("EventID = %q, want %q", p.EventID, "event-123")
				}
				if p.JoinKey != "key-abc" {
					t.Errorf("JoinKey = %q, want %q", p.JoinKey, "key-abc")
				}
				if p.DisplayName != "Alice Smith" {
					t.Errorf("DisplayName = %q, want %q", p.DisplayName, "Alice Smith")
				}
				if p.NormalizedName != "alice smith" {
					t.Errorf("NormalizedName = %q, want %q", p.NormalizedName, "alice smith")
				}
				if p.Status != models.ParticipantStatusJoined {
					t.Errorf("Status = %q, want %q", p.Status, models.ParticipantStatusJoined)
				}
				if p.Profile["role"] != "player" {
					t.Errorf("Profile[role] = %v, want %q", p.Profile["role"], "player")
				}
				if p.CreatedAt.IsZero() {
					t.Error("CreatedAt should not be zero")
				}
				if p.UpdatedAt.IsZero() {
					t.Error("UpdatedAt should not be zero")
				}
			},
		},
		{
			name:        "with nil profile",
			eventID:     "event-456",
			joinKey:     "key-def",
			displayName: "Bob",
			profile:     nil,
			setupMock:   func(m *MockParticipantRepository) {},
			checkResult: func(t *testing.T, p *models.Participant, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if p == nil {
					t.Fatal("expected participant, got nil")
				}
				// nil profile is valid - service stores whatever is passed
				if p.Profile != nil {
					t.Errorf("Profile = %v, want nil", p.Profile)
				}
			},
		},
		{
			name:        "repository create error",
			eventID:     "event-789",
			joinKey:     "key-ghi",
			displayName: "Charlie",
			profile:     nil,
			setupMock: func(m *MockParticipantRepository) {
				m.CreateFunc = func(ctx context.Context, participant *models.Participant) error {
					return errors.New("database connection failed")
				}
			},
			checkResult: func(t *testing.T, p *models.Participant, err error) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if err.Error() != "database connection failed" {
					t.Errorf("error = %q, want %q", err.Error(), "database connection failed")
				}
				if p != nil {
					t.Error("expected nil participant on error")
				}
			},
		},
		{
			name:        "name normalization - uppercase with spaces",
			eventID:     "event-normalize",
			joinKey:     "key-norm",
			displayName: "  JANE DOE  ",
			profile:     nil,
			setupMock:   func(m *MockParticipantRepository) {},
			checkResult: func(t *testing.T, p *models.Participant, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if p.NormalizedName != "jane doe" {
					t.Errorf("NormalizedName = %q, want %q", p.NormalizedName, "jane doe")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRepo()
			tt.setupMock(mock)

			svc := &ParticipantService{Repo: mock}

			participant, err := svc.JoinEvent(testCtx, tt.eventID, tt.joinKey, tt.displayName, tt.profile)

			tt.checkResult(t, participant, err)

			// Verify participant was stored in mock
			if err == nil && len(mock.CreatedParticipants) != 1 {
				t.Errorf("Create was called %d times, want 1", len(mock.CreatedParticipants))
			}
		})
	}
}

// =============================================================================
// GetParticipantsByEvent tests
// =============================================================================

func TestGetParticipantsByEvent(t *testing.T) {
	t.Run("returns empty list when no participants", func(t *testing.T) {
		mock := NewMockRepo()
		mock.GetByEventIDResults["event-empty"] = []models.Participant{}

		svc := &ParticipantService{Repo: mock}

		result, err := svc.GetParticipantsByEvent(testCtx, "event-empty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("got %d participants, want 0", len(result))
		}
	})

	t.Run("returns single participant", func(t *testing.T) {
		mock := NewMockRepo()
		mock.GetByEventIDResults["event-single"] = []models.Participant{
			{ID: "123456", EventID: "event-single", DisplayName: "Alice"},
		}

		svc := &ParticipantService{Repo: mock}

		result, err := svc.GetParticipantsByEvent(testCtx, "event-single")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("got %d participants, want 1", len(result))
		}
		if result[0].DisplayName != "Alice" {
			t.Errorf("DisplayName = %q, want %q", result[0].DisplayName, "Alice")
		}
	})

	t.Run("returns multiple participants", func(t *testing.T) {
		mock := NewMockRepo()
		mock.GetByEventIDResults["event-multi"] = []models.Participant{
			{ID: "111111", EventID: "event-multi", DisplayName: "Alice"},
			{ID: "222222", EventID: "event-multi", DisplayName: "Bob"},
			{ID: "333333", EventID: "event-multi", DisplayName: "Charlie"},
		}

		svc := &ParticipantService{Repo: mock}

		result, err := svc.GetParticipantsByEvent(testCtx, "event-multi")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Fatalf("got %d participants, want 3", len(result))
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := NewMockRepo()
		mock.GetByEventIDFunc = func(ctx context.Context, eventID string) ([]models.Participant, error) {
			return nil, errors.New("database timeout")
		}

		svc := &ParticipantService{Repo: mock}

		result, err := svc.GetParticipantsByEvent(testCtx, "event-error")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "database timeout" {
			t.Errorf("error = %q, want %q", err.Error(), "database timeout")
		}
		if result != nil {
			t.Error("expected nil result on error")
		}
	})
}

// =============================================================================
// ParticipantStatus constants tests
// =============================================================================

func TestParticipantStatusConstants(t *testing.T) {
	if models.ParticipantStatusJoined != "joined" {
		t.Errorf("ParticipantStatusJoined = %q, want %q", models.ParticipantStatusJoined, "joined")
	}
	if models.ParticipantStatusActive != "active" {
		t.Errorf("ParticipantStatusActive = %q, want %q", models.ParticipantStatusActive, "active")
	}
	if models.ParticipantStatusLeft != "left" {
		t.Errorf("ParticipantStatusLeft = %q, want %q", models.ParticipantStatusLeft, "left")
	}
	if models.ParticipantStatusDisqualified != "disqualified" {
		t.Errorf("ParticipantStatusDisqualified = %q, want %q", models.ParticipantStatusDisqualified, "disqualified")
	}
}

// =============================================================================
// Integration-style test (requires actual mock behavior verification)
// =============================================================================

func TestJoinEvent_And_GetParticipantsByEvent(t *testing.T) {
	mock := NewMockRepo()
	svc := &ParticipantService{Repo: mock}

	// Join two participants
	p1, err := svc.JoinEvent(testCtx, "event-integrate", "key-1", "Alice", map[string]any{"team": "red"})
	if err != nil {
		t.Fatalf("JoinEvent 1 failed: %v", err)
	}

	p2, err := svc.JoinEvent(testCtx, "event-integrate", "key-2", "Bob", map[string]any{"team": "blue"})
	if err != nil {
		t.Fatalf("JoinEvent 2 failed: %v", err)
	}

	// Verify stored participants
	if len(mock.CreatedParticipants) != 2 {
		t.Errorf("CreatedParticipants has %d items, want 2", len(mock.CreatedParticipants))
	}

	// Get participants and verify
	participants, err := svc.GetParticipantsByEvent(testCtx, "event-integrate")
	if err != nil {
		t.Fatalf("GetParticipantsByEvent failed: %v", err)
	}

	if len(participants) != 2 {
		t.Errorf("got %d participants, want 2", len(participants))
	}

	// Verify IDs are valid and unique
	ids := make(map[string]bool)
	for _, p := range participants {
		if _, err := uuid.Parse(p.ID); err != nil {
			t.Errorf("invalid ID format: %s", p.ID)
		}
		if ids[p.ID] {
			t.Errorf("duplicate ID: %s", p.ID)
		}
		ids[p.ID] = true
	}

	// Verify p1 and p2 are in the list (by ID)
	foundP1, foundP2 := false, false
	for _, p := range participants {
		if p.ID == p1.ID {
			foundP1 = true
			if p.DisplayName != "Alice" {
				t.Errorf("p1.DisplayName = %q, want %q", p.DisplayName, "Alice")
			}
		}
		if p.ID == p2.ID {
			foundP2 = true
			if p.DisplayName != "Bob" {
				t.Errorf("p2.DisplayName = %q, want %q", p.DisplayName, "Bob")
			}
		}
	}
	if !foundP1 {
		t.Error("p1 not found in participants list")
	}
	if !foundP2 {
		t.Error("p2 not found in participants list")
	}
}
