package handlers

import (
	"context"
	"encoding/json"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"github.com/user/go-expo-sample/backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

func CreateEvent(w http.ResponseWriter, r *http.Request) {
	var event models.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&event); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if err := services.NewEventService().Create(r.Context(), &event); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	eventsListCache.Delete(eventsListCacheKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	utils.DebugLog("CreateEvent: Created event %s with %d modules", event.ID, len(event.Modules))
	json.NewEncoder(w).Encode(event)
}

func GetEvents(w http.ResponseWriter, r *http.Request) {
	if cached, ok := eventsListCache.Get(eventsListCacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	events, err := services.NewEventService().List(r.Context())
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

func GetEvent(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /events/{id}
	path := r.URL.Path
	prefix := "/events/"
	if len(path) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := path[len(prefix):]

	// Check for userId in query
	userID := r.URL.Query().Get("userId")

	cacheKey := eventCacheKey(id, userID)
	if cached, ok := eventResponseCache.Get(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	event, err := services.NewEventService().Get(r.Context(), id)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	response := EventWithStatus{
		Event: *event,
	}

	if userID != "" {
		// Fetch participant info
		partCollection := db.GetCollection("participants")
		var participant models.Participant
		err := partCollection.FindOne(context.Background(), bson.M{"eventId": id, "_id": userID}).Decode(&participant)
		if err == nil {
			status := &ParticipantStatusResponse{
				DisplayName: participant.DisplayName,
			}

			// Fetch total points
			scoreCollection := db.GetCollection("event_scores")
			pipeline := []bson.M{
				{"$match": bson.M{"eventId": id, "participantId": userID}},
				{"$group": bson.M{"_id": nil, "total": bson.M{"$sum": "$points"}}},
			}
			cursor, err := scoreCollection.Aggregate(context.Background(), pipeline)
			if err == nil {
				defer cursor.Close(context.Background())
				if cursor.Next(context.Background()) {
					var result struct {
						Total float64 `bson:"total"`
					}
					if err := cursor.Decode(&result); err == nil {
						status.TotalPoints = result.Total
					}
				}
			}

			response.ParticipantStatus = status
		}
	}

	// Fetch dynamic cards
	cardCollection := db.GetCollection("dynamic_cards")
	cardCursor, err := cardCollection.Find(context.Background(), bson.M{"eventId": id, "isActive": true})
	if err == nil {
		defer cardCursor.Close(context.Background())
		var cards []models.DynamicCard
		if err := cardCursor.All(context.Background(), &cards); err == nil {
			response.DynamicCards = cards
		}
	}

	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	eventResponseCache.Set(cacheKey, data)

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL (naive approach since using net/http)
	// Assuming /events/{id}
	id := r.URL.Path[len("/events/"):]
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	var event models.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&event); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if err := services.NewEventService().Update(r.Context(), id, &event); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event updated"})
}

func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/events/"):]
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	if err := services.NewEventService().Delete(r.Context(), id); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event deleted"})
}

func ConnectToEvent(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Pin string `json:"pin"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	event, err := services.NewEventService().FindByPIN(r.Context(), input.Pin)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if !event.IsActive {
		http.Error(w, "Event is not active", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /events/{id}/status
	// Simple parsing assuming correct format
	path := r.URL.Path
	prefix := "/events/"
	suffix := "/status"
	if len(path) <= len(prefix)+len(suffix) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	id := path[len(prefix) : len(path)-len(suffix)]

	var input struct {
		IsActive bool `json:"isActive"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if err := services.NewEventService().SetStatus(r.Context(), id, input.IsActive); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Event status updated"})
}
