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
		state = &models.ScreenState{
			CurrentView: "screensaver",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

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

	notifyScreenSubscribers(id, input)

	if h.deps.QuizHub != nil {
		h.deps.QuizHub.BroadcastScreenState(id, input)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(input)
}

func (h *Handler) ScreenStream(w http.ResponseWriter, r *http.Request) {
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := addScreenSubscriber(eventID)
	defer removeScreenSubscriber(eventID, ch)

	state := event.ScreenState
	if state == nil {
		state = &models.ScreenState{CurrentView: "screensaver"}
	}
	data, _ := json.Marshal(state)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case state := <-ch:
			data, _ := json.Marshal(state)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
