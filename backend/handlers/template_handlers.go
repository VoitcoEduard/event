package handlers

import (
	"context"
	"encoding/json"
	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"time"
)

func SaveTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.Template
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&template); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if template.Name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	template.ID = utils.NewID()
	template.CreatedAt = time.Now()

	collection := db.GetCollection("templates")
	_, err := collection.InsertOne(context.Background(), template)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func GetTemplates(w http.ResponseWriter, r *http.Request) {
	collection := db.GetCollection("templates")
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	templates := make([]models.Template, 0)
	if err := cursor.All(context.Background(), &templates); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// PublishModule finalizes configuration and makes it ready (conceptually "creates table")

func DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /templates/{id}
	path := r.URL.Path
	prefix := "/templates/"
	if len(path) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := path[len(prefix):]

	collection := db.GetCollection("templates")
	_, err := collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Template deleted"})
}

func UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /templates/{id}
	path := r.URL.Path
	prefix := "/templates/"
	if len(path) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := path[len(prefix):]

	var template models.Template
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&template); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	collection := db.GetCollection("templates")
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"name":        template.Name,
			"description": template.Description,
			"data":        template.Data,
			"moduleType":  template.ModuleType,
		},
	}

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Template updated"})
}
