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

// H1: renames via UpdateModule and PublishModuleWithName must persist
// modules.name while preserving manualScoreTarget and judgeToken.
func TestUpdateAndPublishModuleRenamePersist(t *testing.T) {
	ctx, db := startDB(t)
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	err := db.CreateEvent(ctx, &models.Event{
		ID:        eventID,
		Pin:       "246811",
		CreatedAt: time.Now().UTC(),
		Modules: []models.Module{{
			ID: moduleID, Type: models.ModuleTypeForm, Name: "Original",
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

	renameOnly := "Renamed via update"
	if err := db.UpdateModule(ctx, eventID, moduleID, &renameOnly, nil); err != nil {
		t.Fatalf("UpdateModule name-only: %v", err)
	}
	mod, err := db.GetModule(ctx, eventID, moduleID)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Name != renameOnly {
		t.Fatalf("name-only rename: got %q, want %q", mod.Name, renameOnly)
	}
	if mod.Data["manualScoreTarget"] != "teams" || mod.Data["judgeToken"] != "judge-tok-1" {
		t.Fatalf("name-only rename dropped preserved keys: %#v", mod.Data)
	}

	renameAndData := "Renamed with data"
	err = db.UpdateModule(ctx, eventID, moduleID, &renameAndData, map[string]any{
		"timeline": []any{map[string]any{"id": "step-2"}},
	})
	if err != nil {
		t.Fatalf("UpdateModule name+data: %v", err)
	}
	mod, err = db.GetModule(ctx, eventID, moduleID)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Name != renameAndData {
		t.Fatalf("combined rename: got %q, want %q", mod.Name, renameAndData)
	}
	if mod.Data["manualScoreTarget"] != "teams" || mod.Data["judgeToken"] != "judge-tok-1" {
		t.Fatalf("combined rename dropped preserved keys: %#v", mod.Data)
	}

	publishName := "Renamed via publish"
	found, err := db.PublishModuleWithName(ctx, eventID, moduleID, &publishName, map[string]any{
		"mandatoryOnEntry": true,
	})
	if err != nil || !found {
		t.Fatalf("PublishModuleWithName: found=%v err=%v", found, err)
	}
	mod, err = db.GetModule(ctx, eventID, moduleID)
	if err != nil {
		t.Fatal(err)
	}
	if mod.Name != publishName {
		t.Fatalf("publish rename: got %q, want %q", mod.Name, publishName)
	}
	if mod.Data["manualScoreTarget"] != "teams" || mod.Data["judgeToken"] != "judge-tok-1" {
		t.Fatalf("publish rename dropped preserved keys: %#v", mod.Data)
	}
	if mod.Data["mandatoryOnEntry"] != true {
		t.Fatalf("expected mandatoryOnEntry=true, got %#v", mod.Data)
	}
}
