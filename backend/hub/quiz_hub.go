package hub

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	GracePeriodSeconds   = 2
	TargetBatchSize      = 50 * 1024 // 50KB
	MaxQuestionsPerBatch = 50        // Conservative limit
)

// AcceptanceResult represents the result of signal acceptance check
type AcceptanceResult struct {
	Valid  bool   // Whether to include in statistics
	Scored bool   // Whether to award points
	Reason string // "on_time", "grace_period", "too_late"
}

// CanAcceptSignal checks if a signal is within the acceptance window
func CanAcceptSignal(qw models.QuestionWindow, signalTime int64) AcceptanceResult {
	if signalTime <= qw.EndTime {
		return AcceptanceResult{Valid: true, Scored: true, Reason: "on_time"}
	}
	if signalTime <= qw.GraceEndTime {
		return AcceptanceResult{Valid: true, Scored: false, Reason: "grace_period"}
	}
	return AcceptanceResult{Valid: false, Scored: false, Reason: "too_late"}
}

// CalculateBatches splits questions into batches
func CalculateBatches(questions []models.QuizQuestion) []models.QuestionBatch {
	if len(questions) == 0 {
		return []models.QuestionBatch{}
	}

	// For simplicity, we'll use a fixed number of questions per batch (max 50)
	var batches []models.QuestionBatch
	batchSize := Min(MaxQuestionsPerBatch, len(questions))
	totalBatches := (len(questions) + batchSize - 1) / batchSize

	for i := 0; i < totalBatches; i++ {
		start := i * batchSize
		end := Min(start+batchSize, len(questions))

		batch := models.QuestionBatch{
			BatchIndex:   i,
			TotalBatches: totalBatches,
			Questions:    questions[start:end],
			NextBatchAt:  end - 2, // Request next batch when 2 questions remain
			StartIndex:   start,
			EndIndex:     end - 1,
		}
		batches = append(batches, batch)
	}

	return batches
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// QuizHub maintains the set of active clients and broadcasts messages to the
// clients.
type QuizHub struct {
	// Registered clients.
	clients map[*Client]bool

	// Active quiz sessions
	sessions map[string]*models.QuizSession

	// Mutex for thread-safe access
	mu sync.RWMutex
}

func NewQuizHub() *QuizHub {
	return &QuizHub{
		clients:  make(map[*Client]bool),
		sessions: make(map[string]*models.QuizSession),
	}
}

func (h *QuizHub) Run() {
	// No-op for now, or use for cleanup tasks
	// We switched to Mutex based approach for easier state management
}

func (h *QuizHub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
	log.Println("New client registered")
}

func (h *QuizHub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		log.Println("Client unregistered")

		// Handle player disconnect from session logic here if needed
		if client.ModuleID != "" && !client.IsHost {
			// potentially remove player from session or mark as disconnected
		}
	}
}

