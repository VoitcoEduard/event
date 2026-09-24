package db

import (
	"log"

	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var gridfsBucket *gridfs.Bucket

func InitGridFS() {
	if Client == nil {
		log.Fatal("MongoDB client is not initialized. Call ConnectDB() first.")
	}

	db := Database()

	bucket, err := gridfs.NewBucket(db, options.GridFSBucket())
	if err != nil {
		log.Fatalf("Failed to create GridFS bucket: %v", err)
	}

	gridfsBucket = bucket
}

func GetGridFSBucket() *gridfs.Bucket {
	return gridfsBucket
}
