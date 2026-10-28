package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
)

func fixtureTeams() *models.TeamsActivity {
	a := NewTeamsActivity("event", "relay", "Relay", models.TeamsConfig{Method: "choice", Presets: []int{1, 5, 10}})
	a.Teams = []models.ActivityTeam{{ID: "blue", Name: "Blue", Color: "#3B82F6"}, {ID: "red", Name: "Red", Color: "#EF4444"}}
	a.Entrants = []models.TeamsEntrant{{ParticipantID: "anna", Name: "Anna", TeamID: "blue"}, {ParticipantID: "bob", Name: "Bob", TeamID: "blue"}, {ParticipantID: "cara", Name: "Cara", TeamID: "red"}}
	return a
}
func mustTeams(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func TestTeamsCanStartBeforeParticipantsEnter(t *testing.T) {
	a := NewTeamsActivity("event", "relay", "Relay", models.TeamsConfig{Method: "manual", Presets: []int{1, 5, 10}})
	a.Teams = []models.ActivityTeam{{ID: "blue", Name: "Blue", Color: "#3B82F6"}}

	mustTeams(t, TransitionTeams(a, "start"))
	if a.Status != "live" {
		t.Fatalf("activity status = %q, want live", a.Status)
	}

	mustTeams(t, AddTeamsEntrant(a, models.TeamsEntrant{ParticipantID: "late", Name: "Late participant"}))
	mustTeams(t, AssignTeamsEntrant(a, "late", "blue", false))
	if len(a.Entrants) != 1 || a.Entrants[0].TeamID != "blue" {
		t.Fatal("participant could not enter and be assigned after the activity started")
	}
}
func TestTeamsAwardsKeepOriginalRecipients(t *testing.T) {
	a := fixtureTeams()
	mustTeams(t, TransitionTeams(a, "start"))
	award := models.TeamAward{ID: "win", RequestID: "request", TeamID: "blue", Points: 20, Reason: "Relay"}
	mustTeams(t, AwardTeam(a, award))
	mustTeams(t, AwardTeam(a, award))
	if len(a.Awards) != 1 {
		t.Fatal("retry duplicated award")
	}
	award.Points = 21
	if AwardTeam(a, award) == nil {
		t.Fatal("conflicting retry accepted")
	}
	mustTeams(t, AssignTeamsEntrant(a, "anna", "red", false))
	mustTeams(t, AwardTeam(a, models.TeamAward{ID: "later", RequestID: "later", TeamID: "red", Points: 15}))
	teams, people := a.Scores()
	if teams["blue"] != 20 || teams["red"] != 15 || people["anna"] != 35 || people["bob"] != 20 {
		t.Fatalf("wrong totals %v %v", teams, people)
	}
	entries := a.ScoreEntries()
	teamTotal, annaTotal := 0.0, 0.0
	for _, e := range entries {
		if e.TeamID == "blue" {
			teamTotal += e.Points
		}
		if e.ParticipantID == "anna" {
			annaTotal += e.Points
		}
		if e.TeamID != "" && e.ParticipantID != "" {
			t.Fatal("double-count entry")
		}
	}
	if teamTotal != 20 || annaTotal != 35 {
		t.Fatal("event score projection differs")
	}
	mustTeams(t, TransitionTeams(a, "complete"))
	mustTeams(t, TransitionTeams(a, "complete"))
	if UndoTeamAward(a, "win", "undo", "admin", "correction") == nil {
		t.Fatal("corrected completed activity")
	}
	mustTeams(t, TransitionTeams(a, "reopen"))
	mustTeams(t, UndoTeamAward(a, "win", "undo", "admin", "correction"))
	mustTeams(t, UndoTeamAward(a, "win", "undo", "admin", "correction"))
	if UndoTeamAward(a, "win", "other", "admin", "correction2") == nil {
		t.Fatal("double reversal accepted")
	}
	_, people = a.Scores()
	if people["anna"] != 15 || people["bob"] != 0 {
		t.Fatal("undo did not use original recipients")
	}
	if len(a.Awards) != 3 {
		t.Fatal("lifecycle/retry changed history")
	}
}
func TestTeamsRosterRules(t *testing.T) {
	a := fixtureTeams()
	a.Teams[0].Capacity = 2
	if AssignTeamsEntrant(a, "cara", "blue", true) == nil {
		t.Fatal("capacity bypass")
	}
	mustTeams(t, AssignTeamsEntrant(a, "bob", "", true))
	mustTeams(t, AssignTeamsEntrant(a, "cara", "blue", true))
	if AssignTeamsEntrant(a, "other", "blue", true) == nil {
		t.Fatal("nonentrant assigned")
	}
	mustTeams(t, TransitionTeams(a, "start"))
	if AssignTeamsEntrant(a, "anna", "red", true) == nil {
		t.Fatal("self change after start")
	}
	mustTeams(t, AssignTeamsEntrant(a, "anna", "red", false))
	if AwardTeam(a, models.TeamAward{ID: "bad", RequestID: "bad", TeamID: "elsewhere", Points: 1}) == nil {
		t.Fatal("cross-activity team award")
	}
	a.Teams = append(a.Teams, models.ActivityTeam{ID: "empty", Name: "Empty", Color: "#ffffff"})
	if AwardTeam(a, models.TeamAward{ID: "bad", RequestID: "bad", TeamID: "empty", Points: 1}) == nil {
		t.Fatal("empty team award")
	}
	mustTeams(t, AddTeamsEntrant(a, models.TeamsEntrant{ParticipantID: "late", Name: "Late"}))
	mustTeams(t, AssignTeamsEntrant(a, "late", "red", false))
	mustTeams(t, AwardTeam(a, models.TeamAward{ID: "penalty", RequestID: "penalty", TeamID: "red", Points: -5}))
	if DeleteActivityTeam(a, "red") == nil {
		t.Fatal("deleted awarded team")
	}
	_, scores := a.Scores()
	if scores["late"] != -5 {
		t.Fatal("late entrant/negative score")
	}
}
func TestTeamsRandomBalancedAndCapacity(t *testing.T) {
	for _, useSize := range []bool{false, true} {
		a := NewTeamsActivity("e", "m", "", models.TeamsConfig{Method: "random", TeamCount: 3})
		if useSize {
			a.Config.TeamCount = 0
			a.Config.TargetSize = 3
		}
		for i := 0; i < 7; i++ {
			mustTeams(t, AddTeamsEntrant(a, models.TeamsEntrant{ParticipantID: fmt.Sprint(i)}))
		}
		mustTeams(t, DistributeTeams(a, false))
		counts := map[string]int{}
		for _, p := range a.Entrants {
			counts[p.TeamID]++
		}
		if len(counts) != 3 {
			t.Fatalf("expected 3 teams: %v", counts)
		}
		for _, count := range counts {
			if count < 2 || count > 3 {
				t.Fatal("unbalanced")
			}
		}
		first := a.Entrants[0].TeamID
		mustTeams(t, AddTeamsEntrant(a, models.TeamsEntrant{ParticipantID: "late"}))
		mustTeams(t, DistributeTeams(a, false))
		if a.Entrants[0].TeamID != first {
			t.Fatal("moved existing player")
		}
		mustTeams(t, DistributeTeams(a, true))
		a.Status = "live"
		if DistributeTeams(a, true) == nil {
			t.Fatal("reshuffled live activity")
		}
	}
	a := fixtureTeams()
	a.Config.Method = "random"
	a.Config.TeamCount = 2
	a.Config.Capacity = 1
	if DistributeTeams(a, true) == nil {
		t.Fatal("insufficient capacity accepted")
	}
}
func TestTeamsReuseOnlyEntrantsEarnAndTies(t *testing.T) {
	source := fixtureTeams()
	a := NewTeamsActivity("event", "next", "Next", models.TeamsConfig{Method: "choice"})
	a.Teams = append(a.Teams, source.Teams...)
	a.SourceMembers = append(a.SourceMembers, source.Entrants...)
	mustTeams(t, AddTeamsEntrant(a, models.TeamsEntrant{ParticipantID: "anna", Name: "Anna"}))
	if a.Entrants[0].TeamID != "blue" {
		t.Fatal("source assignment not restored")
	}
	mustTeams(t, TransitionTeams(a, "start"))
	mustTeams(t, AwardTeam(a, models.TeamAward{ID: "a", RequestID: "a", TeamID: "blue", Points: 20}))
	_, scores := a.Scores()
	if scores["bob"] != 0 || scores["anna"] != 20 {
		t.Fatal("source nonentrant received points")
	}
	mustTeams(t, AssignTeamsEntrant(a, "anna", "red", false))
	if source.Entrants[0].TeamID != "blue" {
		t.Fatal("modified source")
	}
	a.Awards = nil
	rows := a.Standings()
	if rows[0].Rank != 1 || rows[1].Rank != 1 || rows[0].Name != "Blue" {
		t.Fatal("tie rank/display order")
	}
}

type memoryTeamsStore struct {
	mu        sync.Mutex
	a         *models.TeamsActivity
	conflicts int
}

func cloneActivity(a *models.TeamsActivity) *models.TeamsActivity {
	b, _ := json.Marshal(a)
	var c models.TeamsActivity
	_ = json.Unmarshal(b, &c)
	c.Deleted = a.Deleted
	return &c
}
func (s *memoryTeamsStore) Get(_ context.Context, e, m string) (*models.TeamsActivity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.a == nil || s.a.Deleted || s.a.EventID != e || s.a.ModuleID != m {
		return nil, database.ErrNotFound
	}
	return cloneActivity(s.a), nil
}
func (s *memoryTeamsStore) Create(_ context.Context, a *models.TeamsActivity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.a != nil {
		return database.ErrConflict
	}
	s.a = cloneActivity(a)
	return nil
}
func (s *memoryTeamsStore) CompareAndSwap(_ context.Context, a *models.TeamsActivity, revision int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conflicts > 0 {
		s.conflicts--
		return false, nil
	}
	if s.a.Revision != revision || s.a.Deleted {
		return false, nil
	}
	s.a = cloneActivity(a)
	return true, nil
}
func TestTeamsConcurrentCapacityAndAwards(t *testing.T) {
	a := fixtureTeams()
	a.Entrants[0].TeamID = ""
	a.Entrants[1].TeamID = ""
	a.Teams[0].Capacity = 1
	store := &memoryTeamsStore{a: a, conflicts: 2}
	service := TeamsActivityService{store}
	var wg sync.WaitGroup
	success := make(chan bool, 2)
	for _, id := range []string{"anna", "bob"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error { return AssignTeamsEntrant(a, id, "blue", true) })
			success <- err == nil
		}(id)
	}
	wg.Wait()
	close(success)
	n := 0
	for ok := range success {
		if ok {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("last slot accepted %d people", n)
	}
	_, err := service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error { return TransitionTeams(a, "start") })
	mustTeams(t, err)
	errors := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error {
				return AwardTeam(a, models.TeamAward{ID: fmt.Sprint(i), RequestID: fmt.Sprint(i), TeamID: "blue", Points: 1})
			})
			errors <- err
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		mustTeams(t, err)
	}
	state, err := store.Get(context.Background(), "event", "relay")
	mustTeams(t, err)
	totals, _ := state.Scores()
	if totals["blue"] != 12 {
		t.Fatal("lost concurrent awards")
	}
	if GuardTeamsModuleDeletion(context.Background(), store, "event", "relay") == nil {
		t.Fatal("deleted awarded module")
	}
}
func TestTeamsDeleteAndFirstAwardSerialize(t *testing.T) {
	store := &memoryTeamsStore{a: fixtureTeams()}
	store.a.Status = "live"
	service := TeamsActivityService{store}
	var wg sync.WaitGroup
	var deleteErr, awardErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		deleteErr = GuardTeamsModuleDeletion(context.Background(), store, "event", "relay")
	}()
	go func() {
		defer wg.Done()
		_, awardErr = service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error {
			return AwardTeam(a, models.TeamAward{ID: "first", RequestID: "first", TeamID: "blue", Points: 20})
		})
	}()
	wg.Wait()
	if deleteErr == nil && awardErr == nil {
		t.Fatal("deletion and first award both succeeded")
	}
}
func TestWholeTeamsPoints(t *testing.T) {
	for _, v := range []float64{0, 1.5, 1000001} {
		if _, err := WholeTeamsPoints(&v); err == nil {
			t.Fatal("invalid points accepted")
		}
	}
}

