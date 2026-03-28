package testutils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBContainer holds the testcontainer and mongo client
type MongoDBContainer struct {
	Container *mongodb.MongoDBContainer
	Client    *mongo.Client
	URI       string
}

// SetupMongoDB starts a MongoDB testcontainer and returns a connected client
func SetupMongoDB(t *testing.T) *MongoDBContainer {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}

	return &MongoDBContainer{
		Container: container,
		Client:    client,
		URI:       uri,
	}
}

// TeardownMongoDB stops and removes the MongoDB container
func TeardownMongoDB(t *testing.T, mongoContainer *MongoDBContainer) {
	if mongoContainer == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if mongoContainer.Client != nil {
		if err := mongoContainer.Client.Disconnect(ctx); err != nil {
			t.Logf("Error disconnecting MongoDB client: %v", err)
		}
	}

	if mongoContainer.Container != nil {
		if err := testcontainers.TerminateContainer(mongoContainer.Container.Container); err != nil {
			t.Logf("Error terminating container: %v", err)
		}
	}
}

// GetCollection is a helper to get a collection from the test MongoDB
func (m *MongoDBContainer) GetCollection(database, collection string) *mongo.Collection {
	return m.Client.Database(database).Collection(collection)
}

// NewMongoDBContainerForIntegration creates a container for integration test packages
// This is an alternative that doesn't use t.Fatalf (for use outside test files)
func NewMongoDBContainerForIntegration() (*MongoDBContainer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB container: %w", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoDBContainer{
		Container: container,
		Client:    client,
		URI:       uri,
	}, nil
}
