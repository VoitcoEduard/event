package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
	"go.mongodb.org/mongo-driver/bson"
)

var teamRepo *database.TeamRepository

func getTeamRepo() *database.TeamRepository {
	if teamRepo == nil {
		teamRepo = database.NewTeamRepository()
	}
	return teamRepo
}

func enableCors(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
	}
}

func GetTeams(w http.ResponseWriter, r *http.Request) {
	enableCors(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	teams, err := getTeamRepo().GetByEventID(r.Context(), eventID)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	if teams == nil {
		teams = []models.Team{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

func CreateTeam(w http.ResponseWriter, r *http.Request) {
	enableCors(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	var input struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		http.Error(w, "Team name is required", http.StatusBadRequest)
		return
	}

	if input.Color == "" {
		input.Color = "#3B82F6"
	}

	team := &models.Team{
		ID:        utils.NewID(),
		EventID:   eventID,
		Name:      input.Name,
		Color:     input.Color,
		CreatedAt: time.Now(),
	}

	if err := getTeamRepo().Create(r.Context(), team); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	BroadcastTeamUpdate(eventID, "team:created", team)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(team)
}

func UpdateTeam(w http.ResponseWriter, r *http.Request) {
	enableCors(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]
	teamID := parts[4]

	var input struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	team, err := getTeamRepo().GetByID(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if input.Name != "" {
		team.Name = input.Name
	}
	if input.Color != "" {
		team.Color = input.Color
	}

	if err := getTeamRepo().Update(r.Context(), team); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	BroadcastTeamUpdate(eventID, "team:updated", team)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

func DeleteTeam(w http.ResponseWriter, r *http.Request) {
	enableCors(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 6 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]
	teamID := parts[4]

	_, err := getTeamRepo().GetByID(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if err := getTeamRepo().Delete(r.Context(), teamID); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	clearParticipantsTeamAssignment(r.Context(), teamID)

	BroadcastTeamUpdate(eventID, "team:deleted", map[string]string{"teamId": teamID})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Team deleted"})
}

func AssignParticipant(w http.ResponseWriter, r *http.Request) {
	enableCors(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	var input struct {
		ParticipantID string `json:"participantId"`
		TeamID        string `json:"teamId"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if input.ParticipantID == "" || input.TeamID == "" {
		http.Error(w, "participantId and teamId are required", http.StatusBadRequest)
		return
	}

	team, err := getTeamRepo().GetByID(r.Context(), input.TeamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	collection := db.GetCollection("participants")
	filter := bson.M{"_id": input.ParticipantID, "eventId": eventID}
	update := bson.M{"$set": bson.M{"teamId": input.TeamID, "updatedAt": time.Now()}}

	result, err := collection.UpdateOne(r.Context(), filter, update)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	if result.MatchedCount == 0 {
		http.Error(w, "Participant not found", http.StatusNotFound)
		return
	}

	var participant models.Participant
	collection.FindOne(r.Context(), bson.M{"_id": input.ParticipantID}).Decode(&participant)

	BroadcastTeamUpdate(eventID, "participant:assigned", map[string]interface{}{
		"participant": participant,
		"team":        team,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Participant assigned to team",
		"participant": participant,
		"team":        team,
	})
}

func AutoAssignParticipants(w http.ResponseWriter, r *http.Request) {
	enableCors(w, r)
	if r.Method == "OPTIONS" {
		return
	}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	teams, err := getTeamRepo().GetByEventID(r.Context(), eventID)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	if len(teams) == 0 {
		http.Error(w, "No teams exist for this event", http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("participants")
	cursor, err := collection.Find(r.Context(), bson.M{"eventId": eventID, "teamId": ""})
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var unassigned []models.Participant
	if err := cursor.All(r.Context(), &unassigned); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	if len(unassigned) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":       "No unassigned participants",
			"teams":         teams,
			"assignedCount": 0,
		})
		return
	}

	now := time.Now()
	for i, participant := range unassigned {
		teamIndex := i % len(teams)
		targetTeam := teams[teamIndex]

		update := bson.M{"$set": bson.M{
			"teamId":    targetTeam.ID,
			"updatedAt": now,
		}}
		collection.UpdateOne(r.Context(), bson.M{"_id": participant.ID}, update)

		participant.TeamID = targetTeam.ID
		unassigned[i] = participant
	}

	BroadcastTeamUpdate(eventID, "team:auto_assigned", map[string]interface{}{
		"teams":         teams,
		"participants":  unassigned,
		"assignedCount": len(unassigned),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Auto-assignment complete",
		"assignedCount": len(unassigned),
		"teams":         teams,
		"participants":  unassigned,
	})
}

func clearParticipantsTeamAssignment(ctx context.Context, teamID string) {
	collection := db.GetCollection("participants")
	filter := bson.M{"teamId": teamID}
	update := bson.M{"$set": bson.M{"teamId": "", "updatedAt": time.Now()}}
	collection.UpdateMany(ctx, filter, update)
}

func BroadcastTeamUpdate(eventID string, messageType string, payload interface{}) {
	if quizHub == nil {
		return
	}

	serverMsg := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    messageType,
		Payload: payload,
	}

	quizHub.BroadcastToEvent(eventID, serverMsg)
}
