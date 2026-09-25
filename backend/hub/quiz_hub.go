package hub

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/services"
)

const (
	GracePeriodSeconds   = 2
	TargetBatchSize      = 50 * 1024 // 50KB
	MaxQuestionsPerBatch = 50        // Conservative limit

	// A8: defaults matching every client (`question.timeLimit || 20`).
	// A missing TimeLimit (zero value in old templates/JSON) must not
	// produce a 0-second acceptance window on the server.
	DefaultQuestionTimeLimit = 20
)

// EffectiveTimeLimit returns the question's time limit in seconds,
// falling back to DefaultQuestionTimeLimit when unset or invalid.
func EffectiveTimeLimit(q models.QuizQuestion) int64 {
	if q.TimeLimit > 0 {
		return int64(q.TimeLimit)
	}
	return DefaultQuestionTimeLimit
}

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
	authService   *auth.Service
	quizResults   *services.QuizResultService
	eventByModule func(context.Context, string) (*models.Event, error)
	teamsByEvent  func(context.Context, string) ([]models.Team, error)
	saveTeams     func(string, string, string) (*models.TeamResult, error)
	// Registered clients.
	clients map[*Client]bool

	// Active quiz sessions
	sessions map[string]*models.QuizSession

	// Active Simple Quiz sessions (moduleID -> session)
	simpleQuizSessions map[string]*models.SimpleQuizSession
	// Per-judge undo stacks (moduleID -> judgeKey -> stack of undo entries)
	simpleQuizUndo map[string]map[string][]simpleQuizUndoEntry
	// Active auto-lock timers (moduleID -> timer)
	simpleQuizTimers map[string]*time.Timer

	// Mutex for thread-safe access
	mu sync.RWMutex
}

type Dependencies struct {
	AuthService *auth.Service
	Events      *services.EventService
	Teams       *database.TeamRepository
	QuizResults *services.QuizResultService
}

func (h *QuizHub) AdminActive(token string) bool {
	if h.authService == nil || token == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, user, err := h.authService.Store.Resolve(ctx, token)
	return err == nil && user.Status == "active" && (user.Role == "admin" || user.Role == "owner")
}

func NewQuizHub(deps Dependencies) *QuizHub {
	h := &QuizHub{
		authService:        deps.AuthService,
		quizResults:        deps.QuizResults,
		clients:            make(map[*Client]bool),
		sessions:           make(map[string]*models.QuizSession),
		simpleQuizSessions: make(map[string]*models.SimpleQuizSession),
		simpleQuizUndo:     make(map[string]map[string][]simpleQuizUndoEntry),
		simpleQuizTimers:   make(map[string]*time.Timer),
	}
	if deps.Events != nil {
		h.eventByModule = deps.Events.GetByModule
	}
	if deps.Teams != nil {
		h.teamsByEvent = deps.Teams.GetByEventID
	}
	h.saveTeams = h.saveTeamResults
	return h
}

func (h *QuizHub) Run() {
	// No-op for now, or use for cleanup tasks
	// We switched to Mutex based approach for easier state management
}

// Shutdown closes upgraded connections, which http.Server.Shutdown does not own.
func (h *QuizHub) Shutdown() {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		_ = client.conn.Close()
	}
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
	h.mu.Lock()
	defer h.mu.Unlock()

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
	// Resolve against MongoDB before taking the hub lock. A client-provided role
	// or an earlier host join never authorizes a later privileged message.
	if strings.HasPrefix(msg.Type, "host:") || msg.Type == "teams:admin_join" {
		if !h.AdminActive(client.AdminToken) {
			client.Reject()
			return
		}
	}
	// These handlers perform database I/O and acquire the hub lock only when
	// reading or changing in-memory state.
	switch msg.Type {
	case "host:start_quiz":
		h.processHostStartQuiz(client, msg)
		return
	case "host:end_teams":
		h.processHostEndTeams(client, msg)
		return
	case "participant:join":
		h.processParticipantJoin(client, msg)
		return
	case "simple:host_join":
		h.processSimpleHostJoin(client, msg)
		return
	case "simple:player_join":
		h.processSimplePlayerJoin(client, msg)
		return
	case "simple:projector_join":
		h.processSimpleProjectorJoin(client, msg)
		return
	case "simple:judge_join":
		h.processSimpleJudgeJoin(client, msg)
		return
	}

	// Lock for state updates
	h.mu.Lock()
	defer h.mu.Unlock()

	switch msg.Type {
	case "player:join":
		h.processPlayerJoin(client, msg)
	case "host:join":
		h.processHostJoin(client, msg)
	case "projector:join":
		h.processProjectorJoin(client, msg)
	case "player:signal":
		h.processPlayerSignal(client, msg)
	case "player:batch_request":
		h.processPlayerBatchRequest(client, msg)
	case "player:reconnect":
		h.processPlayerReconnect(client, msg)
	case "host:next_question":
		h.processHostNextQuestion(client, msg)
	case "host:show_results":
		h.processHostShowResults(client, msg)
	case "host:show_leaderboard":
		h.processHostShowLeaderboard(client, msg)
	case "host:end_quiz":
		h.processHostEndQuiz(client, msg)
	case "host:reset_quiz":
		h.processHostResetQuiz(client, msg)
	case "teams:admin_join":
		h.processTeamsAdminJoin(client, msg)
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

// Internal helper that broadcasts only to Host and Projector clients (assumes Lock is held)
func (h *QuizHub) broadcastToHostsAndProjectorsLocked(moduleID string, message interface{}) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return
	}

	for client := range h.clients {
		if client.ModuleID == moduleID && (client.IsHost || client.IsProjector) {
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
	session.AnsweredPlayerIDs = make(map[string]struct{})
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

// saveQuizResults persists the current session state through the quiz service.
func (h *QuizHub) saveQuizResults(session *models.QuizSession) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return h.quizResults.SaveQuiz(ctx, session)
}

func (h *QuizHub) saveTeamResults(eventID, moduleID, moduleName string) (*models.TeamResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return h.quizResults.SaveTeams(ctx, eventID, moduleID, moduleName)
}

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return nil
	}
	return b
}

// BroadcastScreenState broadcasts the screen state update to Big Screens connected to a specific event
func (h *QuizHub) BroadcastScreenState(eventID string, state interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	jsonMsg, err := json.Marshal(ServerMessage{
		Type:    "screen:state_update",
		Payload: state,
	})
	if err != nil {
		log.Println("Error marshalling broadcast message:", err)
		return
	}

	for client := range h.clients {
		if client.IsProjector && client.EventID == eventID {
			select {
			case client.send <- jsonMsg:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastToEvent broadcasts a message to all clients connected to a specific event
func (h *QuizHub) BroadcastToEvent(eventID string, message interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	jsonMsg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling broadcast message:", err)
		return
	}

	for client := range h.clients {
		if client.EventID == eventID {
			select {
			case client.send <- jsonMsg:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// BroadcastToTeamsAdmins broadcasts to all admin clients for a specific event
func (h *QuizHub) BroadcastToTeamsAdmins(eventID string, message interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcastToTeamsAdminsLocked(eventID, message)
}

func (h *QuizHub) broadcastToTeamsAdminsLocked(eventID string, message interface{}) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling broadcast message:", err)
		return
	}

	for client := range h.clients {
		if client.EventID == eventID && client.IsTeamsAdmin {
			select {
			case client.send <- jsonMsg:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}
