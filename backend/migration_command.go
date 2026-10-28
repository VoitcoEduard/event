package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/user/go-expo-sample/backend/postgres"
)

func runMigrationCommand(args []string) error {
	if len(args) != 2 || (args[1] != "up" && args[1] != "status") {
		return errors.New("usage: server migrate up | status")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
	if args[1] == "up" {
		if err = db.Migrate(ctx); err != nil {
			return err
		}
	}
	current, target, err := db.Versions(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("schema version %d (application target %d)\n", current, target)
	return nil
}
