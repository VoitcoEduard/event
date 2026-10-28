package models

import "time"

type Team struct {
	ID        string    `json:"id"`
	EventID   string    `json:"eventId"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"createdAt"`
}
