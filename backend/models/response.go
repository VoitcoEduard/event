package models

import (
	"time"
)

type Response struct {
	ID        string         `json:"id"`
	EventID   string         `json:"eventId"`
	ModuleID  string         `json:"moduleId"`
	UserID    string         `json:"userId"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"createdAt"`
}
