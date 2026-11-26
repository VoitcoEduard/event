package hub

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/database"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/services"
)

const (
	GracePeriodSeconds   = 2
	TargetBatchSize      = 50 * 1024
	MaxQuestionsPerBatch = 50

	DefaultQuestionTimeLimit = 20
)

func EffectiveTimeLimit(q models.QuizQuestion) int64 {
	if q.TimeLimit > 0 {
		return int64(q.TimeLimit)
	}
	return DefaultQuestionTimeLimit
}

type AcceptanceResult struct {
	Valid  bool
	Scored bool
	Reason string
}

func CanAcceptSignal(qw models.QuestionWindow, signalTime int64) AcceptanceResult {
	if signalTime <= qw.EndTime {
		return AcceptanceResult{Valid: true, Scored: true, Reason: "on_time"}
	}
	if signalTime <= qw.GraceEndTime {
		return AcceptanceResult{Valid: true, Scored: false, Reason: "grace_period"}
	}
	return AcceptanceResult{Valid: false, Scored: false, Reason: "too_late"}
}

func CalculateBatches(questions []models.QuizQuestion) []models.QuestionBatch {
	if len(questions) == 0 {
		return []models.QuestionBatch{}
	}

	var batches []models.QuestionBatch
	batchSize := min(MaxQuestionsPerBatch, len(questions))
	totalBatches := (len(questions) + batchSize - 1) / batchSize

	for i := 0; i < totalBatches; i++ {
		start := i * batchSize
		end := min(start+batchSize, len(questions))

		batch := models.QuestionBatch{
			BatchIndex:   i,
			TotalBatches: totalBatches,
			Questions:    questions[start:end],
			NextBatchAt:  end - 2,
			StartIndex:   start,
			EndIndex:     end - 1,
		}
		batches = append(batches, batch)
	}

	return batches
}

type QuizHub struct {
	db              *postgres.Store
	playerIdentity  PlayerIdentityStore
	simpleQuizLocks [256]sync.Mutex
	authService     *auth.Service
	quizResults     *services.QuizResultService
	eventByModule   func(context.Context, string) (*models.Event, error)
	teamsByEvent    func(context.Context, string) ([]models.Team, error)
	saveTeams       func(string, string, string) (*models.TeamResult, error)
	clients         map[*Client]bool
	closed          bool

	sessions   map[string]*models.QuizSession
	quizTimers map[string]*quizTransition

	simpleQuizSessions        map[string]*models.SimpleQuizSession
	simpleQuizUndo            map[string]map[string][]simpleQuizUndoEntry
	simpleQuizTimers          map[string]*time.Timer
	simpleQuizTimerGeneration map[string]uint64

	mu sync.RWMutex
}

type PlayerIdentityStore interface {
	GetEventByModule(context.Context, string) (*models.Event, error)
	CheckParticipantProof(context.Context, string, string, string) (bool, bool, error)
	EnsureParticipant(context.Context, *models.Participant) (*models.Participant, error)
}

type Dependencies struct {
	DB             *postgres.Store
	PlayerIdentity PlayerIdentityStore
	AuthService    *auth.Service
	Events         *services.EventService
	Teams          *database.TeamRepository
	QuizResults    *services.QuizResultService
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
	identity := deps.PlayerIdentity
	if identity == nil && deps.DB != nil {
		identity = deps.DB
	}
	h := &QuizHub{
		db:                        deps.DB,
		playerIdentity:            identity,
		authService:               deps.AuthService,
		quizResults:               deps.QuizResults,
		clients:                   make(map[*Client]bool),
		sessions:                  make(map[string]*models.QuizSession),
		quizTimers:                make(map[string]*quizTransition),
		simpleQuizSessions:        make(map[string]*models.SimpleQuizSession),
		simpleQuizUndo:            make(map[string]map[string][]simpleQuizUndoEntry),
		simpleQuizTimers:          make(map[string]*time.Timer),
		simpleQuizTimerGeneration: make(map[string]uint64),
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
}

func (h *QuizHub) Shutdown() {
	h.mu.Lock()
	h.closed = true
	for moduleID := range h.quizTimers {
		h.cancelQuizTransitionLocked(moduleID)
	}
	for moduleID, timer := range h.simpleQuizTimers {
		if timer != nil {
			timer.Stop()
		}
		delete(h.simpleQuizTimers, moduleID)
	}
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.Unlock()
	for _, client := range clients {
		if client.conn != nil {
			_ = client.conn.Close()
		}
	}
}

func (h *QuizHub) Register(client *Client) bool {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		if client.conn != nil {
			_ = client.conn.Close()
		}
		return false
	}
	h.clients[client] = true
	h.mu.Unlock()
	log.Println("New client registered")
	return true
}

func (h *QuizHub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		log.Println("Client unregistered")

		if client.ModuleID != "" && !client.IsHost {
		}
	}
}

func (h *QuizHub) BroadcastToSession(moduleID string, message interface{}) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling broadcast message:", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
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

