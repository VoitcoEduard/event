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
	ID             string            `json:"id" bson:"_id"`
	EventID        string            `json:"eventId" bson:"eventId"`
	JoinKey        string            `json:"joinKey" bson:"joinKey"`
	DisplayName    string            `json:"displayName" bson:"displayName"`
	NormalizedName string            `json:"normalizedName" bson:"normalizedName"`
	TeamID         string            `json:"teamId,omitempty" bson:"teamId,omitempty"`
	Profile        map[string]any    `json:"profile" bson:"profile"`
	Status         ParticipantStatus `json:"status" bson:"status"`
	CreatedAt      time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt" bson:"updatedAt"`
}
