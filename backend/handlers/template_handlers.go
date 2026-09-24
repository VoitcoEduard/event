package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
)

func SaveTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.Template
	if err := decodeJSON(w, r, &template); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if template.Name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	err := services.NewTemplateService().Create(r.Context(), &template)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func GetTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := services.NewTemplateService().List(r.Context())
	if err != nil {
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

	err := services.NewTemplateService().Delete(r.Context(), id)
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
	if err := decodeJSON(w, r, &template); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	err := services.NewTemplateService().Update(r.Context(), id, &template)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Template updated"})
}