// Helper to broadcast to specific session
func (h *QuizHub) BroadcastToSession(moduleID string, message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Serialize message
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling broadcast message:", err)
		return
	}

	for client := range h.clients {
		if client.ModuleID == moduleID {
			select {
			case client.send <- jsonMsg:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// ProcessMessage handles incoming WebSocket messages
func (h *QuizHub) ProcessMessage(client *Client, message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Println("Error unmarshalling message:", err)
		return
	}

	// Lock for state updates
	h.mu.Lock()
	defer h.mu.Unlock()

	switch msg.Type {
	case "player:join":
		// Payload: { "moduleId": "...", "nickname": "..." }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		moduleID, _ := data["moduleId"].(string)
		nickname, _ := data["nickname"].(string)

		if moduleID == "" || nickname == "" {
			return
		}

		// Associate client
		client.ModuleID = moduleID
		client.IsHost = false
		client.PlayerID = nickname // Using nickname as ID for now, or generate GUID

		// Get or Create Session (though REST API should have created it, WS might need to get it)
		session, exists := h.sessions[moduleID]
		if !exists {
			// Implicitly create session or error?
			// Better to error if session not started by host?
			// For simplicity, let's create a lobby state
			session = &models.QuizSession{
				ModuleID:      moduleID,
				State:         "lobby",
				Players:       make(map[string]*models.Player),
				PlayerSignals: []models.PlayerSignal{},
				AnswerCounts:  make(map[int]int),
			}
			h.sessions[moduleID] = session
		}

		// Add player
		session.Players[nickname] = &models.Player{
			ID:       nickname,
			Nickname: nickname,
			Score:    0,
		}

		// Notify session
		h.broadcastToSessionLocked(moduleID, ServerMessage{
			Type:    "cmd:state",
			Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
		})

	case "host:join":
		// Payload: { "moduleId": "..." }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		moduleID, _ := data["moduleId"].(string)

		client.ModuleID = moduleID
		client.IsHost = true

		// Ensure session exists
		if _, exists := h.sessions[moduleID]; !exists {
			h.sessions[moduleID] = &models.QuizSession{
				ModuleID:      moduleID,
				State:         "lobby",
				Players:       make(map[string]*models.Player),
				PlayerSignals: []models.PlayerSignal{},
				AnswerCounts:  make(map[int]int),
			}
		}

		// Send current state to host via cmd:state (new protocol)
		jsonMsg, _ := json.Marshal(ServerMessage{
			Type: "cmd:state",
			Payload: map[string]interface{}{
				"state":        h.sessions[moduleID].State,
				"currentIndex": h.sessions[moduleID].CurrentIndex,
			},
		})
		client.send <- jsonMsg

		// Also send full session data via state:update (for player list)
		jsonMsg2, _ := json.Marshal(ServerMessage{
			Type:    "state:update",
			Payload: h.sessions[moduleID],
		})
		client.send <- jsonMsg2

	case "host:start_quiz":
		if !client.IsHost {
			return
		}
		// Load questions from DB (outside lock ideally, but here we are in lock)
		h.mu.Unlock()

		collection := db.GetCollection("events")
		var event models.Event
		var questions []models.QuizQuestion

		// Find event containing this module
		err := collection.FindOne(context.TODO(), bson.M{"modules.id": client.ModuleID}).Decode(&event)
		if err == nil {
			for _, mod := range event.Modules {
				if mod.ID == client.ModuleID {
					// Extract questions
					dataBytes, _ := json.Marshal(mod.Data)
					var qData models.QuizModuleData
					json.Unmarshal(dataBytes, &qData)
					questions = qData.Questions
					break
				}
			}
		} else {
			log.Println("Error finding event for module:", client.ModuleID, err)
		}

		h.mu.Lock()

		// Re-fetch session in case it changed while unlocked (unlikely for start_quiz but good practice)
		session := h.sessions[client.ModuleID]
		if session != nil {
			session.State = "get-ready"
			session.CurrentIndex = 0
			session.PlayerSignals = []models.PlayerSignal{}
			session.AnswerCounts = make(map[int]int)
			session.Questions = questions // Assigned loaded questions
			session.Batches = CalculateBatches(questions)
			session.StartTime = time.Now().Unix()             // Reset timer
			session.Players = make(map[string]*models.Player) // Reset players for new quiz

			// Create question window for first question
			if len(questions) > 0 {
				qWindow := models.QuestionWindow{
					QuestionIndex: int64(session.CurrentIndex),
					StartTime:     session.StartTime,
					EndTime:       session.StartTime + int64(questions[0].TimeLimit),
					GraceEndTime:  session.StartTime + int64(questions[0].TimeLimit) + GracePeriodSeconds,
					State:         "active",
				}
				session.QuestionWindows = []models.QuestionWindow{qWindow}
			}

			// Send first batch to clients
			if len(session.Batches) > 0 {
				h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
					Type:    "data:batch",
					Payload: session.Batches[0],
				})
			}

			// Notify state change - send 'get-ready' for countdown
			h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": "get-ready", "currentIndex": session.CurrentIndex},
			})

			// Auto-transition to question-active after 2 seconds
			go func(modID string, idx int) {
				time.Sleep(2 * time.Second)
				h.mu.Lock()
				defer h.mu.Unlock()

				sess := h.sessions[modID]
				if sess != nil && sess.CurrentIndex == idx && sess.State == "get-ready" {
					sess.State = "question-active"
					sess.StartTime = time.Now().Unix()

					// Update the QuestionWindow start time to now
					if idx < len(sess.QuestionWindows) {
						sess.QuestionWindows[idx].StartTime = sess.StartTime
						sess.QuestionWindows[idx].EndTime = sess.StartTime + int64(sess.Questions[idx].TimeLimit)
						sess.QuestionWindows[idx].GraceEndTime = sess.QuestionWindows[idx].EndTime + GracePeriodSeconds
					}

					h.broadcastToSessionLocked(modID, ServerMessage{
						Type:    "cmd:state",
						Payload: map[string]interface{}{"state": "question-active", "currentIndex": idx, "timeLimit": sess.Questions[idx].TimeLimit},
					})
				}
			}(client.ModuleID, session.CurrentIndex)
		}

	case "player:signal":
		if client.IsHost {
			return
		}
		// Payload: { "questionIndex": 0, "answerIndex": 0, "scoreEarned": 1000, "timeSpent": 5000 }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}

		questionIndexFloat, ok := data["questionIndex"].(float64)
		if !ok {
			return
		}
		answerIndexFloat, ok := data["answerIndex"].(float64)
		if !ok {
			return
		}
		scoreEarnedFloat, ok := data["scoreEarned"].(float64)
		if !ok {
			return
		}
		timeSpentFloat, ok := data["timeSpent"].(float64)
		if !ok {
			return
		}

		questionIndex := int(questionIndexFloat)
		answerIndex := int(answerIndexFloat)
		scoreEarned := int(scoreEarnedFloat)
		timeSpent := int(timeSpentFloat)

		session := h.sessions[client.ModuleID]
		if session != nil {
			// Check if signal is within acceptance window
			var status string = "on_time"
			if questionIndex < len(session.QuestionWindows) {
				qWindow := session.QuestionWindows[questionIndex]
				result := CanAcceptSignal(qWindow, time.Now().Unix())
				status = result.Reason
			}

			// Create and store signal
			signal := models.PlayerSignal{
				PlayerID:      client.PlayerID,
				QuestionIndex: questionIndex,
				AnswerIndex:   answerIndex,
				ScoreEarned:   scoreEarned,
				TimeSpent:     timeSpent,
				Timestamp:     time.Now().Unix(),
				Status:        status,
			}
			session.PlayerSignals = append(session.PlayerSignals, signal)

			// Update answer counts for statistics
			if status != "too_late" {
				session.AnswerCounts[answerIndex]++
				log.Printf("Answer counted for module %s: Question %d, Answer %d. Total for this answer: %d",
					client.ModuleID, questionIndex, answerIndex, session.AnswerCounts[answerIndex])
			} else {
				log.Printf("Answer rejected (too_late) for module %s: Question %d, Status %s",
					client.ModuleID, questionIndex, status)
			}

			// Send minimal sync message (answer counts for host histogram)
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type: "sync:minimal",
				Payload: map[string]interface{}{
					"questionIndex": questionIndex,
					"answeredCount": len(session.PlayerSignals),
					"totalPlayers":  len(session.Players),
					"answerCounts":  session.AnswerCounts,
				},
			})
		}

	case "player:batch_request":
		if client.IsHost {
			return
		}
		// Payload: { "currentBatchIndex": 0 }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		currentBatchIndexFloat, ok := data["currentBatchIndex"].(float64)
		if !ok {
			return
		}
		currentBatchIndex := int(currentBatchIndexFloat)

		session := h.sessions[client.ModuleID]
		if session != nil && currentBatchIndex+1 < len(session.Batches) {
			// Send next batch
			client.send <- mustMarshal(ServerMessage{
				Type:    "data:batch",
				Payload: session.Batches[currentBatchIndex+1],
			})
		}

	case "player:reconnect":
		if client.IsHost {
			return
		}
		// Payload: { "moduleId": "...", "nickname": "...", "lastKnownQuestionIndex": 0, "lastKnownState": "question" }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		moduleID, _ := data["moduleId"].(string)
		nickname, _ := data["nickname"].(string)
		lastKnownIndexFloat, ok := data["lastKnownQuestionIndex"].(float64)
		if !ok {
			return
		}
		lastKnownIndex := int(lastKnownIndexFloat)

		session := h.sessions[moduleID]
		if session == nil {
			client.send <- mustMarshal(ServerMessage{
				Type:    "sync:quiz_ended",
				Payload: map[string]interface{}{"reason": "quiz_not_found"},
			})
			return
		}

		// Calculate what user missed
		currentIndex := session.CurrentIndex
		missedQuestions := []int{}

		for i := lastKnownIndex + 1; i < currentIndex; i++ {
			missedQuestions = append(missedQuestions, i)

			// Mark as skipped (0 points, no answer)
			if session.Players[nickname] != nil {
				session.PlayerSignals = append(session.PlayerSignals, models.PlayerSignal{
					PlayerID:      nickname,
					QuestionIndex: i,
					AnswerIndex:   -1, // -1 = no answer / skipped
					ScoreEarned:   0,
					TimeSpent:     0,
					Timestamp:     time.Now().Unix(),
					Status:        "missed",
				})
			}
		}

		// Determine which batches user needs
		batchesNeeded := []int{}
		for batchIdx, batch := range session.Batches {
			batchStart := batch.StartIndex
			batchEnd := batch.EndIndex

			if (batchStart <= currentIndex && batchEnd >= currentIndex) ||
				(batchStart > currentIndex) {
				batchesNeeded = append(batchesNeeded, batchIdx)
			}
		}

		// Check if user is still active (allow up to 5 missed questions)
		isActive := len(missedQuestions) <= 5

		// Send catch-up response
		client.send <- mustMarshal(ServerMessage{
			Type: "sync:catchup",
			Payload: map[string]interface{}{
				"currentState":         session.State,
				"currentQuestionIndex": currentIndex,
				"missedQuestions":      missedQuestions,
				"score":                session.Players[nickname].Score,
				"isActive":             isActive,
				"batchesNeeded":        batchesNeeded,
			},
		})

	case "host:next_question":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			// session.State = "question-active" // Will be set after delay
			session.CurrentIndex++
			session.AnswerCounts = make(map[int]int)
			session.StartTime = time.Now().Unix()

			// Create question window for new question
			if session.CurrentIndex < len(session.Questions) {
				qWindow := models.QuestionWindow{
					QuestionIndex: int64(session.CurrentIndex),
					StartTime:     session.StartTime,
					EndTime:       session.StartTime + int64(session.Questions[session.CurrentIndex].TimeLimit),
					GraceEndTime:  session.StartTime + int64(session.Questions[session.CurrentIndex].TimeLimit) + GracePeriodSeconds,
					State:         "active",
				}
				session.QuestionWindows = append(session.QuestionWindows, qWindow)
			}

			// Notify state change - send 'get-ready' first
			session.State = "get-ready"
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": "get-ready", "currentIndex": session.CurrentIndex},
			})

			// Auto-transition to question-active after 2 seconds
			go func(modID string, idx int) {
				time.Sleep(2 * time.Second)
				h.mu.Lock()
				defer h.mu.Unlock()

				sess := h.sessions[modID]
				if sess != nil && sess.CurrentIndex == idx && sess.State == "get-ready" {
					sess.State = "question-active"
					sess.StartTime = time.Now().Unix()

					// Update the QuestionWindow start time to now
					if idx < len(sess.QuestionWindows) {
						sess.QuestionWindows[idx].StartTime = sess.StartTime
						sess.QuestionWindows[idx].EndTime = sess.StartTime + int64(sess.Questions[idx].TimeLimit)
						sess.QuestionWindows[idx].GraceEndTime = sess.QuestionWindows[idx].EndTime + GracePeriodSeconds
					}

					h.broadcastToSessionLocked(modID, ServerMessage{
						Type:    "cmd:state",
						Payload: map[string]interface{}{"state": "question-active", "currentIndex": idx, "timeLimit": sess.Questions[idx].TimeLimit},
					})
				}
			}(client.ModuleID, session.CurrentIndex)
		}

	case "host:show_results":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			session.State = "results"

			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
			})
		}

	case "host:show_leaderboard":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			session.State = "leaderboard"
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
			})
		}

	case "host:end_quiz":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			session.State = "finished"
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
			})
		}

	case "host:reset_quiz":
		if !client.IsHost {
			return
		}
		h.ResetSession(client.ModuleID)
	}
}