func (h *QuizHub) ProcessMessage(client *Client, message []byte) {
	var msg ClientMessage
	defer func() {
		if r := recover(); r != nil {
			log.Printf("recovered panic in hub handler %q: %v\n%s", msg.Type, r, debug.Stack())
		}
	}()
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Println("Error unmarshalling message:", err)
		return
	}
	if strings.HasSuffix(msg.Type, ":join") || strings.HasSuffix(msg.Type, "_join") {
		h.mu.Lock()
		client.quizBinding = nil
		h.mu.Unlock()
	}
	if strings.HasPrefix(msg.Type, "host:") || msg.Type == "teams:admin_join" || msg.Type == "simple:host_join" {
		if !h.AdminActive(client.AdminToken) {
			client.Reject()
			return
		}
	}
	switch msg.Type {
	case "host:start_quiz":
		h.processHostStartQuiz(client, msg)
		return
	case "host:end_teams":
		h.processHostEndTeams(client, msg)
		return
	case "host:end_quiz":
		h.processHostEndQuiz(client, msg)
		return
	case "participant:join":
		h.processParticipantJoin(client, msg)
		return
	case "player:join":
		h.processPlayerJoinWithPersistence(client, msg)
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

	h.mu.Lock()
	defer h.mu.Unlock()
	if session := h.sessions[client.ModuleID]; session != nil && session.State == "finishing" && strings.HasPrefix(msg.Type, "host:") {
		return
	}

	switch msg.Type {
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
	case "host:reset_quiz":
		h.processHostResetQuiz(client, msg)
	case "teams:admin_join":
		h.processTeamsAdminJoin(client, msg)
	}
}

func (h *QuizHub) sendToClientLocked(client *Client, msg []byte) bool {
	if client == nil || client.send == nil {
		return false
	}
	if _, ok := h.clients[client]; !ok {
		return false
	}
	select {
	case client.send <- msg:
		return true
	default:
		close(client.send)
		delete(h.clients, client)
		return false
	}
}

func (h *QuizHub) sendToClient(client *Client, msg []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; !ok {
		return false
	}
	return h.sendToClientLocked(client, msg)
}

