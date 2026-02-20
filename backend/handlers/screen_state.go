package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	screenSubscribers = make(map[string][]chan models.ScreenState)
	screenSubMutex    = sync.Mutex{}
)

func notifyScreenSubscribers(eventID string, state models.ScreenState) {
	screenSubMutex.Lock()
	defer screenSubMutex.Unlock()
	subs := screenSubscribers[eventID]
	for _, ch := range subs {
		select {
		case ch <- state:
		default:
		}
	}
}

func addScreenSubscriber(eventID string) chan models.ScreenState {
	ch := make(chan models.ScreenState, 10)
	screenSubMutex.Lock()
	screenSubscribers[eventID] = append(screenSubscribers[eventID], ch)
	screenSubMutex.Unlock()
	return ch
}

func removeScreenSubscriber(eventID string, ch chan models.ScreenState) {
	screenSubMutex.Lock()
	defer screenSubMutex.Unlock()
	subs := screenSubscribers[eventID]
	for i, sub := range subs {
		if sub == ch {
			screenSubscribers[eventID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// GetScreenState handles GET /events/{eventId}/screen-state
func GetScreenState(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/events/"
	suffix := "/screen-state"
	if len(path) <= len(prefix)+len(suffix) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&event)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	state := event.ScreenState
	if state == nil {
		// Default state
		state = &models.ScreenState{
			CurrentView: "screensaver",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// UpdateScreenState handles PUT /events/{eventId}/screen-state
func UpdateScreenState(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/events/"
	suffix := "/screen-state"
	if len(path) <= len(prefix)+len(suffix) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	var input models.ScreenState
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("events")
	update := bson.M{
		"$set": bson.M{
			"screenState": input,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": id}, update, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify SSE subscribers
	notifyScreenSubscribers(id, input)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}

// ScreenStream handles GET /events/{eventId}/screen-stream for SSE
func ScreenStream(w http.ResponseWriter, r *http.Request) {
	// Parse eventID from URL
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]

	// Ensure the connection supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create and register channel
	ch := addScreenSubscriber(eventID)
	defer removeScreenSubscriber(eventID, ch)

	// Send initial state
	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": eventID}).Decode(&event)
	if err == nil {
		state := event.ScreenState
		if state == nil {
			state = &models.ScreenState{CurrentView: "screensaver"}
		}
		data, _ := json.Marshal(state)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Listen for updates or client disconnect
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case state := <-ch:
			data, _ := json.Marshal(state)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
