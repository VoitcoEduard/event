//go:build integration

package postgres_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

func TestResponsesDoNotCreateUnnamedParticipants(t *testing.T) {
	ctx, db := startDB(t)
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: "963852", CreatedAt: time.Now().UTC(), Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeForm}}}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		data map[string]any
	}{
		{"missing", map[string]any{"answer": "yes"}},
		{"blank", map[string]any{"step_welcome_name": "  \t "}},
		{"wrong type", map[string]any{"step_welcome_name": 42}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := &models.Response{EventID: eventID, ModuleID: moduleID, UserID: tc.name, Data: tc.data, CreatedAt: time.Now().UTC()}
			if err := db.InsertResponse(ctx, response); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Participant(ctx, eventID, tc.name); !errors.Is(err, postgres.ErrNotFound) {
				t.Fatalf("unnamed participant created: %v", err)
			}
		})
	}
	// A later welcome response registers the same device with its actual name.
	for _, data := range []map[string]any{
		{"step_welcome_name": "  Alice  "},
		{"answer": "ordinary form"},
		{"step_welcome_name": "Replacement"},
	} {
		if err := db.InsertResponse(ctx, &models.Response{EventID: eventID, ModuleID: moduleID, UserID: "missing", Data: data, CreatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
		p, err := db.Participant(ctx, eventID, "missing")
		if err != nil || p.DisplayName != "Alice" || p.NormalizedName != "alice" {
			t.Fatalf("participant identity: %+v %v", p, err)
		}
	}
	responses, err := db.Responses(ctx, eventID, moduleID)
	if err != nil || len(responses) != 6 {
		t.Fatalf("responses were lost: count=%d error=%v", len(responses), err)
	}
}
