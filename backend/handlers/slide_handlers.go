package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/user/go-expo-sample/backend/models"
)

func (h *Handler) GetSlides(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	slides, err := h.deps.Events.GetSlides(r.Context(), eventID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(slides)
}

func (h *Handler) SaveSlides(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventID")
	var input []models.Slide
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if err := h.deps.Events.SaveSlides(r.Context(), eventID, input); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(input)
}
