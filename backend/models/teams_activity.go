package models

import (
	"fmt"
	"sort"
	"time"
)

type TeamsConfig struct {
	SourceModuleID string `json:"sourceModuleId"`
	Method         string `json:"method"`
	TeamCount      int    `json:"teamCount"`
	TargetSize     int    `json:"targetSize"`
	Capacity       int    `json:"capacity"`
	Presets        []int  `json:"presets"`
}

type ActivityTeam struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Capacity int    `json:"capacity"`
}

type TeamsEntrant struct {
	ParticipantID string `json:"participantId"`
	Name          string `json:"name"`
	TeamID        string `json:"teamId"`
}

type TeamAward struct {
	ID         string    `json:"id"`
	RequestID  string    `json:"requestId"`
	TeamID     string    `json:"teamId"`
	Points     int       `json:"points"`
	Reason     string    `json:"reason"`
	Recipients []string  `json:"recipients"`
	ActorID    string    `json:"actorId"`
	CreatedAt  time.Time `json:"createdAt"`
	ReversesID string    `json:"reversesId,omitempty"`
}

// Roster and award history share one atomic persistence boundary. SourceMembers
// are invitations from a reused roster, not recipients until they enter.
type TeamsActivity struct {
	ID            string         `json:"id"`
	EventID       string         `json:"eventId"`
	ModuleID      string         `json:"moduleId"`
	Name          string         `json:"name"`
	Status        string         `json:"status"`
	Revision      int64          `json:"revision"`
	Config        TeamsConfig    `json:"config"`
	Teams         []ActivityTeam `json:"teams"`
	Entrants      []TeamsEntrant `json:"entrants"`
	SourceMembers []TeamsEntrant `json:"sourceMembers"`
	Awards        []TeamAward    `json:"awards"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	FinishedAt    *time.Time     `json:"finishedAt,omitempty"`
	Deleted       bool           `json:"-"`
}

func (a *TeamsActivity) Scores() (map[string]int, map[string]int) {
	teams, people := map[string]int{}, map[string]int{}
	for _, award := range a.Awards {
		teams[award.TeamID] += award.Points
		for _, id := range award.Recipients {
			people[id] += award.Points
		}
	}
	return teams, people
}

// Entries are derived, never dual-written to event_scores. Team entries contain
// only TeamID; recipient entries contain only ParticipantID to avoid double count.
func (a *TeamsActivity) ScoreEntries() []ScoreEntry {
	entries := []ScoreEntry{}
	for _, award := range a.Awards {
		base := ScoreEntry{ID: fmt.Sprintf("teams:%s:%s", a.ID, award.ID), EventID: a.EventID, ModuleID: a.ModuleID, Category: ScoreCategoryBase, Points: float64(award.Points), Reason: award.Reason, CreatedAt: award.CreatedAt}
		if award.Points < 0 {
			base.Category = ScoreCategoryPenalty
		}
		if award.ReversesID != "" {
			base.Category = ScoreCategoryAdjustment
		}
		team := base
		team.TeamID = award.TeamID
		entries = append(entries, team)
		for _, id := range award.Recipients {
			person := base
			person.ID += ":" + id
			person.ParticipantID = id
			entries = append(entries, person)
		}
	}
	return entries
}

type TeamsStanding struct {
	ActivityTeam
	Score   int            `json:"score"`
	Rank    int            `json:"rank"`
	Members []TeamsEntrant `json:"members"`
}

func (a *TeamsActivity) Standings() []TeamsStanding {
	totals, _ := a.Scores()
	rows := []TeamsStanding{}
	for _, team := range a.Teams {
		row := TeamsStanding{ActivityTeam: team, Score: totals[team.ID], Members: []TeamsEntrant{}}
		for _, member := range a.Entrants {
			if member.TeamID == team.ID {
				row.Members = append(row.Members, member)
			}
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Score == rows[j].Score {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Score > rows[j].Score
	})
	for i := range rows {
		rows[i].Rank = i + 1
		if i > 0 && rows[i].Score == rows[i-1].Score {
			rows[i].Rank = rows[i-1].Rank
		}
	}
	return rows
}