func (h *QuizHub) broadcastToSessionLocked(moduleID string, message interface{}) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return
	}

	for client := range h.clients {
		if client.ModuleID == moduleID {
			out := jsonMsg
			if !client.IsHost {
				out = publicSessionMessage(message, jsonMsg, h.sessions[moduleID], client)
			}
			select {
			case client.send <- out:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

func (h *QuizHub) broadcastToHostsAndProjectorsLocked(moduleID string, message interface{}) {
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		return
	}

	for client := range h.clients {
		if client.ModuleID == moduleID && (client.IsHost || client.IsProjector) {
			out := jsonMsg
			if !client.IsHost {
				out = publicSessionMessage(message, jsonMsg, h.sessions[moduleID], client)
			}
			select {
			case client.send <- out:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

func publicSessionMessage(message interface{}, original []byte, session *models.QuizSession, client *Client) []byte {
	m, ok := message.(ServerMessage)
	if !ok {
		return original
	}
	if m.Type == "data:batch" {
		batch, ok := m.Payload.(models.QuestionBatch)
		if !ok || session == nil {
			return nil
		}
		return publicBatchMessage(batch, session.CurrentIndex, len(session.Questions), session.RunID)
	}
	if m.Type != "state:update" {
		return original
	}
	s, ok := m.Payload.(*models.QuizSession)
	if !ok {
		return original
	}
	players := s.Players
	rank := 0
	if !client.IsProjector {
		players = map[string]*models.Player{}
		if self := s.Players[client.PlayerID]; self != nil {
			players[client.PlayerID] = self
			rank = 1
			for id, other := range s.Players {
				if id != client.PlayerID && other != nil && other.Score > self.Score {
					rank++
				}
			}
		}
	}
	b, err := json.Marshal(ServerMessage{Type: m.Type, Payload: map[string]any{
		"eventId": s.EventID, "moduleId": s.ModuleID, "state": s.State,
		"currentIndex": s.CurrentIndex, "totalQuestions": len(s.Questions),
		"players": players, "rank": rank,
		"answerCounts": s.AnswerCounts, "answeredCount": len(s.AnsweredPlayerIDs),
		"questionWindows": s.QuestionWindows, "startTime": s.StartTime,
	}})
	if err != nil {
		return nil
	}
	return b
}

func publicBatchMessage(batch models.QuestionBatch, currentIndex, totalQuestions int, runID string) []byte {
	questions := []map[string]any{}
	if currentIndex >= batch.StartIndex && currentIndex <= batch.EndIndex {
		localIndex := currentIndex - batch.StartIndex
		if localIndex < len(batch.Questions) {
			questions = append(questions, models.PublicQuizQuestion(batch.Questions[localIndex]))
		}
	}
	payload := map[string]any{
		"batchIndex": batch.BatchIndex, "totalBatches": batch.TotalBatches,
		"totalQuestions": totalQuestions,
		"startIndex":     currentIndex, "endIndex": currentIndex,
		"nextBatchAt": currentIndex, "questions": questions,
	}
	if runID != "" {
		payload["runId"] = runID
	}
	return mustMarshal(ServerMessage{Type: "data:batch", Payload: payload})
}

func revealQuestionPayload(q models.QuizQuestion, index int) map[string]any {
	return map[string]any{
		"questionIndex": index, "correctIndex": q.CorrectIndex,
		"correctIndices": q.CorrectIndices, "correctNumber": q.CorrectNumber,
		"correctValue": q.CorrectValue, "wrongPictureConfig": q.WrongPictureConfig,
	}
}

func (h *QuizHub) GetSession(moduleID string) *models.QuizSession {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s := h.sessions[moduleID]
	if s == nil {
		return nil
	}
	return snapshotSession(s)
}

func snapshotSession(s *models.QuizSession) *models.QuizSession {
	copy := *s
	copy.Players = make(map[string]*models.Player, len(s.Players))
	for id, player := range s.Players {
		if player != nil {
			p := *player
			copy.Players[id] = &p
		} else {
			copy.Players[id] = nil
		}
	}
	copy.PlayerSignals = slices.Clone(s.PlayerSignals)
	for i := range copy.PlayerSignals {
		copy.PlayerSignals[i].AnswerIndices = slices.Clone(s.PlayerSignals[i].AnswerIndices)
		copy.PlayerSignals[i].Guesses = slices.Clone(s.PlayerSignals[i].Guesses)
	}
	copy.AnswerCounts = maps.Clone(s.AnswerCounts)
	copy.AnsweredPlayerIDs = maps.Clone(s.AnsweredPlayerIDs)
	copy.QuestionWindows = slices.Clone(s.QuestionWindows)
	copy.Questions = cloneQuestions(s.Questions)
	copy.Batches = slices.Clone(s.Batches)
	for i := range copy.Batches {
		copy.Batches[i].Questions = cloneQuestions(s.Batches[i].Questions)
	}
	return &copy
}

func cloneQuestions(questions []models.QuizQuestion) []models.QuizQuestion {
	cloned := slices.Clone(questions)
	for i := range cloned {
		q := &cloned[i]
		q.Answers = slices.Clone(q.Answers)
		q.CorrectIndices = slices.Clone(q.CorrectIndices)
		q.ImageOptions = slices.Clone(q.ImageOptions)
		if q.ImageID != nil {
			imageID := *q.ImageID
			q.ImageID = &imageID
		}
		if q.WrongPictureConfig != nil {
			config := *q.WrongPictureConfig
			config.Targets = slices.Clone(config.Targets)
			q.WrongPictureConfig = &config
		}
	}
	return cloned
}

func (h *QuizHub) UpdateSession(moduleID string, updater func(*models.QuizSession)) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	session, ok := h.sessions[moduleID]
	if ok && session.State != "finishing" {
		updater(session)
		if timer := h.quizTimers[moduleID]; timer != nil &&
			(session.State != "question-presenting" || timer.runID != session.RunID || timer.index != session.CurrentIndex) {
			h.cancelQuizTransitionLocked(moduleID)
		}
		h.broadcastToSessionLocked(moduleID, ServerMessage{
			Type:    "state:update",
			Payload: session,
		})
		return true
	}
	return false
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
		}
	}
}

func (h *QuizHub) ResetSession(moduleID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s := h.sessions[moduleID]; s != nil && s.State == "finishing" {
		return false
	}
	h.resetSessionLocked(moduleID)
	return true
}

func (h *QuizHub) resetSessionLocked(moduleID string) {
	h.cancelQuizTransitionLocked(moduleID)
	DebugLog("resetSessionLocked: Starting reset for module: %s", moduleID)

	session, exists := h.sessions[moduleID]
	if !exists {
		DebugLog("resetSessionLocked: No session found for module: %s", moduleID)
		return
	}

	DebugLog("resetSessionLocked: Session found, current state: %s, players: %d", session.State, len(session.Players))

	session.RunID = ""
	session.State = "lobby"
	session.CurrentIndex = 0
	session.Players = make(map[string]*models.Player)
	session.PlayerSignals = []models.PlayerSignal{}
	session.AnswerCounts = make(map[int]int)
	session.AnsweredPlayerIDs = make(map[string]struct{})
	session.Streaks = make(map[string]int)
	session.WrongPictureAspects = nil
	session.Questions = nil
	session.Batches = nil
	session.QuestionWindows = nil
	session.StartTime = 0

	h.broadcastToSessionLocked(moduleID, ServerMessage{
		Type:    "cmd:state",
		Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex, "runId": session.RunID, "reset": true},
	})
	h.broadcastToHostsAndProjectorsLocked(moduleID, ServerMessage{
		Type:    "state:update",
		Payload: session,
	})
	for client := range h.clients {
		if (client.ModuleID == moduleID && !client.IsHost && !client.IsProjector && !client.IsTeamsAdmin) ||
			(client.quizBinding != nil && client.quizBinding.moduleID == moduleID) {
			client.ModuleID = ""
			client.PlayerID = ""
			client.quizBinding = nil
		}
	}

	DebugLog("resetSessionLocked: Reset complete for module: %s, broadcast sent", moduleID)
}

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
