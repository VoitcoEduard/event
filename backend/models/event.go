package models

import (
	"time"
)

type Event struct {
	ID          string    `json:"id" bson:"_id"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	IsActive    bool      `json:"isActive" bson:"isActive"`
	CreatedAt   time.Time `json:"createdAt" bson:"createdAt"`
}
