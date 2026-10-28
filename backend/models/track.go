package models

import (
	"time"
)

// Track represents an audio track stored in the global library
type Track struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	MediaID   string    `json:"mediaId"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
}

// TrackResponse is the API response for track items
type TrackResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaID   string `json:"mediaId"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
}
