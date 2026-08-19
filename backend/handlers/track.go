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

	"github.com/user/go-expo-sample/backend/cache"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
)

// GetTracks returns all tracks from the global library
// URL: /tracks
func GetTracks(w http.ResponseWriter, r *http.Request) {
	collection := db.GetCollection("tracks")
	var tracks []models.Track

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to fetch tracks", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	if err := cursor.All(context.Background(), &tracks); err != nil {
		http.Error(w, "Failed to decode tracks", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]models.TrackResponse, len(tracks))
	for i, t := range tracks {
		responses[i] = models.TrackResponse{
			ID:        t.ID.Hex(),
			Name:      t.Name,
			MediaID:   t.MediaID.Hex(),
			URL:       t.URL,
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// CreateTrack uploads an audio file and creates a new track in the global library
// URL: /tracks
func CreateTrack(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file from form: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get name from form, fallback to original filename
	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine MIME type (basic check or rely on header)
	mimeType := http.DetectContentType(data)
	// Optionally validate audio mime type here if needed

	// Determine extension
	ext := ".mp3" // Default fall back
	switch mimeType {
	case "audio/wav", "audio/x-wav":
		ext = ".wav"
	case "audio/ogg":
		ext = ".ogg"
	case "audio/mpeg":
		ext = ".mp3"
	}

	fileUUID := uuid.New().String()
	filename := fileUUID + ext

	// Upload to GridFS
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

	_, err = uploadStream.Write(data)
	if err != nil {
		http.Error(w, "Failed to write file to GridFS: "+err.Error(), http.StatusInternalServerError)
		return
	}

	gridfsID, ok := uploadStream.FileID.(primitive.ObjectID)
	if !ok {
		http.Error(w, "Failed to get GridFS file ID", http.StatusInternalServerError)
		return
	}

	// Save media file metadata
	mediaFile := models.MediaFile{
		ID:           primitive.NewObjectID(),
		GridFSID:     gridfsID,
		Filename:     filename,
		OriginalName: header.Filename,
		MimeType:     mimeType,
		Size:         int64(len(data)),
		Width:        0, // Audio has no dimensions
		Height:       0,
		CreatedAt:    time.Now(),
	}

	mediaCollection := db.GetCollection("media")
	_, err = mediaCollection.InsertOne(context.Background(), mediaFile)
	if err != nil {
		http.Error(w, "Failed to save media metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Pre-populate RAM cache
	mediaMemoryCache.Set(mediaFile.ID.Hex(), cache.MediaItem{
		ID:        mediaFile.ID.Hex(),
		Filename:  mediaFile.Filename,
		MimeType:  mediaFile.MimeType,
		Data:      data,
		CreatedAt: mediaFile.CreatedAt,
		Size:      int64(len(data)),
	})

	// Create track entry
	track := models.Track{
		ID:        primitive.NewObjectID(),
		Name:      name,
		MediaID:   mediaFile.ID,
		URL:       "/media/" + mediaFile.ID.Hex() + "/file",
		CreatedAt: time.Now(),
	}

	trackCollection := db.GetCollection("tracks")
	_, err = trackCollection.InsertOne(context.Background(), track)
	if err != nil {
		http.Error(w, "Failed to save track: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.TrackResponse{
		ID:        track.ID.Hex(),
		Name:      track.Name,
		MediaID:   track.MediaID.Hex(),
		URL:       track.URL,
		CreatedAt: track.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// DeleteTrack deletes a track from the global library
// URL: /tracks/{id}
func DeleteTrack(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path
	prefix := "/tracks/"
	if len(id) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id = id[len(prefix):]

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	// Find track first
	trackCollection := db.GetCollection("tracks")
	var track models.Track
	err = trackCollection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&track)
	if err != nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	// Invalidate RAM cache
	InvalidateMediaCache(track.MediaID.Hex())

	// Delete the associated media file
	mediaCollection := db.GetCollection("media")
	var mediaFile models.MediaFile
	err = mediaCollection.FindOne(context.Background(), bson.M{"_id": track.MediaID}).Decode(&mediaFile)
	if err == nil {
		// Delete from GridFS
		bucket := db.GetGridFSBucket()
		if bucket != nil {
			bucket.Delete(mediaFile.GridFSID)
		}
		// Delete media metadata
		mediaCollection.DeleteOne(context.Background(), bson.M{"_id": track.MediaID})
	}

	// Delete track
	_, err = trackCollection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		http.Error(w, "Failed to delete track: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Track deleted successfully"})
}
