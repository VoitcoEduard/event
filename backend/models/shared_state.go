package models

import "time"

type SharedStateScope string

const (
	SharedStateScopeEvent       SharedStateScope = "event"
	SharedStateScopeParticipant SharedStateScope = "participant"
	SharedStateScopeTeam        SharedStateScope = "team"
	SharedStateScopeModule      SharedStateScope = "module"
)

type SharedStateValue struct {
	ID                string           `json:"id" bson:"_id"`
	EventSessionID    string           `json:"eventSessionId" bson:"eventSessionId"` // Links back to the session context
	Key               string           `json:"key" bson:"key"`
	Scope             SharedStateScope `json:"scope" bson:"scope"`
	ProducerModuleID  string           `json:"producerModuleId,omitempty" bson:"producerModuleId,omitempty"`
	ConsumerModuleIDs []string         `json:"consumerModuleIds,omitempty" bson:"consumerModuleIds,omitempty"`
	ValueType         string           `json:"valueType" bson:"valueType"` // string, number, boolean, object, array
	Value             any              `json:"value" bson:"value"`
	UpdatedAt         time.Time        `json:"updatedAt" bson:"updatedAt"`
}
