package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func runAuthCommand(args []string) error {
	if len(args) < 2 || args[0] != "auth" {
		return errors.New("usage: server auth list-users | promote-owner --user-id <id>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db.ConnectDB()
	store := auth.Store{DB: db.Database()}
	switch args[1] {
	case "list-users":
		if len(args) != 2 {
			return errors.New("usage: server auth list-users")
		}
		cursor, err := store.Users().Find(ctx, bson.M{"issuer": bson.M{"$ne": "development"}})
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var user auth.User
			if err := cursor.Decode(&user); err != nil {
				return err
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", user.ID, user.Email, user.Status, user.Role)
		}
		return cursor.Err()
	case "promote-owner":
		flags := flag.NewFlagSet("promote-owner", flag.ContinueOnError)
		id := flags.String("user-id", "", "existing Google user ID")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		if *id == "" || flags.NArg() != 0 {
			return errors.New("usage: server auth promote-owner --user-id <id>")
		}
		filter := bson.M{"_id": *id, "issuer": bson.M{"$ne": "development"}}
		var user auth.User
		if err := store.Users().FindOne(ctx, filter).Decode(&user); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return errors.New("Google user not found")
			}
			return err
		}
		if user.Role == "owner" && user.Status == "active" {
			return nil
		}
		if _, err := store.Users().UpdateOne(ctx, filter, bson.M{"$set": bson.M{"role": "owner", "status": "active"}}); err != nil {
			return err
		}
		if _, err := store.Sessions().DeleteMany(ctx, bson.M{"userId": *id}); err != nil {
			return err
		}
		if err := store.AuditAction(ctx, "server-cli", *id, "promote-owner"); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Promoted %s to owner. Sign in again.\n", user.Email)
		return nil
	default:
		return errors.New("usage: server auth list-users | promote-owner --user-id <id>")
	}
}
