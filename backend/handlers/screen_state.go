package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
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
			subs = append(subs[:i], subs[i+1:]...)
			if len(subs) == 0 {
				delete(screenSubscribers, eventID)
			} else {
				screenSubscribers[eventID] = subs
			}
			close(ch)
			break
		}
	}
}

// GetScreenState handles GET /events/{eventId}/screen-state
func (h *Handler) GetScreenState(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/events/"
	suffix := "/screen-state"
	if len(path) <= len(prefix)+len(suffix) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	event, err := h.deps.DB.GetEvent(r.Context(), id)
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
func (h *Handler) UpdateScreenState(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/events/"
	suffix := "/screen-state"
	if len(path) <= len(prefix)+len(suffix) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	var input models.ScreenState
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	err := h.deps.DB.SetScreenState(r.Context(), id, input)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)

	// Notify SSE subscribers
	notifyScreenSubscribers(id, input)

	// Notify WS subscribers (Big Screen)
	if h.deps.QuizHub != nil {
		h.deps.QuizHub.BroadcastScreenState(id, input)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}

// ScreenStream handles GET /events/{eventId}/screen-stream for SSE
func (h *Handler) ScreenStream(w http.ResponseWriter, r *http.Request) {
	// Parse eventID from PathValue or URL path
	eventID := r.PathValue("eventID")
	if eventID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
		eventID = parts[2]
	}
	if _, err := uuid.Parse(eventID); err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Validate the event exists before allocating a subscriber channel or opening SSE.
	lookupEvent := h.getEvent
	if lookupEvent == nil {
		if h.deps.DB == nil {
			http.Error(w, "Storage unavailable", http.StatusServiceUnavailable)
			return
		}
		lookupEvent = h.deps.DB.GetEvent
	}
	event, err := lookupEvent(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

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

	// Create and register channel
	ch := addScreenSubscriber(eventID)
	defer removeScreenSubscriber(eventID, ch)

	// Send initial state
	state := event.ScreenState
	if state == nil {
		state = &models.ScreenState{CurrentView: "screensaver"}
	}
	data, _ := json.Marshal(state)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

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
