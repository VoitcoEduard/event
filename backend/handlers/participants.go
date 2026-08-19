package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/services"
	"go.mongodb.org/mongo-driver/bson"
)

var participantService *services.ParticipantService

// SetupService lazily initializes the service since db.ConnectDB might run later
func getParticipantService() *services.ParticipantService {
	if participantService == nil {
		participantService = services.NewParticipantService()
	}
	return participantService
}

// JoinEvent handles POST /events/{id}/participants/join
func JoinEvent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]

	var input struct {
		JoinKey     string         `json:"joinKey"`
		DisplayName string         `json:"displayName"`
		Profile     map[string]any `json:"profile"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	svc := getParticipantService()
	participant, err := svc.JoinEvent(r.Context(), eventID, input.JoinKey, input.DisplayName, input.Profile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(participant)
}

// GetParticipants handles GET /events/{id}/participants
func GetParticipants(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]

	svc := getParticipantService()
	participants, err := svc.GetParticipantsByEvent(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}
// UpdateParticipantName handles POST /events/{id}/participants/{userId}/name
func UpdateParticipantName(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]
	userID := parts[3]

	var input struct {
		DisplayName string `json:"displayName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("participants")
	filter := bson.M{"eventId": eventID, "_id": userID}
	update := bson.M{"$set": bson.M{"displayName": input.DisplayName}}

	_, err := collection.UpdateOne(r.Context(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	invalidateParticipantStatus(eventID, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Name updated"})
}
