package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/user/go-expo-sample/backend/postgres"
)

func runAuthCommand(args []string) error {
	if len(args) < 2 || args[0] != "auth" {
		return errors.New("usage: server auth list-users | promote-owner --user-id <id>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := postgres.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.Migrate(ctx); err != nil {
		return err
	}
	switch args[1] {
	case "list-users":
		if len(args) != 2 {
			return errors.New("usage: server auth list-users")
		}
		for offset := 0; ; offset += 100 {
			users, err := db.AuthUsers(ctx, offset, 100)
			if err != nil {
				return err
			}
			for _, user := range users {
				fmt.Printf("%s\t%s\t%s\t%s\n", user.ID, user.Email, user.Status, user.Role)
			}
			if len(users) < 100 {
				return nil
			}
		}
	case "promote-owner":
		flags := flag.NewFlagSet("promote-owner", flag.ContinueOnError)
		id := flags.String("user-id", "", "existing Google user ID")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		if *id == "" || flags.NArg() != 0 {
			return errors.New("usage: server auth promote-owner --user-id <id>")
		}
		user, err := db.PromoteOwner(ctx, *id)
		if errors.Is(err, postgres.ErrNotFound) {
			return errors.New("Google user not found")
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Promoted %s to owner. Sign in again.\n", user.Email)
		return nil
	default:
		return errors.New("usage: server auth list-users | promote-owner --user-id <id>")
	}
}
