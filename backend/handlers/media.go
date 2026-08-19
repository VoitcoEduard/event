package handlers

import (
	"bytes"
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
	"github.com/user/go-expo-sample/backend/utils"
)

var (
	// mediaMemoryCache stores recently served media items in RAM (up to 256MB)
	// so 100 concurrent players fetching round images never hit disk or MongoDB.
	mediaMemoryCache = cache.NewMediaCache(256 * 1024 * 1024)
)

// InvalidateMediaCache removes a media item from the in-memory cache.
func InvalidateMediaCache(id string) {
	mediaMemoryCache.Delete(id)
}

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

	// Pre-populate RAM cache so subsequent player fetches never hit disk/DB
	mediaMemoryCache.Set(mediaFile.ID.Hex(), cache.MediaItem{
		ID:        mediaFile.ID.Hex(),
		Filename:  mediaFile.Filename,
		MimeType:  mediaFile.MimeType,
		Data:      processedData,
		CreatedAt: mediaFile.CreatedAt,
		Size:      int64(len(processedData)),
	})

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
	json.NewEncoder(w).Encode(response)
}

func ServeMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMediaID(r.URL.Path)
	if !ok || id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	// 1. FAST PATH: Check RAM cache first
	if item, found := mediaMemoryCache.Get(id); found {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if item.MimeType != "" {
			w.Header().Set("Content-Type", item.MimeType)
		}
		w.Header().Set("ETag", `"`+id+`"`)
		http.ServeContent(w, r, item.Filename, item.CreatedAt, bytes.NewReader(item.Data))
		return
	}

	// 2. SLOW PATH: Query MongoDB & GridFS
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

	// 3. Save into RAM cache for all subsequent requests
	mediaMemoryCache.Set(id, cache.MediaItem{
		ID:        id,
		Filename:  mediaFile.Filename,
		MimeType:  mediaFile.MimeType,
		Data:      data,
		CreatedAt: mediaFile.CreatedAt,
		Size:      int64(len(data)),
	})

	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if mediaFile.MimeType != "" {
		w.Header().Set("Content-Type", mediaFile.MimeType)
	}
	w.Header().Set("ETag", `"`+id+`"`)
	// Use ServeContent to automatically support Range / chunked requests
	http.ServeContent(w, r, mediaFile.Filename, mediaFile.CreatedAt, bytes.NewReader(data))
}

func DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := parseMediaID(r.URL.Path)
	if !ok || id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	// Invalidate RAM cache
	mediaMemoryCache.Delete(id)

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

// GetScreensavers returns all screensavers from the global library
// URL: /screensavers
func GetScreensavers(w http.ResponseWriter, r *http.Request) {
	collection := db.GetCollection("screensavers")
	var screensavers []models.Screensaver

	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Failed to fetch screensavers", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	if err := cursor.All(context.Background(), &screensavers); err != nil {
		http.Error(w, "Failed to decode screensavers", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responses := make([]models.ScreensaverResponse, len(screensavers))
	for i, s := range screensavers {
		responses[i] = models.ScreensaverResponse{
			ID:           s.ID.Hex(),
			Name:         s.Name,
			MediaID:      s.MediaID.Hex(),
			ThumbnailURL: s.ThumbnailURL,
			CreatedAt:    s.CreatedAt.Format(time.RFC3339),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// CreateScreensaver creates a new screensaver in the global library
// URL: /screensavers
func CreateScreensaver(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20)
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

	// Validate image
	_, err = utils.ValidateImage(data)
	if err != nil {
		http.Error(w, "Invalid image: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Process image (resize, optimize)
	processedData, width, height, mimeType, err := utils.ProcessImage(data)
	if err != nil {
		http.Error(w, "Failed to process image: "+err.Error(), http.StatusInternalServerError)
		return
	}

	ext := mimeToExtension(mimeType)
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

	// Save media file metadata
	mediaFile := models.MediaFile{
		ID:           primitive.NewObjectID(),
		GridFSID:     gridfsID,
		Filename:     filename,
		OriginalName: header.Filename,
		MimeType:     mimeType,
		Size:         int64(len(processedData)),
		Width:        width,
		Height:       height,
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
		Data:      processedData,
		CreatedAt: mediaFile.CreatedAt,
		Size:      int64(len(processedData)),
	})

	// Create screensaver entry
	now := time.Now()
	screensaver := models.Screensaver{
		ID:           primitive.NewObjectID(),
		Name:         name,
		MediaID:      mediaFile.ID,
		ThumbnailURL: "/media/" + mediaFile.ID.Hex() + "/file",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	screensaverCollection := db.GetCollection("screensavers")
	_, err = screensaverCollection.InsertOne(context.Background(), screensaver)
	if err != nil {
		http.Error(w, "Failed to save screensaver: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.ScreensaverResponse{
		ID:           screensaver.ID.Hex(),
		Name:         screensaver.Name,
		MediaID:      screensaver.MediaID.Hex(),
		ThumbnailURL: screensaver.ThumbnailURL,
		CreatedAt:    screensaver.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// DeleteScreensaver deletes a screensaver from the global library
// URL: /screensavers/{id}
func DeleteScreensaver(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path
	prefix := "/screensavers/"
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

	// Find screensaver first
	screensaverCollection := db.GetCollection("screensavers")
	var screensaver models.Screensaver
	err = screensaverCollection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&screensaver)
	if err != nil {
		http.Error(w, "Screensaver not found", http.StatusNotFound)
		return
	}

	// Delete from RAM cache
	mediaMemoryCache.Delete(screensaver.MediaID.Hex())

	// Delete the associated media file
	mediaCollection := db.GetCollection("media")
	var mediaFile models.MediaFile
	err = mediaCollection.FindOne(context.Background(), bson.M{"_id": screensaver.MediaID}).Decode(&mediaFile)
	if err == nil {
		// Delete from GridFS
		bucket := db.GetGridFSBucket()
		if bucket != nil {
			bucket.Delete(mediaFile.GridFSID)
		}
		// Delete media metadata
		mediaCollection.DeleteOne(context.Background(), bson.M{"_id": screensaver.MediaID})
	}

	// Delete screensaver
	_, err = screensaverCollection.DeleteOne(context.Background(), bson.M{"_id": objectID})
	if err != nil {
		http.Error(w, "Failed to delete screensaver: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Screensaver deleted successfully"})
}
