package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PreloadResponse struct {
	Questions []PreloadQuestion `json:"questions"`
}

type PreloadQuestion struct {
	Index    int    `json:"index"`
	ImageURL string `json:"imageUrl"`
	ThumbURL string `json:"thumbUrl"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

func PreloadWrongPictures(w http.ResponseWriter, r *http.Request) {
	// Parse URL: /events/{eventId}/modules/{moduleId}/wrongpictures/preload
	parts := strings.Split(r.URL.Path, "/")
	// /events/123/modules/456/wrongpictures/preload -> length 7
	if len(parts) < 7 {
		http.Error(w, "Invalid path parameters", http.StatusBadRequest)
		return
	}
	moduleIdStr := parts[4]

	moduleId, err := primitive.ObjectIDFromHex(moduleIdStr)
	if err != nil {
		http.Error(w, "Invalid module ID", http.StatusBadRequest)
		return
	}

	var module models.Module
	err = db.GetCollection("modules").FindOne(r.Context(), bson.M{"_id": moduleId}).Decode(&module)
	if err != nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	var quizData models.QuizModuleData
	dataBytes, err := bson.Marshal(module.Data)
	if err != nil {
		http.Error(w, "Failed to encode module data", http.StatusInternalServerError)
		return
	}
	if err := bson.Unmarshal(dataBytes, &quizData); err != nil {
		http.Error(w, "Failed to parse module data", http.StatusInternalServerError)
		return
	}

	// We need to look up media files for these questions
	// Collect all image IDs
	var mediaIDs []primitive.ObjectID
	for _, q := range quizData.Questions {
		if q.Type == models.QuestionTypeWrongPictures && q.ImageID != nil && *q.ImageID != "" {
			oid, e := primitive.ObjectIDFromHex(*q.ImageID)
			if e == nil {
				mediaIDs = append(mediaIDs, oid)
			}
		}
	}

	// Fetch media metadata
	mediaCursor, err := db.GetCollection("media").Find(r.Context(), bson.M{"_id": bson.M{"$in": mediaIDs}})
	var mediaList []models.MediaFile
	if err == nil {
		mediaCursor.All(r.Context(), &mediaList)
	}

	// Map media for fast lookup
	mediaMap := make(map[string]models.MediaFile)
	for _, m := range mediaList {
		mediaMap[m.ID.Hex()] = m
	}

	response := PreloadResponse{
		Questions: make([]PreloadQuestion, 0),
	}

	for i, q := range quizData.Questions {
		if q.Type == models.QuestionTypeWrongPictures && q.ImageID != nil && *q.ImageID != "" {
			media, ok := mediaMap[*q.ImageID]
			if !ok {
				continue
			}
			response.Questions = append(response.Questions, PreloadQuestion{
				Index:    i,
				ImageURL: "/media/" + media.ID.Hex() + "/file",
				ThumbURL: "/media/" + media.ID.Hex() + "/file", // we don't have separate thumbs yet, so just full size
				Width:    media.Width,
				Height:   media.Height,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
