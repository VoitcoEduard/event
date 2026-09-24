package handlers

import (
	"context"
	"encoding/json"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"strings"
)

func GetScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/events/"):]
	// Remove /screensaver suffix if present
	id = strings.TrimSuffix(id, "/screensaver")

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Return screensaver config or empty config if not set
	if event.ScreensaverConfig != nil {
		json.NewEncoder(w).Encode(event.ScreensaverConfig)
	} else {
		json.NewEncoder(w).Encode(models.ScreensaverConfig{})
	}
}

// UpdateScreensaverConfig updates the screensaver configuration for an event
// URL: /events/{id}/screensaver

func UpdateScreensaverConfig(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/events/"):]
	// Remove /screensaver suffix if present
	id = strings.TrimSuffix(id, "/screensaver")

	var input models.ScreensaverConfig
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"screensaverConfig": input}}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}

// GetSlides handles GET /events/{eventId}/slides
