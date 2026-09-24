package handlers

import (
	"time"

	"github.com/user/go-expo-sample/backend/cache"
	"github.com/user/go-expo-sample/backend/models"
)

var (
	// eventResponseCache serves GET /events/{id} responses. It is keyed per
	// (event, user) because the payload includes the caller's participant status.
	// Players poll this endpoint every 3s, so a 15s TTL turns ~33 reads/s at
	// 100 players into a handful of cache misses.
	eventResponseCache = cache.New(15 * time.Second)
	// eventsListCache serves GET /events (admin event list).
	eventsListCache = cache.New(2 * time.Second)
)

const eventsListCacheKey = "evt:list"

func eventCacheKey(eventID, userID string) string {
	return "evt:" + eventID + "|u:" + userID
}

// invalidateEventCache drops every cached response for an event (all users)
// plus the global event list. Call after any write to the event document.
func invalidateEventCache(eventID string) {
	eventResponseCache.DeletePrefix("evt:" + eventID + "|")
	eventsListCache.Delete(eventsListCacheKey)
}

// invalidateParticipantStatus drops the cached response for a single
// participant of an event. Call after writes that only affect that user.
func invalidateParticipantStatus(eventID, userID string) {
	eventResponseCache.Delete(eventCacheKey(eventID, userID))
}

type ParticipantStatusResponse struct {
	DisplayName string  `json:"displayName"`
	TotalPoints float64 `json:"totalPoints"`
}

type EventWithStatus struct {
	models.Event
	ParticipantStatus *ParticipantStatusResponse `json:"participantStatus,omitempty"`
	DynamicCards      []models.DynamicCard       `json:"dynamicCards,omitempty"`
}
