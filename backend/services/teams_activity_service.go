package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
)

var ErrTeamsConflict = errors.New("activity changed; refresh and try again")

type TeamsError struct{ Message string }

func (e *TeamsError) Error() string   { return e.Message }
func teamsError(message string) error { return &TeamsError{message} }

type TeamsActivityStore interface {
	Get(context.Context, string, string) (*models.TeamsActivity, error)
	Create(context.Context, *models.TeamsActivity) error
	CompareAndSwap(context.Context, *models.TeamsActivity, int64) (bool, error)
}
type TeamsActivityService struct{ Store TeamsActivityStore }

// Every operation rereads and revalidates against the roster which it commits.
// Retrying never uses an old recipient snapshot or stale capacity check.
func (s *TeamsActivityService) Mutate(ctx context.Context, eventID, moduleID string, fn func(*models.TeamsActivity) error) (*models.TeamsActivity, error) {
	for attempt := 0; attempt < 24; attempt++ {
		a, err := s.Store.Get(ctx, eventID, moduleID)
		if err != nil {
			return nil, err
		}
		revision := a.Revision
		if err = fn(a); err != nil {
			return nil, err
		}
		a.Revision++
		a.UpdatedAt = time.Now()
		ok, err := s.Store.CompareAndSwap(ctx, a, revision)
		if err != nil {
			return nil, err
		}
		if ok {
			return a, nil
		}
	}
	return nil, ErrTeamsConflict
}

