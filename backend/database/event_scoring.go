package database

import (
	"context"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

type EventScores struct {
	Games        []models.QuizResult     `json:"games"`
	Entries      []models.ScoreEntry     `json:"entries"`
	Teams        []EventScoreTeam        `json:"teams"`
	Participants []EventScoreParticipant `json:"participants"`
}

type EventScoreTeam struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ModuleID string `json:"moduleId"`
}
type EventScoreParticipant struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

func LoadEventScores(ctx context.Context, db *postgres.Store, eventID string) (*EventScores, error) {
	event, err := db.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}
	modules := map[string]bool{}
	for _, m := range event.Modules {
		modules[m.ID] = true
	}
	games, err := db.LatestQuizResults(ctx, eventID)
	if err != nil {
		return nil, err
	}
	entries, err := db.ScoreEntries(ctx, eventID)
	if err != nil {
		return nil, err
	}
	activities, err := db.TeamsActivities(ctx, eventID)
	if err != nil {
		return nil, err
	}
	out := &EventScores{Games: games, Entries: []models.ScoreEntry{}, Teams: []EventScoreTeam{}, Participants: []EventScoreParticipant{}}
	for _, entry := range entries {
		if modules[entry.ModuleID] {
			out.Entries = append(out.Entries, entry)
		}
	}
	for _, a := range activities {
		if !modules[a.ModuleID] {
			continue
		}
		out.Entries = append(out.Entries, a.ScoreEntries()...)
		for _, team := range a.Teams {
			out.Teams = append(out.Teams, EventScoreTeam{team.ID, team.Name, a.ModuleID})
		}
		for _, person := range a.Entrants {
			out.Participants = append(out.Participants, EventScoreParticipant{person.ParticipantID, person.Name})
		}
	}
	return out, nil
}

func (s *EventScores) PlayerTotal(playerID string) float64 {
	total := 0.0
	for _, game := range s.Games {
		for _, player := range game.Players {
			if player.ID == playerID {
				total += float64(player.Score)
			}
		}
	}
	for _, entry := range s.Entries {
		if entry.ParticipantID == playerID {
			total += entry.Points
		}
	}
	return total
}
