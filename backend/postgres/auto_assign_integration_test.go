//go:build integration

package postgres_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
)

func TestAutoAssignParticipantsBalancesIncrementalLatecomers(t *testing.T) {
	ctx, db := startDB(t)
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	eventID := uuid.NewString()
	now := time.Now().UTC()
	if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: "314159", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	teamIDs := make([]string, 3)
	for i := range teamIDs {
		teamIDs[i] = uuid.NewString()
		if err := db.CreateTeam(ctx, &models.Team{
			ID:        teamIDs[i],
			EventID:   eventID,
			Name:      fmt.Sprintf("Team %d", i+1),
			Color:     "#112233",
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Register and auto-assign 6 latecomers one at a time.
	for i := 0; i < 6; i++ {
		_, err := db.EnsureParticipant(ctx, &models.Participant{
			ID:             fmt.Sprintf("p-%d", i),
			EventID:        eventID,
			DisplayName:    fmt.Sprintf("Player %d", i),
			NormalizedName: fmt.Sprintf("player %d", i),
			Status:         "joined",
			CreatedAt:      now.Add(time.Duration(i) * time.Second),
			UpdatedAt:      now.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatal(err)
		}
		assigned, err := db.AutoAssignParticipants(ctx, eventID)
		if err != nil {
			t.Fatalf("AutoAssignParticipants round %d: %v", i, err)
		}
		if len(assigned) != 1 {
			t.Fatalf("round %d: expected 1 assigned participant, got %d", i, len(assigned))
		}
	}

	all, err := db.Participants(ctx, eventID)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, p := range all {
		counts[p.TeamID]++
	}
	for _, tid := range teamIDs {
		if counts[tid] != 2 {
			t.Fatalf("expected team %s to have 2 members, got %d (counts=%v)", tid, counts[tid], counts)
		}
	}
}
