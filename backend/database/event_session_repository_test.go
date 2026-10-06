package database

import (
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func TestBuildEventSessionUpdate_PartialDoesNotNullTimestamps(t *testing.T) {
	partial := &models.EventSession{
		ID:        "ses-1",
		EventID:   "ev-1",
		Status:    models.EventSessionStatusPaused,
		UpdatedAt: time.Now(),
		// StartedAt, EndedAt, ModuleOrder, ModuleStates left nil/empty
	}

	set := buildEventSessionUpdate(partial)

	if _, ok := set["startedAt"]; ok {
		t.Error("startedAt must be omitted when nil, otherwise it nulls stored value")
	}
	if _, ok := set["endedAt"]; ok {
		t.Error("endedAt must be omitted when nil, otherwise it nulls stored value")
	}
	if _, ok := set["moduleOrder"]; ok {
		t.Error("moduleOrder must be omitted when nil")
	}
	if _, ok := set["_id"]; ok {
		t.Error("_id must never be in $set (immutable)")
	}
	if _, ok := set["eventId"]; ok {
		t.Error("eventId must never be in $set (immutable identity)")
	}
	if _, ok := set["createdAt"]; ok {
		t.Error("createdAt must never be in $set (immutable)")
	}
	if set["status"] != models.EventSessionStatusPaused {
		t.Errorf("status = %v, want paused", set["status"])
	}
}

func TestBuildEventSessionUpdate_FullInputIncludesSetFields(t *testing.T) {
	now := time.Now()
	set := buildEventSessionUpdate(&models.EventSession{
		ID:             "ses-1",
		Status:         models.EventSessionStatusLive,
		ActiveModuleID: "m1",
		ModuleOrder:    []string{"m1"},
		StartedAt:      &now,
		UpdatedAt:      now,
	})

	if set["startedAt"] == nil {
		t.Error("startedAt must be included when non-nil")
	}
	if set["activeModuleId"] != "m1" {
		t.Errorf("activeModuleId = %v, want m1", set["activeModuleId"])
	}
}
