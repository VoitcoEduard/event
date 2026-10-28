//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestEventSessionPersistenceAndPartialUpdate(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: "123456", CreatedAt: now, Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeForm}}}); err != nil {
		t.Fatal(err)
	}
	repo := NewEventSessionRepository(db)
	v := &models.EventSession{ID: uuid.NewString(), EventID: eventID, Status: models.EventSessionStatusLive, ActiveModuleID: moduleID, ModuleOrder: []string{moduleID}, StartedAt: &now, CreatedAt: now, UpdatedAt: now}
	if err := repo.Create(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, &models.EventSession{ID: v.ID, Status: models.EventSessionStatusPaused, UpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, v.ID)
	if err != nil || got.StartedAt == nil || !got.StartedAt.Equal(now) || len(got.ModuleOrder) != 1 || got.Status != models.EventSessionStatusPaused {
		t.Fatalf("partial update: %+v %v", got, err)
	}
	rows, err := repo.GetByEventID(ctx, eventID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("event sessions: %+v %v", rows, err)
	}
}