func TestTeamsAwardRevalidatesAfterRosterChange(t *testing.T) {
	store := &memoryTeamsStore{a: fixtureTeams()}
	store.a.Status = "live"
	service := TeamsActivityService{store}
	read := make(chan struct{})
	changed := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		first := true
		_, err := service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error {
			if first {
				first = false
				close(read)
				<-changed
			}
			return AwardTeam(a, models.TeamAward{ID: "win", RequestID: "win", TeamID: "blue", Points: 20})
		})
		done <- err
	}()
	<-read
	_, err := service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error { return AssignTeamsEntrant(a, "anna", "red", false) })
	mustTeams(t, err)
	close(changed)
	mustTeams(t, <-done)
	current, err := store.Get(context.Background(), "event", "relay")
	mustTeams(t, err)
	_, people := current.Scores()
	if people["anna"] != 0 || people["bob"] != 20 {
		t.Fatal("stale recipient list committed")
	}
}
func TestTeamsAwardCannotCommitAfterCompletion(t *testing.T) {
	store := &memoryTeamsStore{a: fixtureTeams()}
	store.a.Status = "live"
	service := TeamsActivityService{store}
	read := make(chan struct{})
	completed := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		first := true
		_, err := service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error {
			if first {
				first = false
				close(read)
				<-completed
			}
			return AwardTeam(a, models.TeamAward{ID: "win", RequestID: "win", TeamID: "blue", Points: 20})
		})
		done <- err
	}()
	<-read
	_, err := service.Mutate(context.Background(), "event", "relay", func(a *models.TeamsActivity) error { return TransitionTeams(a, "complete") })
	mustTeams(t, err)
	close(completed)
	if <-done == nil {
		t.Fatal("award committed after completion")
	}
	current, err := store.Get(context.Background(), "event", "relay")
	mustTeams(t, err)
	if len(current.Awards) != 0 || current.Status != "completed" {
		t.Fatal("completion lost")
	}
}
func TestTeamsTargetSizeAddsTeamsWithoutMovingAssignedPeople(t *testing.T) {
	a := NewTeamsActivity("e", "m", "", models.TeamsConfig{Method: "random", TargetSize: 2})
	for i := 0; i < 4; i++ {
		mustTeams(t, AddTeamsEntrant(a, models.TeamsEntrant{ParticipantID: fmt.Sprint(i)}))
	}
	mustTeams(t, DistributeTeams(a, false))
	first := a.Entrants[0].TeamID
	mustTeams(t, AddTeamsEntrant(a, models.TeamsEntrant{ParticipantID: "late"}))
	mustTeams(t, DistributeTeams(a, false))
	if len(a.Teams) != 3 || a.Entrants[0].TeamID != first {
		t.Fatal("target size growth moved an existing member or ignored new team count")
	}
}
