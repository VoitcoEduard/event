package utils

import "github.com/google/uuid"

// NewID returns a random identifier for persisted application records.
func NewID() string {
	return uuid.NewString()
}
