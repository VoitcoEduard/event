//go:build integration

package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestEventPINAndModuleLifecycle(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	service := NewEventService(db)
	event := &models.Event{Name: "PIN test"}
	if err := service.Create(ctx, event); err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(event.ID); err != nil || len(event.Pin) != 6 {
		t.Fatalf("event ID/PIN: %q %q %v", event.ID, event.Pin, err)
	}
	found, err := service.FindByPIN(ctx, event.Pin)
	if err != nil || found.ID != event.ID {
		t.Fatalf("PIN lookup: %+v %v", found, err)
	}
	if err = db.CreateEvent(ctx, &models.Event{ID: uuid.NewString(), Pin: event.Pin, CreatedAt: time.Now()}); !errors.Is(err, database.ErrConflict) {
		t.Fatalf("duplicate PIN: %v", err)
	}
	slides, err := service.GetSlides(ctx, event.ID)
	if err != nil || len(slides) == 0 || !strings.Contains(slides[0].Elements[0].Content, "?pin="+event.Pin) {
		t.Fatalf("slides: %+v %v", slides, err)
	}
	modules := NewModuleService(db)
	module := &models.Module{Name: "Round", Type: models.ModuleTypeQuiz}
	if err = modules.Add(ctx, event.ID, module); err != nil {
		t.Fatal(err)
	}
	if found, err := modules.SetActive(ctx, event.ID, module.ID, true); err != nil || !found {
		t.Fatalf("activate: %v %v", found, err)
	}
	if found, err := modules.Publish(ctx, event.ID, module.ID, map[string]any{"questions": []any{}}); err != nil || !found {
		t.Fatalf("publish: %v %v", found, err)
	}
	loaded, err := modules.Get(ctx, event.ID, module.ID)
	if err != nil || !loaded.IsActive {
		t.Fatalf("loaded: %+v %v", loaded, err)
	}
	if found, err := modules.Delete(ctx, event.ID, module.ID); err != nil || !found {
		t.Fatalf("delete: %v %v", found, err)
	}
}
