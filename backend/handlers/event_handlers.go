package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
)

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var event models.Event
	if err := decodeJSON(w, r, &event); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.deps.Events.Create(r.Context(), &event); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	eventsListCache.Delete(eventsListCacheKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	utils.DebugLog("CreateEvent: Created event %s with %d modules", event.ID, len(event.Modules))
	json.NewEncoder(w).Encode(event)
}

func (h *Handler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if cached, ok := eventsListCache.Get(eventsListCacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	events, err := h.deps.Events.List(r.Context())
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(events)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	eventsListCache.Set(eventsListCacheKey, data)

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	// Check for userId in query
	userID := r.URL.Query().Get("userId")
	if len(userID) > 120 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	admin := h.isAdmin(r)
	cacheResponse := !admin
	w.Header().Set("Cache-Control", "private, no-store")
	cacheKey := eventCacheKey(id, userID)
	if cached, ok := eventResponseCache.Get(cacheKey); ok && !admin {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	eventService := h.deps.Events
	event, err := eventService.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	response := EventWithStatus{
		Event: *event,
	}
	if !admin {
		response.Event = publicEvent(*event)
	}

	if userID != "" {
		participant, points, err := eventService.ParticipantStatus(r.Context(), id, userID)
		if err == nil {
			response.ParticipantStatus = &ParticipantStatusResponse{
				DisplayName: participant.DisplayName, TotalPoints: points,
			}
		} else {
			// Arbitrary identities must not create entries or churn the cache.
			// Also avoid caching a missing status after a transient lookup error.
			cacheResponse = false
		}
	}
	if cards, err := eventService.ActiveCards(r.Context(), id); err == nil {
		response.DynamicCards = cards
	}

	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	if cacheResponse {
		eventResponseCache.Set(cacheKey, data)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	var event models.Event
	if err := decodeJSON(w, r, &event); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.deps.Events.Update(r.Context(), id, &event); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event updated"})
}

func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	mediaAccessMu.Lock()
	defer mediaAccessMu.Unlock()
	var moduleIDs []string
	if h.deps.QuizHub != nil {
		event, err := h.deps.Events.Get(r.Context(), id)
		if err != nil {
			writeRequestError(w, err, http.StatusInternalServerError)
			return
		}
		for _, module := range event.Modules {
			moduleIDs = append(moduleIDs, module.ID)
		}
	}
	if err := h.deps.Events.Delete(r.Context(), id); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if h.deps.QuizHub != nil {
		h.deps.QuizHub.ForgetEventSessions(id, moduleIDs)
	}
	mediaMemoryCache.Clear()
	invalidateEventCache(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event deleted"})
}

func (h *Handler) ConnectToEvent(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Pin string `json:"pin"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	event, err := h.deps.Events.FindByPIN(r.Context(), input.Pin)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if !event.IsActive {
		http.Error(w, "Event is not active", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(publicEvent(*event))
}

func (h *Handler) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("eventID")
	if id == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	var input struct {
		IsActive bool `json:"isActive"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if err := h.deps.Events.SetStatus(r.Context(), id, input.IsActive); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event status updated"})
}
