package models

import (
	"fmt"
	"sort"
	"time"
)

type TeamsConfig struct {
	SourceModuleID string `json:"sourceModuleId" bson:"sourceModuleId"`
	Method         string `json:"method" bson:"method"`
	TeamCount      int    `json:"teamCount" bson:"teamCount"`
	TargetSize     int    `json:"targetSize" bson:"targetSize"`
	Capacity       int    `json:"capacity" bson:"capacity"`
	Presets        []int  `json:"presets" bson:"presets"`
}

type ActivityTeam struct {
	ID       string `json:"id" bson:"id"`
	Name     string `json:"name" bson:"name"`
	Color    string `json:"color" bson:"color"`
	Capacity int    `json:"capacity" bson:"capacity"`
}

type TeamsEntrant struct {
	ParticipantID string `json:"participantId" bson:"participantId"`
	Name          string `json:"name" bson:"name"`
	TeamID        string `json:"teamId" bson:"teamId"`
}

type TeamAward struct {
	ID         string    `json:"id" bson:"id"`
	RequestID  string    `json:"requestId" bson:"requestId"`
	TeamID     string    `json:"teamId" bson:"teamId"`
	Points     int       `json:"points" bson:"points"`
	Reason     string    `json:"reason" bson:"reason"`
	Recipients []string  `json:"recipients" bson:"recipients"`
	ActorID    string    `json:"actorId" bson:"actorId"`
	CreatedAt  time.Time `json:"createdAt" bson:"createdAt"`
	ReversesID string    `json:"reversesId,omitempty" bson:"reversesId,omitempty"`
}

// Roster and award history share one atomic persistence boundary. SourceMembers
// are invitations from a reused roster, not recipients until they enter.
type TeamsActivity struct {
	ID            string         `json:"id" bson:"_id"`
	EventID       string         `json:"eventId" bson:"eventId"`
	ModuleID      string         `json:"moduleId" bson:"moduleId"`
	Name          string         `json:"name" bson:"name"`
	Status        string         `json:"status" bson:"status"`
	Revision      int64          `json:"revision" bson:"revision"`
	Config        TeamsConfig    `json:"config" bson:"config"`
	Teams         []ActivityTeam `json:"teams" bson:"teams"`
	Entrants      []TeamsEntrant `json:"entrants" bson:"entrants"`
	SourceMembers []TeamsEntrant `json:"sourceMembers" bson:"sourceMembers"`
	Awards        []TeamAward    `json:"awards" bson:"awards"`
	CreatedAt     time.Time      `json:"createdAt" bson:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt" bson:"updatedAt"`
	FinishedAt    *time.Time     `json:"finishedAt,omitempty" bson:"finishedAt,omitempty"`
	Deleted       bool           `json:"-" bson:"deleted,omitempty"`
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
