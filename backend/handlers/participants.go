package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"errors"
	"github.com/user/go-expo-sample/backend/database"
)

// JoinEvent handles POST /events/{id}/participants/join
func (h *Handler) JoinEvent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]

	var input struct {
		UserID      string         `json:"userId"`
		JoinKey     string         `json:"joinKey"`
		DisplayName string         `json:"displayName"`
		Profile     map[string]any `json:"profile"`
	}

	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	svc := h.deps.Participants
	if len(input.UserID) > 120 || len(input.JoinKey) > 120 || len(input.DisplayName) > 120 {
		http.Error(w, "Invalid participant", http.StatusBadRequest)
		return
	}
	participant, err := svc.JoinEventWithID(r.Context(), eventID, input.UserID, input.JoinKey, input.DisplayName, input.Profile)
	if errors.Is(err, database.ErrConflict) {
		http.Error(w, "Participant identity already in use", http.StatusConflict)
		return
	}
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateParticipantStatus(eventID, participant.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(participant)
}

// GetParticipants handles GET /events/{id}/participants
func (h *Handler) GetParticipants(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]

	svc := h.deps.Participants
	participants, err := svc.GetParticipantsByEvent(r.Context(), eventID)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(participants)
}

// UpdateParticipantName handles POST /events/{id}/participants/{userId}/name
func (h *Handler) UpdateParticipantName(w http.ResponseWriter, r *http.Request) {
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

	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(input.DisplayName)
	if len(name) > 120 {
		http.Error(w, "Invalid name", http.StatusBadRequest)
		return
	}
	err := h.deps.DB.SetParticipantName(r.Context(), eventID, userID, name, strings.ToLower(name))
	if errors.Is(err, database.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateParticipantStatus(eventID, userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Name updated"})
}
