package handlers

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (h *Handler) GetEventScores(w http.ResponseWriter, r *http.Request) {
	scores, err := database.LoadEventScores(r.Context(), r.PathValue("eventID"))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}

func (h *Handler) SetManualScoreTarget(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Target string `json:"target"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if input.Target != "players" && input.Target != "teams" && input.Target != "both" {
		http.Error(w, "Choose players, teams or both", 400)
		return
	}
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	if _, err := h.deps.Modules.Get(r.Context(), eventID, moduleID); err != nil {
		http.Error(w, "Module not found", 404)
		return
	}
	if err := h.deps.Modules.UpdateData(r.Context(), eventID, moduleID, map[string]any{"manualScoreTarget": input.Target}); err != nil {
		writeRequestError(w, err, 500)
		return
	}
	invalidateEventCache(eventID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetManualScore(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Points *float64 `json:"points"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, 400)
		return
	}
	if input.Points == nil || math.IsNaN(*input.Points) || math.IsInf(*input.Points, 0) {
		http.Error(w, "A finite points value is required", 400)
		return
	}
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	target, targetID := r.PathValue("target"), r.PathValue("targetID")
	module, err := h.deps.Modules.Get(r.Context(), eventID, moduleID)
	if err != nil {
		http.Error(w, "Module not found", 404)
		return
	}
	allowed, _ := module.Data["manualScoreTarget"].(string)
	if allowed == "" {
		allowed = "both"
	}
	if (target != "players" && target != "teams") || (allowed != "both" && allowed != target) {
		http.Error(w, "This scoring target is disabled for the module", 400)
		return
	}
	collection := "participants"
	if target == "teams" {
		collection = "teams"
	}
	count, err := db.GetCollection(collection).CountDocuments(r.Context(), bson.M{"_id": targetID, "eventId": eventID})
	if err != nil {
		writeRequestError(w, err, 500)
		return
	}
	if count == 0 {
		http.Error(w, "Player or team not found in this event", 404)
		return
	}
	if err := database.NewScoreEntryRepository().SetManual(r.Context(), eventID, moduleID, target, targetID, *input.Points); err != nil {
		writeRequestError(w, err, 500)
		return
	}
	invalidateEventCache(eventID)
	w.WriteHeader(http.StatusNoContent)
}
