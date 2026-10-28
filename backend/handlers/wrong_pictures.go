package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
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

func (h *Handler) PreloadWrongPictures(w http.ResponseWriter, r *http.Request) {
	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	if _, err := uuid.Parse(eventID); err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(moduleID); err != nil {
		http.Error(w, "Invalid module ID", http.StatusBadRequest)
		return
	}
	module, err := h.deps.DB.GetModule(r.Context(), eventID, moduleID)
	if err != nil {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}
	data, err := json.Marshal(module.Data)
	if err != nil {
		http.Error(w, "Failed to encode module data", http.StatusInternalServerError)
		return
	}
	var quizData models.QuizModuleData
	if err = json.Unmarshal(data, &quizData); err != nil {
		http.Error(w, "Failed to parse module data", http.StatusInternalServerError)
		return
	}
	out := PreloadResponse{Questions: []PreloadQuestion{}}
	for i, q := range quizData.Questions {
		if q.Type != models.QuestionTypeWrongPictures || q.ImageID == nil || *q.ImageID == "" {
			continue
		}
		if _, err = uuid.Parse(*q.ImageID); err != nil {
			continue
		}
		media, err := h.deps.DB.Media(r.Context(), *q.ImageID)
		if err != nil {
			continue
		}
		url := "/media/" + media.ID + "/file"
		out.Questions = append(out.Questions, PreloadQuestion{Index: i, ImageURL: url, ThumbURL: url, Width: media.Width, Height: media.Height})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
