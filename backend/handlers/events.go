package handlers

import (
	"time"

	"github.com/user/go-expo-sample/backend/cache"
	"github.com/user/go-expo-sample/backend/models"
)

var (
	eventResponseCache = cache.New(15 * time.Second)
	eventsListCache = cache.New(2 * time.Second)
)

func CloseEventCaches() {
	eventResponseCache.Close()
	eventsListCache.Close()
}

const eventsListCacheKey = "evt:list"

func eventCacheKey(eventID, userID string) string {
	return "evt:" + eventID + "|u:" + userID
}

func invalidateEventCache(eventID string) {
	eventResponseCache.DeletePrefix("evt:" + eventID + "|")
	eventsListCache.Delete(eventsListCacheKey)
}

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