// Internal helper that assumes Lock is held
func (h *QuizHub) broadcastToSessionLocked(moduleID string, message interface{}) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return
	}

	for client := range h.clients {
		if client.ModuleID == moduleID {
			select {
			case client.send <- jsonMsg:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// Exposed methods for REST handlers to interact with Hub

func (h *QuizHub) GetSession(moduleID string) *models.QuizSession {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessions[moduleID]
}

func (h *QuizHub) UpdateSession(moduleID string, updater func(*models.QuizSession)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, ok := h.sessions[moduleID]
	if ok {
		updater(session)
		// Broadcast update
		h.broadcastToSessionLocked(moduleID, ServerMessage{
			Type:    "state:update",
			Payload: session,
		})
	}
}

func (h *QuizHub) CreateSession(moduleID string, eventID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.sessions[moduleID]; !exists {
		h.sessions[moduleID] = &models.QuizSession{
			EventID:       eventID,
			ModuleID:      moduleID,
			State:         "lobby",
			Players:       make(map[string]*models.Player),
			PlayerSignals: []models.PlayerSignal{},
			AnswerCounts:  make(map[int]int),
			// Questions should be loaded here ideally
		}
	}
}

// ResetSession clears the players and resets the session state for a new quiz
func (h *QuizHub) ResetSession(moduleID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, exists := h.sessions[moduleID]
	if !exists {
		return
	}

	// Reset session state
	session.State = "lobby"
	session.CurrentIndex = 0
	session.Players = make(map[string]*models.Player)
	session.PlayerSignals = []models.PlayerSignal{}
	session.AnswerCounts = make(map[int]int)
	session.Questions = nil
	session.Batches = nil
	session.QuestionWindows = nil
	session.StartTime = 0

	// Broadcast reset to all clients in session
	h.broadcastToSessionLocked(moduleID, ServerMessage{
		Type:    "cmd:state",
		Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
	})

	log.Println("Session reset for module:", moduleID)
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return nil
	}
	return b
}
