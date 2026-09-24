package models

import "time"

type Team struct {
	ID        string    `json:"id" bson:"_id"`
	EventID   string    `json:"eventId" bson:"eventId"`
	Name      string    `json:"name" bson:"name"`
	Color     string    `json:"color" bson:"color"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}
