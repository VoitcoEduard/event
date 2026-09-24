package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
)

func GetScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	config, err := services.NewEventService().GetScreensaver(r.Context(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

func UpdateScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	var input models.ScreensaverConfig
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if err := services.NewEventService().SetScreensaver(r.Context(), id, input); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(input)
}
