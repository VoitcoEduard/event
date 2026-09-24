package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
)

var eventSessionService *services.EventSessionService

func getEventSessionService() *services.EventSessionService {
	if eventSessionService == nil {
		eventSessionService = services.NewEventSessionService()
	}
	return eventSessionService
}

// CreateSession POST /events/{id}/sessions
func CreateSession(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[1]

	var input struct {
		ModuleOrder []string `json:"moduleOrder"`
	}

	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	svc := getEventSessionService()
	session, err := svc.CreateSession(r.Context(), eventID, input.ModuleOrder)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// GetSession GET /sessions/{id}
func GetSession(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	sessionID := parts[1]

	svc := getEventSessionService()
	session, err := svc.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// UpdateSessionStatus PUT /sessions/{id}/status
func UpdateSessionStatus(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	sessionID := parts[1]

	var input struct {
		Status models.EventSessionStatus `json:"status"`
	}

	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	svc := getEventSessionService()
	if err := svc.UpdateSessionStatus(r.Context(), sessionID, input.Status); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Session status updated"})
}

// AdvanceSessionModule PUT /sessions/{id}/active_module
func AdvanceSessionModule(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	sessionID := parts[1]

	var input struct {
		ActiveModuleID string `json:"activeModuleId"`
	}

	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	svc := getEventSessionService()
	if err := svc.AdvanceSessionModule(r.Context(), sessionID, input.ActiveModuleID); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Session active module updated"})
}
