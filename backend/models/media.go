package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MediaFile struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	GridFSID     primitive.ObjectID `json:"gridfsId" bson:"gridfsId"`
	Filename     string             `json:"filename" bson:"filename"`
	OriginalName string             `json:"originalName" bson:"originalName"`
	MimeType     string             `json:"mimeType" bson:"mimeType"`
	Size         int64              `json:"size" bson:"size"`
	Width        int                `json:"width" bson:"width"`
	Height       int                `json:"height" bson:"height"`
	EventID      string             `json:"eventId" bson:"eventId"`
	ModuleID     string             `json:"moduleId" bson:"moduleId"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
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
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	Name         string             `json:"name" bson:"name"`
	MediaID      primitive.ObjectID `json:"mediaId" bson:"mediaId"`
	ThumbnailURL string             `json:"thumbnailUrl" bson:"thumbnailUrl"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ScreensaverResponse is the API response for screensaver items
type ScreensaverResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MediaID      string `json:"mediaId"`
	ThumbnailURL string `json:"thumbnailUrl"`
	CreatedAt    string `json:"createdAt"`
}
