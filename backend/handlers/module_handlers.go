package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
)

func (h *Handler) AddModule(w http.ResponseWriter, r *http.Request) {
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
	if err := h.deps.Modules.Add(r.Context(), eventID, &module); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(module)
}

func (h *Handler) UpdateModuleStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IsActive bool `json:"isActive"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	found, err := h.deps.Modules.SetActive(r.Context(), eventID, moduleID, input.IsActive)
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

func (h *Handler) PublishModule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	found, err := h.deps.Modules.Publish(r.Context(), eventID, moduleID, input.Data)
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

func (h *Handler) SubmitModuleResponse(w http.ResponseWriter, r *http.Request) {
	var response models.Response
	if err := decodeJSON(w, r, &response); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if err := h.deps.Modules.SubmitResponse(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"), &response); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *Handler) DeleteModule(w http.ResponseWriter, r *http.Request) {
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	mediaAccessMu.Lock()
	defer mediaAccessMu.Unlock()
	found, err := h.deps.Modules.Delete(r.Context(), eventID, moduleID)
	if err != nil {
		if errors.Is(err, services.ErrModuleNotFound) {
			http.Error(w, "Module not found", http.StatusNotFound)
			return
		}
		var teamsErr *services.TeamsError
		if errors.As(err, &teamsErr) || errors.Is(err, services.ErrTeamsConflict) {
			teamsFailure(w, err)
			return
		}
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "Event or module not found", http.StatusNotFound)
		return
	}
	if h.deps.QuizHub != nil {
		h.deps.QuizHub.ForgetModuleSessions(moduleID)
	}
	mediaMemoryCache.Clear()
	invalidateEventCache(eventID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Module deleted"})
}

func (h *Handler) UpdateModule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	eventID := r.PathValue("eventID")
	if err := h.deps.Modules.UpdateData(r.Context(), eventID, r.PathValue("moduleID"), input.Data); err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if input.Data != nil {
		invalidateEventCache(eventID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Module updated"})
}

func (h *Handler) GetModule(w http.ResponseWriter, r *http.Request) {
	module, err := h.deps.Modules.Get(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"))
	if err != nil {
		if errors.Is(err, services.ErrModuleNotFound) || errors.Is(err, database.ErrNotFound) {
			http.Error(w, "Module not found", http.StatusNotFound)
		} else {
			writeRequestError(w, err, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	if h.isAdmin(r) {
		_ = json.NewEncoder(w).Encode(module)
	} else {
		_ = json.NewEncoder(w).Encode(publicModule(*module))
	}
}

func (h *Handler) GetModuleResponses(w http.ResponseWriter, r *http.Request) {
	results, err := h.deps.Modules.Responses(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) || errors.Is(err, services.ErrModuleNotFound) {
			http.Error(w, "Event or module not found", http.StatusNotFound)
		} else {
			writeRequestError(w, err, http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"responses": results})
}

func (h *Handler) UpdateModuleResponse(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Data map[string]any `json:"data"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	found, err := h.deps.Modules.UpdateResponse(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"), r.PathValue("responseID"), input.Data)
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

func (h *Handler) UpdateQuizScore(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Score *int `json:"score"`
	}
	if err := decodeJSON(w, r, &input); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if input.Score == nil {
		http.Error(w, "Score is required", 400)
		return
	}
	found, err := h.deps.Modules.UpdateScore(r.Context(), r.PathValue("eventID"), r.PathValue("moduleID"), r.PathValue("userID"), *input.Score)
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
	invalidateEventCache(r.PathValue("eventID"))
}
