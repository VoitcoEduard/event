package models

type Team struct {
	ID                   string         `json:"id" bson:"_id"`
	EventSessionID       string         `json:"eventSessionId" bson:"eventSessionId"`
	Name                 string         `json:"name" bson:"name"`
	Color                string         `json:"color,omitempty" bson:"color,omitempty"`
	MemberParticipantIDs []string       `json:"memberParticipantIds" bson:"memberParticipantIds"`
	Metadata             map[string]any `json:"metadata,omitempty" bson:"metadata,omitempty"`
}
