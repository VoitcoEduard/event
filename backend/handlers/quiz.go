package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
)

func (h *Handler) UpdateQuizQuestions(w http.ResponseWriter, r *http.Request) {
	if h.deps.QuizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	eventID := parts[2]
	moduleID := parts[4]

	var questions []models.QuizQuestion
	if err := decodeJSON(w, r, &questions); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if err := validateQuizRanges(questions); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for i, q := range questions {
		if q.ImageID != nil && *q.ImageID != "" {
			fmt.Printf("[UpdateQuizQuestions] Question %d has imageId: %s\n", i, *q.ImageID)
		}
	}

	err := h.deps.DB.PatchModule(r.Context(), eventID, moduleID, map[string]any{"questions": questions})
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(eventID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Questions updated"})
}

func (h *Handler) GetQuizState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if h.deps.QuizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	if eventID == "" || moduleID == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	session := h.deps.QuizHub.GetSession(moduleID)
	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.QuizSession{State: "inactive"})
		return
	}
	if session.EventID != "" && session.EventID != eventID {
		http.NotFound(w, r)
		return
	}
	if !h.isAdmin(r) {
		writeJSON(w, map[string]any{
			"eventId": session.EventID, "moduleId": session.ModuleID,
			"state": session.State, "currentIndex": session.CurrentIndex,
			"startTime": session.StartTime, "questionWindows": session.QuestionWindows,
			"totalPlayers": len(session.Players), "answeredCount": len(session.AnsweredPlayerIDs),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (h *Handler) StartQuiz(w http.ResponseWriter, r *http.Request) {
	if h.deps.QuizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	if eventID == "" || moduleID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 7 {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
		eventID = parts[2]
		moduleID = parts[4]
	}

	lookupEvent := h.getEvent
	if lookupEvent == nil {
		if h.deps.DB == nil {
			http.Error(w, "Database not initialized", http.StatusInternalServerError)
			return
		}
		lookupEvent = h.deps.DB.GetEvent
	}

	event, err := lookupEvent(r.Context(), eventID)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	if event == nil {
		http.NotFound(w, r)
		return
	}
	var questions []models.QuizQuestion
	foundModule := false
	for _, mod := range event.Modules {
		if mod.ID == moduleID {
			foundModule = true
			if mod.Type != models.ModuleTypeQuiz && mod.Type != models.ModuleTypeWrongPictures {
				http.Error(w, "Module is not a quiz", http.StatusBadRequest)
				return
			}
			dataBytes, err := json.Marshal(mod.Data)
			if err != nil {
				http.Error(w, "Invalid quiz module data", http.StatusBadRequest)
				return
			}
			var qData models.QuizModuleData
			if err := json.Unmarshal(dataBytes, &qData); err != nil {
				http.Error(w, "Invalid quiz module data", http.StatusBadRequest)
				return
			}
			questions = qData.Questions
			break
		}
	}

	if !foundModule {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}
	if len(questions) == 0 {
		http.Error(w, "Quiz has no questions", http.StatusBadRequest)
		return
	}

	aspects, err := h.deps.QuizHub.WrongPictureAspects(r.Context(), questions)
	if err != nil {
		http.Error(w, "This image has no dimensions. Reupload it before starting the quiz.", http.StatusBadRequest)
		return
	}

	started, err := h.deps.QuizHub.StartQuizSession(moduleID, eventID, questions, aspects)
	if err != nil {
		switch {
		case errors.Is(err, hub.ErrQuizEventMismatch):
			http.NotFound(w, r)
		case errors.Is(err, hub.ErrQuizFinishing):
			http.Error(w, "Quiz results are being saved", http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(started)
}

func (h *Handler) NextQuestion(w http.ResponseWriter, r *http.Request) {
	if h.deps.QuizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	eventID, moduleID := r.PathValue("eventID"), r.PathValue("moduleID")
	if eventID == "" || moduleID == "" {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 7 {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
			return
		}
		eventID = parts[2]
		moduleID = parts[4]
	}

	updated, err := h.deps.QuizHub.NextQuizQuestion(moduleID, eventID)
	if err != nil {
		switch {
		case errors.Is(err, hub.ErrQuizNotActive):
			http.Error(w, "Quiz not active", http.StatusBadRequest)
		case errors.Is(err, hub.ErrQuizEventMismatch):
			http.NotFound(w, r)
		case errors.Is(err, hub.ErrQuizFinishing):
			http.Error(w, "Quiz results are being saved", http.StatusConflict)
		case errors.Is(err, hub.ErrQuizFinished):
			http.Error(w, "Quiz already finished", http.StatusConflict)
		case errors.Is(err, hub.ErrQuizNoMoreQuestions):
			http.Error(w, "No more questions", http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *Handler) GetQuizResults(w http.ResponseWriter, r *http.Request) {
	if h.deps.QuizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	moduleID := parts[4]
	eventID := parts[2]
	if h.deps.Modules == nil {
		http.Error(w, "Module service unavailable", http.StatusServiceUnavailable)
		return
	}
	if _, err := h.deps.Modules.Get(r.Context(), eventID, moduleID); err != nil {
		if errors.Is(err, database.ErrNotFound) || errors.Is(err, services.ErrModuleNotFound) {
			http.NotFound(w, r)
		} else {
			writeRequestError(w, err, http.StatusServiceUnavailable)
		}
		return
	}

	session := h.deps.QuizHub.GetSession(moduleID)
	if session == nil {
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, no-store")
	if h.isAdmin(r) {
		json.NewEncoder(w).Encode(session)
		return
	}
	response := map[string]any{"state": session.State, "currentIndex": session.CurrentIndex, "playerCount": len(session.Players), "answerCounts": session.AnswerCounts}
	if session.State == "finished" {
		type standing struct {
			Nickname string `json:"nickname"`
			Score    int    `json:"score"`
		}
		leaderboard := make([]standing, 0, len(session.Players))
		for _, player := range session.Players {
			if player != nil {
				leaderboard = append(leaderboard, standing{player.Nickname, player.Score})
			}
		}
		sort.Slice(leaderboard, func(i, j int) bool {
			if leaderboard[i].Score != leaderboard[j].Score {
				return leaderboard[i].Score > leaderboard[j].Score
			}
			return leaderboard[i].Nickname < leaderboard[j].Nickname
		})
		response["leaderboard"] = leaderboard
	}
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) ResetQuiz(w http.ResponseWriter, r *http.Request) {
	if h.deps.QuizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	moduleID := parts[4]

	if !h.deps.QuizHub.ResetSession(moduleID) {
		http.Error(w, "Quiz results are being saved", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Quiz session reset"})
}
