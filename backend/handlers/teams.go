package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
)

// Team repositories
var (
	teamRepo        *database.TeamRepository
	participantRepo *database.ParticipantRepository
	scoreEntryRepo  *database.ScoreEntryRepository
)

func getTeamRepo() *database.TeamRepository {
	if teamRepo == nil {
		teamRepo = database.NewTeamRepository()
	}
	return teamRepo
}

func getParticipantRepo() *database.ParticipantRepository {
	if participantRepo == nil {
		participantRepo = database.NewParticipantRepository()
	}
	return participantRepo
}

func getScoreEntryRepo() *database.ScoreEntryRepository {
	if scoreEntryRepo == nil {
		scoreEntryRepo = database.NewScoreEntryRepository()
	}
	return scoreEntryRepo
}

// TeamWithScore represents a team with its total score
type TeamWithScore struct {
	Team        models.Team `json:"team"`
	TotalScore  float64     `json:"totalScore"`
	MemberCount int         `json:"memberCount"`
}

// GetTeams handles GET /events/{id}/teams
func GetTeams(w http.ResponseWriter, r *http.Request) {
	eventID := extractEventID(r.URL.Path)
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	teams, err := getTeamRepo().GetByEventSessionID(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get scores for each team
	teamsWithScores := make([]TeamWithScore, len(teams))
	for i, team := range teams {
		score, _ := getScoreEntryRepo().GetTeamTotalScore(r.Context(), team.ID)
		teamsWithScores[i] = TeamWithScore{
			Team:        team,
			TotalScore:  score,
			MemberCount: len(team.MemberParticipantIDs),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teamsWithScores)
}

// SplitTeams handles POST /events/{id}/teams/split
func SplitTeams(w http.ResponseWriter, r *http.Request) {
	eventID := extractEventID(r.URL.Path)
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		SplitMethod string   `json:"splitMethod"`
		TeamNames   []string `json:"teamNames"`
		TeamColors  []string `json:"teamColors"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get all participants for this event
	participants, err := getParticipantRepo().GetByEventID(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(participants) == 0 {
		http.Error(w, "No participants to assign to teams", http.StatusBadRequest)
		return
	}

	// Clear existing teams
	getTeamRepo().DeleteBySessionID(r.Context(), eventID)
	getParticipantRepo().ClearTeamID(r.Context(), eventID)

	// Determine number of teams from team names array
	numTeams := len(req.TeamNames)
	if numTeams < 2 {
		numTeams = 2 // Default minimum
	}
	if numTeams > len(participants) {
		numTeams = len(participants)
	}

	// Generate team names
	teamNames := req.TeamNames
	if len(teamNames) < numTeams {
		for i := len(teamNames); i < numTeams; i++ {
			teamNames = append(teamNames, fmt.Sprintf("Team %d", i+1))
		}
	}

	// Generate team colors
	teamColors := req.TeamColors
	defaultColors := []string{"#FF5733", "#33FF57", "#3357FF", "#FF33F5", "#F5FF33", "#33FFF5"}
	if len(teamColors) < numTeams {
		for i := len(teamColors); i < numTeams; i++ {
			teamColors = append(teamColors, defaultColors[i%len(defaultColors)])
		}
	}

	// Shuffle participants for random distribution (using indices)
	indices := rand.Perm(len(participants))

	// Create teams and distribute participants
	teams := make([]*models.Team, numTeams)
	for i := 0; i < numTeams; i++ {
		teamID := generateTeamID()
		teams[i] = &models.Team{
			ID:                   teamID,
			EventSessionID:       eventID,
			Name:                 teamNames[i],
			Color:                teamColors[i],
			MemberParticipantIDs: []string{},
		}
	}

	// Distribute participants
	for i, idx := range indices {
		teamIndex := i % numTeams
		teams[teamIndex].MemberParticipantIDs = append(teams[teamIndex].MemberParticipantIDs, participants[idx].ID)
	}

	// Save teams to database
	for _, team := range teams {
		if err := getTeamRepo().Create(r.Context(), team); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Update participant team assignments
	for _, team := range teams {
		for _, participantID := range team.MemberParticipantIDs {
			getParticipantRepo().UpdateTeamID(r.Context(), participantID, team.ID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

// AssignParticipant handles POST /events/{id}/teams/assign
func AssignParticipant(w http.ResponseWriter, r *http.Request) {
	eventID := extractEventID(r.URL.Path)
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		ParticipantID string `json:"participantId"`
		TeamID        string `json:"teamId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get participant to find current team
	participant, err := getParticipantRepo().GetByID(r.Context(), req.ParticipantID)
	if err != nil {
		http.Error(w, "Participant not found", http.StatusNotFound)
		return
	}

	// Remove from old team if exists
	if participant.TeamID != "" {
		getTeamRepo().RemoveMember(r.Context(), participant.TeamID, req.ParticipantID)
	}

	// Add to new team
	if req.TeamID != "" {
		getTeamRepo().AddMember(r.Context(), req.TeamID, req.ParticipantID)
		getParticipantRepo().UpdateTeamID(r.Context(), req.ParticipantID, req.TeamID)
	} else {
		getParticipantRepo().ClearTeamID(r.Context(), eventID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// AwardTeamPoints handles POST /events/{id}/teams/{teamId}/award
func AwardTeamPoints(w http.ResponseWriter, r *http.Request) {
	// Extract event ID and team ID from path
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]
	teamID := parts[3]

	var req struct {
		Points float64 `json:"points"`
		Reason string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get event to find the teams module ID
	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// Find teams module
	var teamsModuleID string
	for _, module := range event.Modules {
		if module.Type == models.ModuleTypeTeams {
			teamsModuleID = module.ID
			break
		}
	}

	// Create score entry
	scoreEntry := &models.ScoreEntry{
		ID:        generateScoreEntryID(),
		EventID:   eventID,
		TeamID:    teamID,
		ModuleID:  teamsModuleID,
		Category:  models.ScoreCategoryBonus,
		Points:    req.Points,
		Reason:    req.Reason,
		CreatedAt: time.Now(),
	}

	if err := getScoreEntryRepo().Create(r.Context(), scoreEntry); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated team score
	totalScore, _ := getScoreEntryRepo().GetTeamTotalScore(r.Context(), teamID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scoreEntry": scoreEntry,
		"totalScore": totalScore,
	})
}

// ResetTeams handles POST /events/{id}/teams/reset
func ResetTeams(w http.ResponseWriter, r *http.Request) {
	eventID := extractEventID(r.URL.Path)
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	// Delete all teams for this event
	getTeamRepo().DeleteBySessionID(r.Context(), eventID)

	// Clear team assignments from all participants
	getParticipantRepo().ClearTeamID(r.Context(), eventID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GetTeamLeaderboard handles GET /events/{id}/teams/leaderboard
func GetTeamLeaderboard(w http.ResponseWriter, r *http.Request) {
	eventID := extractEventID(r.URL.Path)
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	teams, err := getTeamRepo().GetByEventSessionID(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get scores for each team
	type scoredTeam struct {
		Team  models.Team
		Score float64
	}
	scoredTeams := make([]scoredTeam, len(teams))

	for i, team := range teams {
		score, _ := getScoreEntryRepo().GetTeamTotalScore(r.Context(), team.ID)
		scoredTeams[i] = scoredTeam{
			Team:  team,
			Score: score,
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(scoredTeams)-1; i++ {
		for j := i + 1; j < len(scoredTeams); j++ {
			if scoredTeams[j].Score > scoredTeams[i].Score {
				scoredTeams[i], scoredTeams[j] = scoredTeams[j], scoredTeams[i]
			}
		}
	}

	// Add rank
	result := make([]map[string]interface{}, len(scoredTeams))
	for i, st := range scoredTeams {
		result[i] = map[string]interface{}{
			"rank":        i + 1,
			"team":        st.Team,
			"totalScore":  st.Score,
			"memberCount": len(st.Team.MemberParticipantIDs),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Helper functions

func extractEventID(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, part := range parts {
		if part == "teams" && i > 0 {
			return parts[i-1]
		}
	}
	// Try to get event ID from path like /events/{id}/teams
	if len(parts) >= 2 && parts[0] == "events" {
		return parts[1]
	}
	return ""
}

func generateTeamID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("team_%06d", r.Intn(1000000))
}

func generateScoreEntryID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("se_%06d", r.Intn(1000000))
}

// GetParticipantsForTeam gets all participants for a specific team
func GetParticipantsForTeam(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	var teamID string
	for i, part := range parts {
		if part == "teams" && i+1 < len(parts) {
			teamID = parts[i+1]
			break
		}
	}

	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	participants, err := getParticipantRepo().GetByTeamID(r.Context(), teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}

// SelfAssignTeam handles POST /events/{id}/teams/{teamId}/join
func SelfAssignTeam(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	var eventID, teamID string
	for i, part := range parts {
		if part == "events" && i+1 < len(parts) {
			eventID = parts[i+1]
		}
		if part == "teams" && i+1 < len(parts) {
			teamID = parts[i+1]
		}
	}

	if eventID == "" || teamID == "" {
		http.Error(w, "Event ID and Team ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		ParticipantID string `json:"participantId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add participant to team
	getTeamRepo().AddMember(r.Context(), teamID, req.ParticipantID)
	getParticipantRepo().UpdateTeamID(r.Context(), req.ParticipantID, teamID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GetEventParticipants handles GET /events/{id}/teams/participants
func GetEventParticipants(w http.ResponseWriter, r *http.Request) {
	eventID := extractEventID(r.URL.Path)
	if eventID == "" {
		http.Error(w, "Event ID required", http.StatusBadRequest)
		return
	}

	participants, err := getParticipantRepo().GetByEventID(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}
