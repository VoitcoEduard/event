package handlers

import (
	"context"
	"encoding/json"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"strings"
)

func GetSlides(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	slides := event.Slides
	if len(slides) == 0 {
		joinUrl := "https://example.com/?pin=" + eventID // Note: adjust to your actual base URL if needed contextually, frontend handles the real one usually but for slide editor representation

		// Initialize default slides
		slides = []models.Slide{
			{
				ID:              utils.NewID(),
				Type:            "welcome",
				Name:            "Welcome",
				BackgroundColor: "#1A1A1D",
				Elements: []models.CanvasElement{
					{
						ID:      utils.NewID(),
						Type:    "qr",
						X:       15,
						Y:       25,
						Width:   25,
						Height:  45,
						Content: joinUrl,
					},
					{
						ID:        utils.NewID(),
						Type:      "text",
						X:         45,
						Y:         25,
						Width:     45,
						Height:    20,
						Content:   "Join the Game!",
						FontSize:  70,
						Color:     "#FFFFFF",
						TextAlign: "left",
					},
					{
						ID:        utils.NewID(),
						Type:      "text",
						X:         45,
						Y:         50,
						Width:     45,
						Height:    10,
						Content:   joinUrl,
						FontSize:  45,
						Color:     "#3b82f6",
						TextAlign: "left",
					},
					{
						ID:        utils.NewID(),
						Type:      "text",
						X:         45,
						Y:         65,
						Width:     45,
						Height:    10,
						Content:   "PIN: " + eventID,
						FontSize:  60,
						Color:     "#FFD700",
						TextAlign: "left",
					},
				},
			},
			{ID: utils.NewID(), Type: "screensaver", Name: "Screensaver", BackgroundColor: "#000000"},
		}

		// Save them back immediately
		update := bson.M{"$set": bson.M{"slides": slides}}
		collection.UpdateOne(context.Background(), bson.M{"_id": eventID}, update)
		invalidateEventCache(eventID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slides)
}

// SaveSlides handles PUT /events/{eventId}/slides

func SaveSlides(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	var input []models.Slide
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	for i := range input {
		if input[i].ID == "" {
			input[i].ID = utils.NewID()
		}
	}

	collection := db.GetCollection("events")
	update := bson.M{"$set": bson.M{"slides": input}}

	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": eventID}, update)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(eventID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}
