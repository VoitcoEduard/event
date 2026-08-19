package handlers

import (
	"context"
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

var (
	quizHub *hub.QuizHub
)

func SetQuizHub(h *hub.QuizHub) {
	quizHub = h
}

// UpdateQuizQuestions - PUT /events/{id}/modules/{moduleId}/quiz/questions
func UpdateQuizQuestions(w http.ResponseWriter, r *http.Request) {
	if quizHub == nil {
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
	if err := json.NewDecoder(r.Body).Decode(&questions); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	_, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	invalidateEventCache(eventID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Questions updated"})
}

// GetQuizState - GET /events/{id}/modules/{moduleId}/quiz/state
func GetQuizState(w http.ResponseWriter, r *http.Request) {
	if quizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	moduleID := parts[4]

	session := quizHub.GetSession(moduleID)
	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.QuizSession{State: "inactive"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// StartQuiz - POST /events/{id}/modules/{moduleId}/quiz/start
func StartQuiz(w http.ResponseWriter, r *http.Request) {
	if quizHub == nil {
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
	quizHub.CreateSession(moduleID, eventID)

	// Load questions from DB
	collection := db.GetCollection("events")
	var event models.Event
	err := collection.FindOne(context.Background(), bson.M{"_id": eventID, "modules.id": moduleID}).Decode(&event)
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
	quizHub.UpdateSession(moduleID, func(s *models.QuizSession) {
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
			qWindow := models.QuestionWindow{
				QuestionIndex: int64(s.CurrentIndex),
				StartTime:     s.StartTime,
				EndTime:       s.StartTime + int64(questions[0].TimeLimit),
				GraceEndTime:  s.StartTime + int64(questions[0].TimeLimit) + hub.GracePeriodSeconds,
				State:         "active",
			}
			s.QuestionWindows = []models.QuestionWindow{qWindow}
		}
	})

	w.Header().Set("Content-Type", "application/json")
	// Retrieve updated session to return
	json.NewEncoder(w).Encode(quizHub.GetSession(moduleID))
}

// NextQuestion - POST /events/{id}/modules/{moduleId}/quiz/next
func NextQuestion(w http.ResponseWriter, r *http.Request) {
	if quizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	moduleID := parts[4]

	session := quizHub.GetSession(moduleID)
	if session == nil {
		http.Error(w, "Quiz not active", http.StatusBadRequest)
		return
	}

	quizHub.UpdateSession(moduleID, func(s *models.QuizSession) {
		s.CurrentIndex++
		s.State = "question"
		s.AnswerCounts = make(map[int]int)
		s.StartTime = time.Now().Unix()

		// Create question window for new question
		if s.CurrentIndex < len(s.Questions) {
			qWindow := models.QuestionWindow{
				QuestionIndex: int64(s.CurrentIndex),
				StartTime:     s.StartTime,
				EndTime:       s.StartTime + int64(s.Questions[s.CurrentIndex].TimeLimit),
				GraceEndTime:  s.StartTime + int64(s.Questions[s.CurrentIndex].TimeLimit) + hub.GracePeriodSeconds,
				State:         "active",
			}
			s.QuestionWindows = append(s.QuestionWindows, qWindow)
		}
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quizHub.GetSession(moduleID))
}

// GetQuizResults - GET /events/{id}/modules/{moduleId}/quiz/results
func GetQuizResults(w http.ResponseWriter, r *http.Request) {
	if quizHub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 7 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	moduleID := parts[4]

	session := quizHub.GetSession(moduleID)
	if session == nil {
		http.Error(w, "No active session", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// ResetQuiz - POST /events/{id}/modules/{moduleId}/quiz/reset
func ResetQuiz(w http.ResponseWriter, r *http.Request) {
	if quizHub == nil {
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
	quizHub.ResetSession(moduleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Quiz session reset"})
}
