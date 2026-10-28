//go:build integration

package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/user/go-expo-sample/backend/postgres"
)

func SetupPostgres(t *testing.T) *postgres.Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container, err := pgcontainer.Run(ctx, "postgres:18.6", pgcontainer.WithDatabase("testdb"), pgcontainer.WithUsername("testuser"), pgcontainer.WithPassword("testpassword"), pgcontainer.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Error(err)
		}
	})
	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return store
}
