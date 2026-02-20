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
		userID, _ := data["userId"].(string)

		DebugLog("player:join: ModuleID=%s, Nickname=%s, UserID=%s", moduleID, nickname, userID)

		if moduleID == "" || nickname == "" {
			return
		}

		// Determine persistent player ID
		playerID := nickname
		if userID != "" {
			playerID = userID // Use persistent UUID if available
		}

		// Associate client
		client.ModuleID = moduleID
		client.IsHost = false
		client.PlayerID = playerID

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
		session.Players[playerID] = &models.Player{
			ID:       playerID,
			Nickname: nickname,
			Score:    0,
		}

		// Notify session state (cmd:state)
		h.broadcastToSessionLocked(moduleID, ServerMessage{
			Type:    "cmd:state",
			Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
		})

		// Notify session update (state:update) to refresh player list for host
		h.broadcastToSessionLocked(moduleID, ServerMessage{
			Type:    "state:update",
			Payload: session,
		})

	case "host:join":
		// Payload: { "moduleId": "..." }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		moduleID, _ := data["moduleId"].(string)

		DebugLog("host:join received for module: %s", moduleID)

		client.ModuleID = moduleID
		client.IsHost = true

		// Ensure session exists
		if _, exists := h.sessions[moduleID]; !exists {
			DebugLog("host:join: Creating new session for module: %s", moduleID)
			h.sessions[moduleID] = &models.QuizSession{
				ModuleID:      moduleID,
				State:         "lobby",
				Players:       make(map[string]*models.Player),
				PlayerSignals: []models.PlayerSignal{},
				AnswerCounts:  make(map[int]int),
			}
		} else {
			DebugLog("host:join: Session exists for module: %s, state: %s, players: %d", moduleID, h.sessions[moduleID].State, len(h.sessions[moduleID].Players))
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
		DebugLog("host:join: Sent state:update to host for module: %s", moduleID)

	case "bigscreen:join":
		// Payload: { "moduleId": "..." }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		moduleID, _ := data["moduleId"].(string)

		DebugLog("bigscreen:join received for module: %s", moduleID)

		client.ModuleID = moduleID
		client.IsBigScreen = true

		// Ensure session exists (same logic as host)
		if _, exists := h.sessions[moduleID]; !exists {
			DebugLog("bigscreen:join: Creating new session for module: %s", moduleID)
			h.sessions[moduleID] = &models.QuizSession{
				ModuleID:      moduleID,
				State:         "lobby",
				Players:       make(map[string]*models.Player),
				PlayerSignals: []models.PlayerSignal{},
				AnswerCounts:  make(map[int]int),
			}
		}

		// Send current state to bigscreen via cmd:state (same as host)
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
		DebugLog("bigscreen:join: Sent state:update to bigscreen for module: %s", moduleID)

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
			session.EventID = event.ID // Ensure EventID is set
			session.State = "question-presenting"
			session.CurrentIndex = 0
			session.PlayerSignals = []models.PlayerSignal{}
			session.AnswerCounts = make(map[int]int)
			session.Questions = questions // Assigned loaded questions
			session.Batches = CalculateBatches(questions)
			session.StartTime = time.Now().Unix() // Reset timer
			// session.Players = make(map[string]*models.Player) // REMOVED: Do not reset players, keep lobby players

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

			// Notify state change - send 'question-presenting' for presentation
			h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": "question-presenting", "currentIndex": session.CurrentIndex},
			})

			// Auto-transition to question-active after 5 seconds
			go func(modID string, idx int) {
				time.Sleep(5 * time.Second)
				h.mu.Lock()
				defer h.mu.Unlock()

				sess := h.sessions[modID]
				if sess != nil && sess.CurrentIndex == idx && sess.State == "question-presenting" {
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
			DebugLog("player:signal rejected - client is host")
			return
		}
		// Payload: { "questionIndex": 0, "answerIndex": 0, "scoreEarned": 1000, "timeSpent": 5000 }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			DebugLog("player:signal rejected - invalid payload type")
			return
		}
		dataBytes, _ := json.Marshal(data)
		DebugLog("player:signal received: %s", string(dataBytes))

		questionIndexFloat, ok := data["questionIndex"].(float64)
		if !ok {
			DebugLog("player:signal rejected - questionIndex not found or wrong type")
			return
		}
		// Handle both single choice (legacy/default) and multi-select
		var answerIndices []int

		// Check for "answerIndices" (multi-select)
		if indicesInterface, ok := data["answerIndices"].([]interface{}); ok {
			for _, v := range indicesInterface {
				if idxFloat, ok := v.(float64); ok {
					answerIndices = append(answerIndices, int(idxFloat))
				}
			}
		}

		// Fallback to single "answerIndex" if valid (and indices is empty)
		if len(answerIndices) == 0 {
			if answerIndexFloat, ok := data["answerIndex"].(float64); ok {
				answerIndices = append(answerIndices, int(answerIndexFloat))
			}
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
		scoreEarned := int(scoreEarnedFloat)
		timeSpent := int(timeSpentFloat)

		DebugLog("player:signal: User=%s, Q=%d, Score=%d, Time=%d", client.PlayerID, questionIndex, scoreEarned, timeSpent)

		// Extract AnswerValue for slider (optional)
		var answerValue float64
		if val, ok := data["answerValue"].(float64); ok {
			answerValue = val
		}

		session := h.sessions[client.ModuleID]
		if session != nil {
			DebugLog("player:signal - Session found, state: %s, currentIndex: %d, questionWindows: %d",
				session.State, session.CurrentIndex, len(session.QuestionWindows))

			// Check if signal is within acceptance window
			var status string = "on_time"
			if questionIndex < len(session.QuestionWindows) {
				qWindow := session.QuestionWindows[questionIndex]
				result := CanAcceptSignal(qWindow, time.Now().Unix())
				status = result.Reason
				DebugLog("player:signal - QuestionWindow check: index=%d, startTime=%d, endTime=%d, graceEndTime=%d, now=%d, status=%s",
					questionIndex, qWindow.StartTime, qWindow.EndTime, qWindow.GraceEndTime, time.Now().Unix(), status)
			} else {
				DebugLog("player:signal - No QuestionWindow for index %d (windows count: %d)", questionIndex, len(session.QuestionWindows))
			}

			// Create and store signal
			signal := models.PlayerSignal{
				PlayerID:      client.PlayerID,
				QuestionIndex: questionIndex,
				AnswerIndices: answerIndices,
				AnswerValue:   answerValue,
				ScoreEarned:   scoreEarned,
				TimeSpent:     timeSpent,
				Timestamp:     time.Now().Unix(),
				Status:        status,
			}
			// Maintain backward compatibility for single choice in DB if needed,
			// though AnswerIndex is less useful for multi-select.
			if len(answerIndices) == 1 {
				signal.AnswerIndex = answerIndices[0]
			} else {
				signal.AnswerIndex = -1 // mixed/multi
			}

			session.PlayerSignals = append(session.PlayerSignals, signal)

			// Update answer counts for statistics
			if status != "too_late" {
				for _, ansIdx := range answerIndices {
					session.AnswerCounts[ansIdx]++
				}

				// UPDATE PLAYER SCORE
				if player, exists := session.Players[client.PlayerID]; exists {
					player.Score += scoreEarned
				}

				log.Printf("Answer counted for module %s: Question %d, Answers %v. Total answers received/updated.",
					client.ModuleID, questionIndex, answerIndices)
			} else {
				log.Printf("Answer rejected (too_late) for module %s: Question %d, Status %s",
					client.ModuleID, questionIndex, status)
			}

			// Calculate actual answered count for THIS question only
			answeredCount := 0
			for _, signal := range session.PlayerSignals {
				if signal.QuestionIndex == questionIndex {
					answeredCount++
				}
			}

			// Send minimal sync message (answer counts for host histogram)
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type: "sync:minimal",
				Payload: map[string]interface{}{
					"questionIndex": questionIndex,
					"answeredCount": answeredCount,
					"totalPlayers":  len(session.Players),
					"answerCounts":  session.AnswerCounts,
				},
			})

			// Debug log for answered count at end of question
			log.Printf("Question %d ended - answeredCount: %d / %d", questionIndex, answeredCount, len(session.Players))
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

			// Notify state change - send 'question-presenting' first
			session.State = "question-presenting"
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": "question-presenting", "currentIndex": session.CurrentIndex},
			})

			// Auto-transition to question-active after 5 seconds
			go func(modID string, idx int) {
				time.Sleep(5 * time.Second)
				h.mu.Lock()
				defer h.mu.Unlock()

				sess := h.sessions[modID]
				if sess != nil && sess.CurrentIndex == idx && sess.State == "question-presenting" {
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

			// BROADCAST FULL STATE SO HOST GETS UPDATED SCORES/ANSWERS
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "state:update",
				Payload: session,
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

			// BROADCAST FULL STATE SO HOST GETS UPDATED SCORES
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "state:update",
				Payload: session,
			})
		}

	case "host:end_quiz":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			// Save results to DB
			if err := h.saveQuizResults(session); err != nil {
				log.Printf("Error saving quiz results for module %s: %v", client.ModuleID, err)
			}

			session.State = "finished"
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "cmd:state",
				Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
			})

			// BROADCAST FULL STATE SO PLAYERS GET FINAL SCORES/RANKS
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type:    "state:update",
				Payload: session,
			})
		}

	case "host:reset_quiz":
		if !client.IsHost {
			DebugLog("host:reset_quiz rejected - not a host")
			return
		}
		DebugLog("host:reset_quiz received for module: %s", client.ModuleID)
		h.resetSessionLocked(client.ModuleID)
		DebugLog("host:reset_quiz completed for module: %s", client.ModuleID)
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
// This is the public method that acquires the lock - use for REST API calls
func (h *QuizHub) ResetSession(moduleID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.resetSessionLocked(moduleID)
}

