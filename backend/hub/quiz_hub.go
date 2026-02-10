package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

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
				ModuleID: moduleID,
				State:    "lobby",
				Players:  make(map[string]*models.Player),
				Answers:  make(map[string]int),
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

		client.ModuleID = moduleID
		client.IsHost = true

		// Ensure session exists
		if _, exists := h.sessions[moduleID]; !exists {
			h.sessions[moduleID] = &models.QuizSession{
				ModuleID: moduleID,
				State:    "lobby",
				Players:  make(map[string]*models.Player),
				Answers:  make(map[string]int),
			}
		}

		// Send current state to host
		jsonMsg, _ := json.Marshal(ServerMessage{
			Type:    "state:update",
			Payload: h.sessions[moduleID],
		})
		client.send <- jsonMsg

	case "host:start_quiz":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			session.State = "question"
			session.CurrentIndex = 0
			session.Answers = make(map[string]int)
			session.StartTime = time.Now().Unix() // Reset timer

			// Broadcast first question separately if we want to hide future questions?
			// For now, full session sync.
			h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
				Type:    "state:update",
				Payload: session,
			})
		}

	case "player:answer":
		if client.IsHost {
			return
		}
		// Payload: { "answerIndex": 0 }
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		// JSON numbers are float64
		ansFloat, ok := data["answerIndex"].(float64)
		if !ok {
			return
		}
		answerIndex := int(ansFloat)

		session := h.sessions[client.ModuleID]
		// Determine if late? (Client enforced, but server check good too)
		if session != nil && session.State == "question" {
			// Record answer if not already answered
			if _, answered := session.Answers[client.PlayerID]; !answered {
				session.Answers[client.PlayerID] = answerIndex
				// Broadcast update (so host sees count go up)
				h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
					Type:    "state:update",
					Payload: session,
				})
			}
		}

	case "host:next_question":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			session.State = "question"
			session.CurrentIndex++
			session.Answers = make(map[string]int)
			session.StartTime = time.Now().Unix()

			h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
				Type:    "state:update",
				Payload: session,
			})
		}

	case "host:show_results":
		if !client.IsHost {
			return
		}
		session := h.sessions[client.ModuleID]
		if session != nil {
			session.State = "results"

			// --- SCORING LOGIC ---
			// We need to know the correct answer.
			// Look up current question from session.Questions
			if session.CurrentIndex < len(session.Questions) {
				currentQ := session.Questions[session.CurrentIndex]
				correctIdx := currentQ.CorrectIndex

				// Iterate over players who answered
				for playerID, answerIdx := range session.Answers {
					if answerIdx == correctIdx {
						if p, ok := session.Players[playerID]; ok {
							// Simple scoring: +1000 points
							// Advanced: Time based?
							// Points = 1000 * (1 - (ResponseTime / TimeLimit) / 2)
							// We'd need to store *when* they answered to do time-based.
							// MVP: Just +1000 for correct.
							p.Score += 1000
						}
					}
				}
			}

			h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
				Type:    "state:update",
				Payload: session, // Players map has updated scores
			})
		}
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
			EventID:  eventID,
			ModuleID: moduleID,
			State:    "lobby",
			Players:  make(map[string]*models.Player),
			Answers:  make(map[string]int),
		}
	}
}
