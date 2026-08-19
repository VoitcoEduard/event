package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

func ConnectDB() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017" // Default for local dev if not in docker
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Retry connection loop
	for i := 0; i < 5; i++ {
		clientOptions := options.Client().
			ApplyURI(mongoURI).
			SetMaxPoolSize(50).
			SetMinPoolSize(10).
			SetMaxConnIdleTime(5 * time.Minute).
			SetConnectTimeout(10 * time.Second)

		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			fmt.Printf("Failed to create client (attempt %d/5): %v\n", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Ping the primary
		if err := client.Ping(ctx, nil); err != nil {
			fmt.Printf("Failed to ping DB (attempt %d/5): %v\n", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Println("Connected to MongoDB!")
		Client = client
		EnsureIndexes()
		return
	}

	log.Fatal("Could not connect to MongoDB after multiple attempts")
}

func GetCollection(collectionName string) *mongo.Collection {
	if Client == nil {
		log.Fatalf("Database client not initialized. Ensure db.ConnectDB() is called and has finished before calling db.GetCollection. Attempted to get collection: %s", collectionName)
	}
	return Client.Database("eventdb").Collection(collectionName)
}

// EnsureIndexes creates all required indexes across collections to avoid full-collection scans (COLLSCAN)
func EnsureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := map[string][]mongo.IndexModel{
		"events": {
			{Keys: bson.D{{Key: "pin", Value: 1}}},
			{Keys: bson.D{{Key: "modules.id", Value: 1}}},
			{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		},
		"participants": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "_id", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "normalizedName", Value: 1}}},
		},
		"event_participants": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "id", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "normalizedName", Value: 1}}},
		},
		"event_scores": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "participantId", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "moduleId", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
		},
		"dynamic_cards": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "isActive", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
		},
		"event_sessions": {
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
			{Keys: bson.D{{Key: "pin", Value: 1}}},
			{Keys: bson.D{{Key: "status", Value: 1}}},
		},
		"module_attempts": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "moduleId", Value: 1}, {Key: "participantId", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "moduleId", Value: 1}}},
		},
		"event_teams": {
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
		},
		"team_results": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "moduleId", Value: 1}}},
			{Keys: bson.D{{Key: "moduleId", Value: 1}}},
		},
		"responses": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "moduleId", Value: 1}, {Key: "participantId", Value: 1}}},
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "moduleId", Value: 1}}},
		},
		"quiz_results": {
			{Keys: bson.D{{Key: "eventId", Value: 1}, {Key: "moduleId", Value: 1}}},
		},
		"media": {
			{Keys: bson.D{{Key: "eventId", Value: 1}}},
			{Keys: bson.D{{Key: "moduleId", Value: 1}}},
			{Keys: bson.D{{Key: "gridfsID", Value: 1}}},
		},
		"screensavers": {
			{Keys: bson.D{{Key: "mediaId", Value: 1}}},
		},
		"tracks": {
			{Keys: bson.D{{Key: "mediaId", Value: 1}}},
		},
	}

	for collName, modelsList := range indexes {
		coll := GetCollection(collName)
		_, err := coll.Indexes().CreateMany(ctx, modelsList)
		if err != nil {
			fmt.Printf("Warning: failed to create indexes on %s: %v\n", collName, err)
		}
	}
	fmt.Println("MongoDB indexes successfully ensured.")
}
