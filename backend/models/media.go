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
