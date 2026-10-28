package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/cache"
	"github.com/user/go-expo-sample/backend/filestore"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/utils"
)

var mediaMemoryCache = cache.NewMediaCache(256 * 1024 * 1024)

// Deletion and cache population are serialized so a concurrent read cannot
// repopulate a cache entry after its metadata has been deleted.
var mediaAccessMu sync.RWMutex

func InvalidateMediaCache(id string) { mediaMemoryCache.Delete(id) }
func validMediaID(w http.ResponseWriter, id string) bool {
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return false
	}
	return true
}
func mediaFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, postgres.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}
	writeRequestError(w, err, http.StatusInternalServerError)
}
func mimeToExtension(m string) string {
	switch m {
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
func cacheMedia(m postgres.MediaRecord, data []byte) {
	mediaMemoryCache.Set(m.ID, cache.MediaItem{ID: m.ID, Filename: m.Filename, MimeType: m.MimeType, Data: data, CreatedAt: m.CreatedAt, Size: m.Size})
}
func mediaResponse(m postgres.MediaRecord) models.MediaResponse {
	return models.MediaResponse{ID: m.ID, Filename: m.Filename, OriginalName: m.OriginalName, MimeType: m.MimeType, Size: m.Size, Width: m.Width, Height: m.Height, URL: "/media/" + m.ID + "/file"}
}

func (h *Handler) uploadMedia(w http.ResponseWriter, r *http.Request, kind string) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		name = header.Filename
	}
	mediaType := http.DetectContentType(data)
	width, height := 0, 0
	ext := ".mp3"
	if kind != "track" {
		if _, err = utils.ValidateImage(data); err != nil {
			writeRequestError(w, err, http.StatusBadRequest)
			return
		}
		data, width, height, mediaType, err = utils.ProcessImage(data)
		if err != nil {
			writeRequestError(w, err, http.StatusInternalServerError)
			return
		}
		ext = mimeToExtension(mediaType)
	} else {
		switch mediaType {
		case "audio/wav", "audio/x-wav":
			ext = ".wav"
		case "audio/ogg":
			ext = ".ogg"
		case "audio/mpeg":
			ext = ".mp3"
		}
	}
	in := filestore.Upload{OriginalName: header.Filename, Filename: uuid.NewString() + ext, MimeType: mediaType, Width: width, Height: height, EventID: r.FormValue("eventId"), ModuleID: r.FormValue("moduleId"), Data: data, Kind: kind, Name: name}
	mediaAccessMu.RLock()
	result, err := h.deps.Files.Save(r.Context(), in)
	if err == nil {
		cacheMedia(result.Media, data)
	}
	mediaAccessMu.RUnlock()
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	switch kind {
	case "screensaver":
		v := result.Screen
		_ = json.NewEncoder(w).Encode(models.ScreensaverResponse{ID: v.ID, Name: v.Name, MediaID: v.MediaID, ThumbnailURL: v.ThumbnailURL, CreatedAt: v.CreatedAt.Format(time.RFC3339)})
	case "track":
		v := result.Track
		_ = json.NewEncoder(w).Encode(models.TrackResponse{ID: v.ID, Name: v.Name, MediaID: v.MediaID, URL: v.URL, CreatedAt: v.CreatedAt.Format(time.RFC3339)})
	default:
		_ = json.NewEncoder(w).Encode(mediaResponse(result.Media))
	}
}
func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) { h.uploadMedia(w, r, "") }
func (h *Handler) CreateScreensaver(w http.ResponseWriter, r *http.Request) {
	h.uploadMedia(w, r, "screensaver")
}
func (h *Handler) CreateTrack(w http.ResponseWriter, r *http.Request) { h.uploadMedia(w, r, "track") }
func (h *Handler) GetMedia(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("mediaID")
	if !validMediaID(w, id) {
		return
	}
	m, err := h.deps.DB.Media(r.Context(), id)
	if err != nil {
		mediaFailure(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mediaResponse(m))
}
func (h *Handler) ServeMedia(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("mediaID")
	if !validMediaID(w, id) {
		return
	}
	mediaAccessMu.RLock()
	item, found := mediaMemoryCache.Get(id)
	if !found {
		m, data, err := h.deps.Files.Read(r.Context(), id)
		if err != nil {
			mediaAccessMu.RUnlock()
			mediaFailure(w, err)
			return
		}
		cacheMedia(m, data)
		item = cache.MediaItem{ID: m.ID, Filename: m.Filename, MimeType: m.MimeType, Data: data, CreatedAt: m.CreatedAt, Size: m.Size}
	}
	mediaAccessMu.RUnlock()
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if item.MimeType != "" {
		w.Header().Set("Content-Type", item.MimeType)
	}
	w.Header().Set("ETag", `"`+id+`"`)
	http.ServeContent(w, r, item.Filename, item.CreatedAt, bytes.NewReader(item.Data))
}
func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("mediaID")
	if !validMediaID(w, id) {
		return
	}
	mediaAccessMu.Lock()
	err := h.deps.Files.Delete(r.Context(), id)
	if err == nil {
		InvalidateMediaCache(id)
	}
	mediaAccessMu.Unlock()
	if err != nil {
		mediaFailure(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Media deleted successfully"})
}
func (h *Handler) GetScreensavers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.deps.DB.Screensavers(r.Context())
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]models.ScreensaverResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, models.ScreensaverResponse{ID: v.ID, Name: v.Name, MediaID: v.MediaID, ThumbnailURL: v.ThumbnailURL, CreatedAt: v.CreatedAt.Format(time.RFC3339)})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
func (h *Handler) GetTracks(w http.ResponseWriter, r *http.Request) {
	rows, err := h.deps.DB.Tracks(r.Context())
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	out := make([]models.TrackResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, models.TrackResponse{ID: v.ID, Name: v.Name, MediaID: v.MediaID, URL: v.URL, CreatedAt: v.CreatedAt.Format(time.RFC3339)})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
func (h *Handler) deleteLibrary(w http.ResponseWriter, r *http.Request, kind, id string) {
	if !validMediaID(w, id) {
		return
	}
	mediaAccessMu.Lock()
	mediaID, err := h.deps.Files.DeleteLibrary(r.Context(), kind, id)
	if err == nil {
		InvalidateMediaCache(mediaID)
	}
	mediaAccessMu.Unlock()
	if err != nil {
		mediaFailure(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": strings.TrimSuffix(kind, "s") + " deleted successfully"})
}
func (h *Handler) DeleteScreensaver(w http.ResponseWriter, r *http.Request) {
	h.deleteLibrary(w, r, "screensavers", r.PathValue("screensaverID"))
}
func (h *Handler) DeleteTrack(w http.ResponseWriter, r *http.Request) {
	h.deleteLibrary(w, r, "tracks", r.PathValue("trackID"))
}
