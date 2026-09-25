package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/user/go-expo-sample/backend/models"
)

func (h *Handler) SaveTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.Template
	if err := decodeJSON(w, r, &template); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}

	if template.Name == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	err := h.deps.Templates.Create(r.Context(), &template)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

func (h *Handler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.deps.Templates.List(r.Context())
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// PublishModule finalizes configuration and makes it ready (conceptually "creates table")

func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL: /templates/{id}
	path := r.URL.Path
	prefix := "/templates/"
	if len(path) <= len(prefix) {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	id := path[len(prefix):]

	err := h.deps.Templates.Delete(r.Context(), id)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Template deleted"})
}

func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
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

	err := h.deps.Templates.Update(r.Context(), id, &template)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Template updated"})
}
