package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
)

func parseMediaID(path string) (string, bool) {
	prefix := "/media/"
	if len(path) <= len(prefix) {
		return "", false
	}
	rest := path[len(prefix):]
	for i, c := range rest {
		if c == '/' {
			return rest[:i], true
		}
	}
	return rest, true
}

func mimeToExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func UploadMedia(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	eventId := r.FormValue("eventId")
	moduleId := r.FormValue("moduleId")

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = utils.ValidateImage(data)
	if err != nil {
		http.Error(w, "Invalid image: "+err.Error(), http.StatusBadRequest)
		return
	}

	processedData, width, height, mimeType, err := utils.ProcessImage(data)
	if err != nil {
		http.Error(w, "Failed to process image: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ext := mimeToExtension(mimeType)
	fileUUID := uuid.New().String()
	filename := fileUUID + ext

	bucket := db.GetGridFSBucket()
	if bucket == nil {
		http.Error(w, "GridFS not initialized", http.StatusInternalServerError)
		return
	}

	uploadStream, err := bucket.OpenUploadStream(filename)
	if err != nil {
		http.Error(w, "Failed to open upload stream: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer uploadStream.Close()

	_, err = uploadStream.Write(processedData)
	if err != nil {
		http.Error(w, "Failed to write file to GridFS: "+err.Error(), http.StatusInternalServerError)
		return
	}

	gridfsID, ok := uploadStream.FileID.(primitive.ObjectID)
	if !ok {
		http.Error(w, "Failed to get GridFS file ID", http.StatusInternalServerError)
		return
	}

	mediaFile := models.MediaFile{
		ID:           primitive.NewObjectID(),
		GridFSID:     gridfsID,
		Filename:     filename,
		OriginalName: header.Filename,
		MimeType:     mimeType,
		Size:         int64(len(processedData)),
		Width:        width,
		Height:       height,
		EventID:      eventId,
		ModuleID:     moduleId,
		CreatedAt:    time.Now(),
	}

	collection := db.GetCollection("media")
	_, err = collection.InsertOne(context.Background(), mediaFile)
	if err != nil {
		http.Error(w, "Failed to save media metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.MediaResponse{
		ID:           mediaFile.ID.Hex(),
		Filename:     mediaFile.Filename,
		OriginalName: mediaFile.OriginalName,
		MimeType:     mediaFile.MimeType,
		Size:         mediaFile.Size,
		Width:        mediaFile.Width,
		Height:       mediaFile.Height,
		URL:          "/media/" + mediaFile.ID.Hex() + "/file",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func GetMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMediaID(r.URL.Path)
	if !ok || id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("media")
	var mediaFile models.MediaFile
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&mediaFile)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mediaFile)
}

func ServeMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMediaID(r.URL.Path)
	if !ok || id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("media")
	var mediaFile models.MediaFile
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&mediaFile)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	bucket := db.GetGridFSBucket()
	if bucket == nil {
		http.Error(w, "GridFS not initialized", http.StatusInternalServerError)
		return
	}

	downloadStream, err := bucket.OpenDownloadStream(mediaFile.GridFSID)
	if err != nil {
		http.Error(w, "File not found in storage", http.StatusNotFound)
		return
	}
	defer downloadStream.Close()

	data, err := io.ReadAll(downloadStream)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mediaFile.MimeType)
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Write(data)
}

func DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMediaID(r.URL.Path)
	if !ok || id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("media")
	var mediaFile models.MediaFile
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&mediaFile)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	bucket := db.GetGridFSBucket()
	if bucket != nil {
		err = bucket.Delete(mediaFile.GridFSID)
		if err != nil {
			http.Error(w, "Failed to delete file from storage: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_, err = collection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		http.Error(w, "Failed to delete media metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Media deleted successfully"})
}
