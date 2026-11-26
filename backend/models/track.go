package models

import (
	"time"
)

type Track struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	MediaID   string    `json:"mediaId"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
}

type TrackResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	MediaID   string `json:"mediaId"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
}