func ValidateTeamsConfig(c *models.TeamsConfig) error {
	if c.Method == "" {
		c.Method = "random"
	}
	if c.Method != "random" && c.Method != "choice" && c.Method != "manual" {
		return teamsError("Choose random, choice or manual formation")
	}
	if c.TeamCount < 0 || c.TargetSize < 0 || c.Capacity < 0 || c.TeamCount > 200 {
		return teamsError("Invalid team count, size or capacity")
	}
	if c.TeamCount > 0 && c.TargetSize > 0 {
		return teamsError("Choose either a team count or a target size")
	}
	if len(c.Presets) == 0 {
		c.Presets = []int{1, 5, 10}
	}
	if len(c.Presets) > 12 {
		return teamsError("Use at most 12 award presets")
	}
	for _, p := range c.Presets {
		if p == 0 || p > 1000000 || p < -1000000 {
			return teamsError("Award presets must be nonzero whole numbers up to 1000000")
		}
	}
	return nil
}
func NewTeamsActivity(eventID, moduleID, name string, c models.TeamsConfig) *models.TeamsActivity {
	return &models.TeamsActivity{ID: utils.NewID(), EventID: eventID, ModuleID: moduleID, Name: name, Status: "setup", Revision: 1, Config: c, Teams: []models.ActivityTeam{}, Entrants: []models.TeamsEntrant{}, SourceMembers: []models.TeamsEntrant{}, Awards: []models.TeamAward{}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}
func TeamIndex(a *models.TeamsActivity, id string) int {
	return slices.IndexFunc(a.Teams, func(t models.ActivityTeam) bool { return t.ID == id })
}
func entrantIndex(a *models.TeamsActivity, id string) int {
	return slices.IndexFunc(a.Entrants, func(p models.TeamsEntrant) bool { return p.ParticipantID == id })
}
func ensureEditable(a *models.TeamsActivity) error {
	if a.Status == "completed" {
		return teamsError("Reopen the activity before making changes")
	}
	return nil
}
func AddTeamsEntrant(a *models.TeamsActivity, p models.TeamsEntrant) error {
	if i := entrantIndex(a, p.ParticipantID); i >= 0 {
		a.Entrants[i].Name = p.Name
		return nil
	}
	if err := ensureEditable(a); err != nil {
		return err
	}
	p.TeamID = ""
	for _, source := range a.SourceMembers {
		if source.ParticipantID == p.ParticipantID && TeamIndex(a, source.TeamID) >= 0 {
			p.TeamID = source.TeamID
			break
		}
	}
	if p.TeamID != "" && !teamHasRoom(a, p.TeamID, "") {
		p.TeamID = ""
	}
	a.Entrants = append(a.Entrants, p)
	return nil
}
func teamHasRoom(a *models.TeamsActivity, teamID, except string) bool {
	i := TeamIndex(a, teamID)
	if i < 0 {
		return false
	}
	capacity := a.Teams[i].Capacity
	if capacity == 0 {
		capacity = a.Config.Capacity
	}
	if capacity == 0 {
		return true
	}
	count := 0
	for _, p := range a.Entrants {
		if p.TeamID == teamID && p.ParticipantID != except {
			count++
		}
	}
	return count < capacity
}
func AssignTeamsEntrant(a *models.TeamsActivity, participantID, teamID string, self bool) error {
	if err := ensureEditable(a); err != nil {
		return err
	}
	if self && (a.Status != "setup" || a.Config.Method != "choice") {
		return teamsError("Team selection is controlled by the admin")
	}
	i := entrantIndex(a, participantID)
	if i < 0 {
		return teamsError("Enter the activity first")
	}
	if teamID != "" {
		if TeamIndex(a, teamID) < 0 {
			return teamsError("Team not found in this activity")
		}
		if !teamHasRoom(a, teamID, participantID) {
			return teamsError("This team is full")
		}
	}
	a.Entrants[i].TeamID = teamID
	return nil
}

var teamColors = []string{"#3B82F6", "#EF4444", "#22C55E", "#F59E0B", "#8B5CF6", "#EC4899", "#14B8A6", "#F97316"}

func SaveActivityTeam(a *models.TeamsActivity, t models.ActivityTeam) error {
	if err := ensureEditable(a); err != nil {
		return err
	}
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" || len(t.Name) > 120 || t.Capacity < 0 {
		return teamsError("Provide a team name and a nonnegative capacity")
	}
	if len(t.Color) != 7 || t.Color[0] != '#' {
		return teamsError("Use a hex team color")
	}
	for _, char := range t.Color[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
			return teamsError("Use a hex team color")
		}
	}
	i := TeamIndex(a, t.ID)
	capacity := t.Capacity
	if capacity == 0 {
		capacity = a.Config.Capacity
	}
	count := 0
	for _, p := range a.Entrants {
		if p.TeamID == t.ID {
			count++
		}
	}
	if capacity > 0 && count > capacity {
		return teamsError("Capacity is smaller than the current roster")
	}
	if i >= 0 {
		a.Teams[i] = t
	} else {
		a.Teams = append(a.Teams, t)
	}
	return nil
}
func DeleteActivityTeam(a *models.TeamsActivity, id string) error {
	if err := ensureEditable(a); err != nil {
		return err
	}
	i := TeamIndex(a, id)
	if i < 0 {
		return teamsError("Team not found")
	}
	for _, award := range a.Awards {
		if award.TeamID == id {
			return teamsError("A team with award history cannot be deleted")
		}
	}
	a.Teams = append(a.Teams[:i], a.Teams[i+1:]...)
	for i := range a.Entrants {
		if a.Entrants[i].TeamID == id {
			a.Entrants[i].TeamID = ""
		}
	}
	return nil
}
func DistributeTeams(a *models.TeamsActivity, reshuffle bool) error {
	if a.Status != "setup" {
		return teamsError("Random distribution is available during setup")
	}
	if a.Config.Method != "random" {
		return teamsError("Choose random formation first")
	}
	if a.Config.TeamCount == 0 && a.Config.TargetSize == 0 {
		return teamsError("Choose a team count or target team size first")
	}
	if len(a.Entrants) == 0 {
		return teamsError("No participants have entered")
	}
	count := a.Config.TeamCount
	if count == 0 {
		count = (len(a.Entrants) + a.Config.TargetSize - 1) / a.Config.TargetSize
	}
	if count > 200 {
		return teamsError("Use at most 200 teams")
	}
	if len(a.Teams) > count {
		return teamsError("Delete extra teams or increase the requested team count before distributing")
	}
	for i := len(a.Teams); i < count; i++ {
		a.Teams = append(a.Teams, models.ActivityTeam{ID: utils.NewID(), Name: fmt.Sprintf("Team %d", i+1), Color: teamColors[i%len(teamColors)], Capacity: a.Config.Capacity})
	}
	counts := make(map[string]int)
	pending := []int{}
	for i, p := range a.Entrants {
		if reshuffle || p.TeamID == "" {
			pending = append(pending, i)
		} else {
			counts[p.TeamID]++
		}
	}
	rand.Shuffle(len(pending), func(i, j int) { pending[i], pending[j] = pending[j], pending[i] })
	for _, i := range pending {
		best := -1
		for j, t := range a.Teams {
			cap := t.Capacity
			if cap == 0 {
				cap = a.Config.Capacity
			}
			if cap > 0 && counts[t.ID] >= cap {
				continue
			}
			if best < 0 || counts[t.ID] < counts[a.Teams[best].ID] {
				best = j
			}
		}
		if best < 0 {
			return teamsError("Team capacity is insufficient; increase capacity or add teams")
		}
		id := a.Teams[best].ID
		a.Entrants[i].TeamID = id
		counts[id]++
	}
	return nil
}
func TransitionTeams(a *models.TeamsActivity, action string) error {
	switch action {
	case "start":
		if a.Status == "live" {
			return nil
		}
		if a.Status != "setup" {
			return teamsError("Only setup activities can start")
		}
		if len(a.Teams) == 0 {
			return teamsError("Create teams before starting")
		}
		a.Status = "live"
	case "complete":
		if a.Status == "completed" {
			return nil
		}
		if a.Status != "live" {
			return teamsError("Start the activity before completing")
		}
		now := time.Now()
		a.Status = "completed"
		a.FinishedAt = &now
	case "reopen":
		if a.Status == "live" {
			return nil
		}
		if a.Status != "completed" {
			return teamsError("Only completed activities can reopen")
		}
		a.Status = "live"
		a.FinishedAt = nil
	default:
		return teamsError("Unknown lifecycle action")
	}
	return nil
}
func AwardTeam(a *models.TeamsActivity, award models.TeamAward) error {
	if award.RequestID == "" || len(award.RequestID) > 120 {
		return teamsError("A request ID is required")
	}
	for _, old := range a.Awards {
		if old.RequestID == award.RequestID {
			if old.TeamID != award.TeamID || old.Points != award.Points || old.Reason != award.Reason || old.ReversesID != award.ReversesID {
				return teamsError("Request ID was already used for a different award")
			}
			return nil
		}
	}
	if a.Status != "live" {
		return teamsError("Awards require a live activity")
	}
	if TeamIndex(a, award.TeamID) < 0 {
		return teamsError("Team not found in this activity")
	}
	if award.Points == 0 || award.Points > 1000000 || award.Points < -1000000 {
		return teamsError("Use nonzero whole points up to 1000000")
	}
	if len(award.Reason) > 500 {
		return teamsError("Reason is too long")
	}
	award.Recipients = []string{}
	for _, p := range a.Entrants {
		if p.TeamID == award.TeamID {
			award.Recipients = append(award.Recipients, p.ParticipantID)
		}
	}
	if len(award.Recipients) == 0 {
		return teamsError("Empty teams cannot receive awards")
	}
	award.CreatedAt = time.Now()
	a.Awards = append(a.Awards, award)
	return nil
}
func UndoTeamAward(a *models.TeamsActivity, id, requestID, actorID, newID string) error {
	if requestID == "" || len(requestID) > 120 {
		return teamsError("A request ID is required")
	}
	for _, old := range a.Awards {
		if old.RequestID == requestID {
			if old.ReversesID == id {
				return nil
			}
			return teamsError("Request ID was already used for a different award")
		}
	}
	if a.Status != "live" {
		return teamsError("Reopen the activity to correct awards")
	}
	for _, old := range a.Awards {
		if old.ReversesID == id {
			return teamsError("This award was already undone")
		}
	}
	for _, old := range a.Awards {
		if old.ID == id {
			if old.ReversesID != "" {
				return teamsError("Corrections cannot be undone")
			}
			correction := models.TeamAward{ID: newID, RequestID: requestID, TeamID: old.TeamID, Points: -old.Points, Reason: "Undo: " + old.Reason, Recipients: slices.Clone(old.Recipients), ActorID: actorID, CreatedAt: time.Now(), ReversesID: id}
			a.Awards = append(a.Awards, correction)
			return nil
		}
	}
	return teamsError("Award not found")
}
func WholeTeamsPoints(points *float64) (int, error) {
	if points == nil || math.IsNaN(*points) || math.IsInf(*points, 0) || math.Trunc(*points) != *points || *points > 1000000 || *points < -1000000 || *points == 0 {
		return 0, teamsError("Use nonzero whole points up to 1000000")
	}
	return int(*points), nil
}

