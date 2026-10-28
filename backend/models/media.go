package models

import (
	"time"
)

type MediaFile struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"originalName"`
	MimeType     string    `json:"mimeType"`
	Size         int64     `json:"size"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	EventID      string    `json:"eventId"`
	ModuleID     string    `json:"moduleId"`
	CreatedAt    time.Time `json:"createdAt"`
}

type UploadRequest struct {
	EventID  string `json:"eventId"`
	ModuleID string `json:"moduleId"`
}

type MediaResponse struct {
	ID           string `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	Size         int64  `json:"size"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	URL          string `json:"url"`
}

// Screensaver represents a screensaver image stored in the global library
type Screensaver struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MediaID      string    `json:"mediaId"`
	ThumbnailURL string    `json:"thumbnailUrl"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// ScreensaverResponse is the API response for screensaver items
type ScreensaverResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MediaID      string `json:"mediaId"`
	ThumbnailURL string `json:"thumbnailUrl"`
	CreatedAt    string `json:"createdAt"`
}
