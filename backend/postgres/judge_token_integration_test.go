//go:build integration

package postgres_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
)

func TestConcurrentJudgeTokenCreationPreservesWinner(t *testing.T) {
	ctx, db := startDB(t)
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	eventID, moduleID := uuid.NewString(), uuid.NewString()
	if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: "159753", CreatedAt: time.Now().UTC(), Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeSimpleQuiz, Data: map[string]any{"title": "Keep me"}}}}); err != nil {
		t.Fatal(err)
	}
	const n = 12
	results := make([]string, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = db.EnsureJudgeToken(ctx, eventID, moduleID, fmt.Sprintf("candidate-%d", i))
		}(i)
	}
	close(start)
	wg.Wait()
	for i := range n {
		if errs[i] != nil || results[i] == "" || results[i] != results[0] {
			t.Fatalf("caller %d: token=%q error=%v, want %q", i, results[i], errs[i], results[0])
		}
	}
	module, err := db.GetModule(ctx, eventID, moduleID)
	if err != nil || module.Data["judgeToken"] != results[0] || module.Data["title"] != "Keep me" {
		t.Fatalf("persisted metadata: %+v error=%v", module, err)
	}
}
