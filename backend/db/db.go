package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

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
		clientOptions := options.Client().ApplyURI(mongoURI)
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
