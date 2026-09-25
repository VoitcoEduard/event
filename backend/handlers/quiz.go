package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
)

// UpdateQuizQuestions - PUT /events/{id}/modules/{moduleId}/quiz/questions
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

	// Debug: log questions with images
	for i, q := range questions {
		if q.ImageID != nil && *q.ImageID != "" {
			fmt.Printf("[UpdateQuizQuestions] Question %d has imageId: %s\n", i, *q.ImageID)
		}
	}

	collection := db.GetCollection("events")
	filter := bson.M{"_id": eventID, "modules.id": moduleID}
	update := bson.M{
		"$set": bson.M{
			"modules.$.data.questions": questions,
		},
	}

	_, err := collection.UpdateOne(r.Context(), filter, update)
	if err != nil {
		writeRequestError(w, err, http.StatusInternalServerError)
		return
	}
	invalidateEventCache(eventID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Questions updated"})
}

// GetQuizState - GET /events/{id}/modules/{moduleId}/quiz/state
func (h *Handler) GetQuizState(w http.ResponseWriter, r *http.Request) {
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

	session := h.deps.QuizHub.GetSession(moduleID)
	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.QuizSession{State: "inactive"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// StartQuiz - POST /events/{id}/modules/{moduleId}/quiz/start
func (h *Handler) StartQuiz(w http.ResponseWriter, r *http.Request) {
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

	// Create session if not exists
	h.deps.QuizHub.CreateSession(moduleID, eventID)

	// Load questions from DB
	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(r.Context(), bson.M{"_id": eventID, "modules.id": moduleID}).Decode(&event)
	var questions []models.QuizQuestion
	if err == nil {
		for _, mod := range event.Modules {
			if mod.ID == moduleID {
				// unmarshal data
				dataBytes, _ := json.Marshal(mod.Data)
				var qData models.QuizModuleData
				json.Unmarshal(dataBytes, &qData)
				questions = qData.Questions
				// Debug: log questions with images
				for i, q := range questions {
					if q.ImageID != nil && *q.ImageID != "" {
						fmt.Printf("[StartQuiz] Question %d has imageId: %s\n", i, *q.ImageID)
					}
				}
				break
			}
		}
	}

	// Update session state - reset players and signals when starting new quiz
	h.deps.QuizHub.UpdateSession(moduleID, func(s *models.QuizSession) {
		s.State = "question"
		s.CurrentIndex = 0
		s.PlayerSignals = []models.PlayerSignal{}
		s.AnswerCounts = make(map[int]int)
		s.Questions = questions
		s.Batches = hub.CalculateBatches(questions)
		s.StartTime = time.Now().Unix()
		// Reset players for new quiz session
		s.Players = make(map[string]*models.Player)

		// Create question window for first question
		if len(questions) > 0 {
			limit := hub.EffectiveTimeLimit(questions[0])
			qWindow := models.QuestionWindow{
				QuestionIndex: int64(s.CurrentIndex),
				StartTime:     s.StartTime,
				EndTime:       s.StartTime + limit,
				GraceEndTime:  s.StartTime + limit + hub.GracePeriodSeconds,
				State:         "active",
			}
			s.QuestionWindows = []models.QuestionWindow{qWindow}
		}
	})

	w.Header().Set("Content-Type", "application/json")
	// Retrieve updated session to return
	json.NewEncoder(w).Encode(h.deps.QuizHub.GetSession(moduleID))
}

// NextQuestion - POST /events/{id}/modules/{moduleId}/quiz/next
func (h *Handler) NextQuestion(w http.ResponseWriter, r *http.Request) {
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

	session := h.deps.QuizHub.GetSession(moduleID)
	if session == nil {
		http.Error(w, "Quiz not active", http.StatusBadRequest)
		return
	}

	h.deps.QuizHub.UpdateSession(moduleID, func(s *models.QuizSession) {
		s.CurrentIndex++
		s.State = "question"
		s.AnswerCounts = make(map[int]int)
		s.StartTime = time.Now().Unix()

		// Create question window for new question
		if s.CurrentIndex < len(s.Questions) {
			limit := hub.EffectiveTimeLimit(s.Questions[s.CurrentIndex])
			qWindow := models.QuestionWindow{
				QuestionIndex: int64(s.CurrentIndex),
				StartTime:     s.StartTime,
				EndTime:       s.StartTime + limit,
				GraceEndTime:  s.StartTime + limit + hub.GracePeriodSeconds,
				State:         "active",
			}
			s.QuestionWindows = append(s.QuestionWindows, qWindow)
		}
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.deps.QuizHub.GetSession(moduleID))
}

// GetQuizResults - GET /events/{id}/modules/{moduleId}/quiz/results
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

	session := h.deps.QuizHub.GetSession(moduleID)
	if session == nil {
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// ResetQuiz - POST /events/{id}/modules/{moduleId}/quiz/reset
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

	// Reset the session
	h.deps.QuizHub.ResetSession(moduleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Quiz session reset"})
}
