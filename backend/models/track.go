package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Track represents an audio track stored in the global library
type Track struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Name      string             `json:"name" bson:"name"`
	MediaID   primitive.ObjectID `json:"mediaId" bson:"mediaId"`
	URL       string             `json:"url" bson:"url"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

// TrackResponse is the API response for track items
type TrackResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaID   string `json:"mediaId"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
}
