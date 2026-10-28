package models

import "time"

type ParticipantStatus string

const (
	ParticipantStatusJoined       ParticipantStatus = "joined"
	ParticipantStatusActive       ParticipantStatus = "active"
	ParticipantStatusLeft         ParticipantStatus = "left"
	ParticipantStatusDisqualified ParticipantStatus = "disqualified"
)

type Participant struct {
	ID             string            `json:"id"`
	EventID        string            `json:"eventId"`
	JoinKey        string            `json:"joinKey"`
	DisplayName    string            `json:"displayName"`
	NormalizedName string            `json:"normalizedName"`
	TeamID         string            `json:"teamId,omitempty"`
	Profile        map[string]any    `json:"profile"`
	Status         ParticipantStatus `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}
