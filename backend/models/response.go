package models

import (
	"time"
)

type Response struct {
	ID        string         `json:"id" bson:"_id,omitempty"`
	EventID   string         `json:"eventId" bson:"eventId"`
	ModuleID  string         `json:"moduleId" bson:"moduleId"`
	UserID    string         `json:"userId" bson:"userId"`
	Data      map[string]any `json:"data" bson:"data"`
	CreatedAt time.Time      `json:"createdAt" bson:"createdAt"`
}