// A deletion tombstone uses the same CAS boundary as awards, so a first award
// and deletion cannot both succeed. Uninitialized modules also get a tombstone.
func GuardTeamsModuleDeletion(ctx context.Context, store TeamsActivityStore, eventID, moduleID string) error {
	for i := 0; i < 24; i++ {
		a, err := store.Get(ctx, eventID, moduleID)
		if errors.Is(err, database.ErrNotFound) {
			a = NewTeamsActivity(eventID, moduleID, "", models.TeamsConfig{})
			a.Deleted = true
			err = store.Create(ctx, a)
			if errors.Is(err, database.ErrConflict) {
				continue
			}
			return err
		}
		if err != nil {
			return err
		}
		if len(a.Awards) > 0 {
			return teamsError("This Teams activity contains awards; deactivate it instead of deleting it")
		}
		revision := a.Revision
		a.Revision++
		a.Deleted = true
		ok, err := store.CompareAndSwap(ctx, a, revision)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return ErrTeamsConflict
}

func TeamsActivityResult(a *models.TeamsActivity) map[string]any {
	_, scores := a.Scores()
	teams := []map[string]any{}
	for _, row := range a.Standings() {
		members := []map[string]any{}
		for _, p := range row.Members {
			members = append(members, map[string]any{"id": p.ParticipantID, "nickname": p.Name, "points": scores[p.ParticipantID]})
		}
		teams = append(teams, map[string]any{"id": row.ID, "name": row.Name, "color": row.Color, "score": row.Score, "rank": row.Rank, "members": members})
	}
	people := []map[string]any{}
	for _, p := range a.Entrants {
		people = append(people, map[string]any{"id": p.ParticipantID, "nickname": p.Name, "teamId": p.TeamID, "points": scores[p.ParticipantID]})
	}
	return map[string]any{"eventId": a.EventID, "moduleId": a.ModuleID, "moduleName": a.Name, "status": a.Status, "finishedAt": a.FinishedAt, "teams": teams, "participants": people, "awards": a.Awards}
}
