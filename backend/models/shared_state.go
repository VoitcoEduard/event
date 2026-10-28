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
	ID                string           `json:"id"`
	EventSessionID    string           `json:"eventSessionId"` // Links back to the session context
	Key               string           `json:"key"`
	Scope             SharedStateScope `json:"scope"`
	ProducerModuleID  string           `json:"producerModuleId,omitempty"`
	ConsumerModuleIDs []string         `json:"consumerModuleIds,omitempty"`
	ValueType         string           `json:"valueType"` // string, number, boolean, object, array
	Value             any              `json:"value"`
	UpdatedAt         time.Time        `json:"updatedAt"`
}
