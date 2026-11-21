//go:build integration

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
)

func TestPublishModulePreservesMetadataKeys(t *testing.T) {
	ctx, db := startDB(t)
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	err := db.CreateEvent(ctx, &models.Event{
		ID:        eventID,
		Pin:       "246810",
		CreatedAt: time.Now().UTC(),
		Modules: []models.Module{{
			ID:   moduleID,
			Type: models.ModuleTypeForm,
			Data: map[string]any{
				"manualScoreTarget": "teams",
				"judgeToken":        "judge-tok-1",
				"timeline":          []any{},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	found, err := db.PublishModule(ctx, eventID, moduleID, map[string]any{
		"timeline":            []any{map[string]any{"id": "step-1"}},
		"mandatoryOnEntry":    true,
		"hideAfterCompletion": false,
	})
	if err != nil || !found {
		t.Fatalf("PublishModule: found=%v err=%v", found, err)
	}

	mod, err := db.GetModule(ctx, eventID, moduleID)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Data["manualScoreTarget"] != "teams" {
		t.Fatalf("expected manualScoreTarget=teams to survive PublishModule, got %#v", mod.Data)
	}
	if mod.Data["judgeToken"] != "judge-tok-1" {
		t.Fatalf("expected judgeToken to survive PublishModule, got %#v", mod.Data)
	}
	if mod.Data["mandatoryOnEntry"] != true {
		t.Fatalf("expected mandatoryOnEntry=true, got %#v", mod.Data)
	}
}
