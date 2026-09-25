package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/user/go-expo-sample/backend/models"
)

func (h *Handler) GetScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	config, err := h.deps.Events.GetScreensaver(r.Context(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(config)
}

func (h *Handler) UpdateScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	var input models.ScreensaverConfig
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if err := h.deps.Events.SetScreensaver(r.Context(), id, input); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(input)
}
