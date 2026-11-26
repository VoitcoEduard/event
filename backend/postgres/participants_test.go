package postgres

import (
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestAssignBalancedTeamsRespectsExistingSizes(t *testing.T) {
	teams := []models.Team{
		{ID: "team-a"},
		{ID: "team-b"},
		{ID: "team-c"},
	}

	counts := map[string]int{
		"team-a": 3,
		"team-b": 1,
		"team-c": 0,
	}

	people := []models.Participant{
		{ID: "p1"},
		{ID: "p2"},
		{ID: "p3"},
		{ID: "p4"},
		{ID: "p5"},
	}

	assignBalancedTeams(teams, counts, people)

	want := []string{"team-c", "team-b", "team-c", "team-b", "team-c"}
	for i, w := range want {
		if people[i].TeamID != w {
			t.Fatalf("person %d: got team %q, want %q", i, people[i].TeamID, w)
		}
	}
}

func TestAssignBalancedTeamsIncrementalLatecomers(t *testing.T) {
	teams := []models.Team{
		{ID: "team-1"},
		{ID: "team-2"},
		{ID: "team-3"},
	}
	counts := map[string]int{}

	for i := 0; i < 6; i++ {
		batch := []models.Participant{{ID: "latecomer"}}
		assignBalancedTeams(teams, counts, batch)
		counts[batch[0].TeamID]++
	}

	for _, team := range teams {
		if counts[team.ID] != 2 {
			t.Fatalf("expected balanced size 2 for %s after 6 incremental arrivals, got %d (all counts: %v)", team.ID, counts[team.ID], counts)
		}
	}
}