// resetSessionLocked is the internal method that assumes the lock is already held
// Use this when calling from within ProcessMessage (which already holds the lock)
func (h *QuizHub) resetSessionLocked(moduleID string) {
	DebugLog("resetSessionLocked: Starting reset for module: %s", moduleID)

	session, exists := h.sessions[moduleID]
	if !exists {
		DebugLog("resetSessionLocked: No session found for module: %s", moduleID)
		return
	}

	DebugLog("resetSessionLocked: Session found, current state: %s, players: %d", session.State, len(session.Players))

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

	DebugLog("resetSessionLocked: Reset complete for module: %s, broadcast sent", moduleID)
}

// saveQuizResults persists the current session state to the database
func (h *QuizHub) saveQuizResults(session *models.QuizSession) error {
	// Robustness: Ensure EventID is present
	if session.EventID == "" {
		collection := db.GetCollection("events")
		var event models.Event
		err := collection.FindOne(context.TODO(), bson.M{"modules.id": session.ModuleID}).Decode(&event)
		if err == nil {
			session.EventID = event.ID
			DebugLog("saveQuizResults: Resolved missing EventID %s for module %s", event.ID, session.ModuleID)
		} else {
			log.Printf("[ERROR] saveQuizResults: Could not resolve EventID for module %s: %v", session.ModuleID, err)
		}
	}

	var playerResults []models.PlayerResult
	for _, p := range session.Players {
		playerResults = append(playerResults, models.PlayerResult{
			ID:       p.ID,
			Nickname: p.Nickname,
			Score:    p.Score,
		})
	}

	var signalResults []models.SignalResult
	for _, s := range session.PlayerSignals {
		signalResults = append(signalResults, models.SignalResult{
			PlayerID:      s.PlayerID,
			QuestionIndex: s.QuestionIndex,
			AnswerValue:   s.AnswerValue,
			ScoreEarned:   s.ScoreEarned,
			TimeSpent:     s.TimeSpent,
		})
	}

	result := models.QuizResult{
		EventID:       session.EventID,
		ModuleID:      session.ModuleID,
		FinishedAt:    time.Now(),
		Players:       playerResults,
		PlayerSignals: signalResults,
	}

	DebugLog("saveQuizResults: Saving result for EventID=%s, ModuleID=%s, Players=%d, Signals=%d",
		result.EventID, result.ModuleID, len(result.Players), len(result.PlayerSignals))

	collection := db.GetCollection("quiz_results")
	_, err := collection.InsertOne(context.TODO(), result)
	if err == nil {
		log.Printf("Successfully saved quiz results for module %s with %d players and %d signals",
			session.ModuleID, len(playerResults), len(signalResults))
	}
	return err
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return nil
	}
	return b
}
