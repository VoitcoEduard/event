package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddModule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type string         `json:"type"`
		Name string         `json:"name"`
		Data map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	module := models.Module{Type: models.ModuleType(input.Type), Name: input.Name, Data: input.Data}
	eventID := r.PathValue("eventID")
	if err := services.NewModuleService().Add(r.Context(), eventID, &module); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(module)
}

func UpdateModuleStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IsActive bool `json:"isActive"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	found, err := services.NewModuleService().SetActive(r.Context(), eventID, moduleID, input.IsActive)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Event or module not found", http.StatusNotFound)
		return
	}
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Module status updated"})
}

func PublishModule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	found, err := services.NewModuleService().Publish(r.Context(), eventID, moduleID, input.Data)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Event or module not found", http.StatusNotFound)
		return
	}
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Module published/configured"})
}

func SubmitModuleResponse(w http.ResponseWriter, r *http.Request) {
	var response models.Response
	if err := decodeJSON(w, r, &response); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if err := services.NewModuleService().SubmitResponse(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"), &response); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func DeleteModule(w http.ResponseWriter, r *http.Request) {
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	found, err := services.NewModuleService().Delete(r.Context(), eventID, moduleID)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Event or module not found", http.StatusNotFound)
		return
	}
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Module deleted"})
}

func UpdateModule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	eventID := r.PathValue("eventID")
	if err := services.NewModuleService().UpdateData(r.Context(), eventID, r.PathValue("moduleID"), input.Data); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if input.Data != nil {
		invalidateEventCache(eventID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Module updated"})
}

func GetModule(w http.ResponseWriter, r *http.Request) {
	module, err := services.NewModuleService().Get(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"))
	if err != nil {
		if errors.Is(err, services.ErrModuleNotFound) || errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Module not found", http.StatusNotFound)
		} else {
			writeRequestError(w, err, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(module)
}

func GetModuleResponses(w http.ResponseWriter, r *http.Request) {
	results, err := services.NewModuleService().Responses(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Event not found", http.StatusNotFound)
		} else {
			writeRequestError(w, err, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"responses": results})
}

func UpdateModuleResponse(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	found, err := services.NewModuleService().UpdateResponse(r.Context(), r.PathValue("responseID"), input.Data)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Response not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Response updated"})
}

func UpdateQuizScore(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Score int `json:"score"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	found, err := services.NewModuleService().UpdateScore(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"), r.PathValue("userID"), input.Score)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Quiz result or player not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Score updated"})
}
