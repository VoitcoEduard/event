package hub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

// Resolve the event and participant before taking the shared hub lock.
func (h *QuizHub) processPlayerJoinWithPersistence(client *Client, msg ClientMessage) {
	var binding *quizPlayerBinding
	if h.playerIdentity != nil {
		data, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return
		}
		moduleID, _ := data["moduleId"].(string)
		userID, _ := data["userId"].(string)
		participantSecret, _ := data["participantSecret"].(string)
		nickname, _ := data["nickname"].(string)
		if moduleID == "" || userID == "" || len(userID) > 120 || strings.TrimSpace(nickname) == "" || len(nickname) > 120 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		event, err := h.playerIdentity.GetEventByModule(ctx, moduleID)
		if err != nil {
			return
		}
		quizModule := false
		for _, module := range event.Modules {
			if module.ID == moduleID && module.Type == models.ModuleTypeQuiz {
				quizModule = true
				break
			}
		}
		if !quizModule {
			return
		}
		if !event.IsActive {
			h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_closed", "message": "This event is not accepting players."}}))
			return
		}
		hasProof, validProof, proofErr := h.playerIdentity.CheckParticipantProof(ctx, event.ID, userID, participantSecret)
		if proofErr != nil && !errors.Is(proofErr, postgres.ErrNotFound) {
			return
		}
		if !hasProof || !validProof {
			h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "invalid_identity", "message": "Join the event again to verify this player identity."}}))
			return
		}
		binding = &quizPlayerBinding{eventID: event.ID, moduleID: moduleID, playerID: userID}
		h.mu.RLock()
		canJoin := h.canJoinQuizLocked(moduleID, userID)
		if session := h.sessions[moduleID]; canJoin && session != nil && (session.State == "finishing" || session.State == "finished") {
			nickname = session.Players[userID].Nickname
		}
		h.mu.RUnlock()
		if !canJoin {
			h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_closed", "message": "This quiz is not accepting new players."}}))
			return
		}
		now := time.Now().UTC()
		_, err = h.playerIdentity.EnsureParticipant(ctx, &models.Participant{ID: userID, EventID: event.ID, DisplayName: nickname, NormalizedName: strings.ToLower(strings.TrimSpace(nickname)), Status: models.ParticipantStatusJoined, CreatedAt: now, UpdatedAt: now})
		if err != nil {
			h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_failed", "message": "Could not join the event. Please try again."}}))
			return
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	client.ModuleID, client.PlayerID = "", ""
	h.processPlayerJoin(client, msg)
	if binding != nil && client.ModuleID == binding.moduleID && client.PlayerID == binding.playerID {
		if session := h.sessions[binding.moduleID]; session != nil {
			if session.EventID == "" {
				session.EventID = binding.eventID
			}
			if session.EventID == binding.eventID {
				client.quizBinding = binding
			}
		}
	}
}

// Call with h.mu held. Reconnects may read finished results, but must not
// change the roster while results are being saved or after they are final.
func (h *QuizHub) canJoinQuizLocked(moduleID, playerID string) bool {
	session := h.sessions[moduleID]
	if session == nil || (session.State != "finishing" && session.State != "finished") {
		return true
	}
	return session.Players[playerID] != nil
}

func (h *QuizHub) processPlayerJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
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
	// Recheck after persistence in case the host ended the quiz during DB I/O.
	if !h.canJoinQuizLocked(moduleID, playerID) {
		h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_closed", "message": "This quiz is not accepting new players."}}))
		return
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

	// Add a new player, or retain the score and history of a reconnecting one.
	if player, exists := session.Players[playerID]; exists {
		if session.State != "finishing" && session.State != "finished" {
			player.Nickname = nickname
		}
	} else {
		session.Players[playerID] = &models.Player{
			ID:       playerID,
			Nickname: nickname,
			Score:    0,
		}
	}

	// Send state directly to the newly joined player
	statePayload := map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex, "runId": session.RunID}
	if session.State == "question-active" && session.CurrentIndex >= 0 && session.CurrentIndex < len(session.QuestionWindows) && session.CurrentIndex < len(session.Questions) {
		statePayload["timeLimit"] = EffectiveTimeLimit(session.Questions[session.CurrentIndex])
		statePayload["serverNow"] = time.Now().UnixMilli()
		statePayload["deadline"] = windowDeadlineMillis(session.QuestionWindows, session.CurrentIndex)
	}
	if jsonMsg, err := json.Marshal(ServerMessage{
		Type:    "cmd:state",
		Payload: statePayload,
	}); err == nil {
		select {
		case client.send <- jsonMsg:
		default:
		}
	}
	if session.CurrentIndex >= 0 && session.CurrentIndex/MaxQuestionsPerBatch < len(session.Batches) {
		batch := session.Batches[session.CurrentIndex/MaxQuestionsPerBatch]
		select {
		case client.send <- publicBatchMessage(batch, session.CurrentIndex, len(session.Questions)):
		default:
		}
	}
	if (session.State == "results" || session.State == "leaderboard" || session.State == "finished") && session.CurrentIndex >= 0 && session.CurrentIndex < len(session.Questions) {
		select {
		case client.send <- mustMarshal(ServerMessage{Type: "data:reveal", Payload: revealQuestionPayload(session.Questions[session.CurrentIndex], session.CurrentIndex)}):
		default:
		}
	}
	if _, registered := h.clients[client]; registered {
		public := publicSessionMessage(ServerMessage{Type: "state:update", Payload: session}, nil, session, client)
		if len(public) > 0 {
			h.sendToClientLocked(client, public)
		}
	}

	// Notify session update (state:update) to refresh player list for host & projector only
	h.broadcastToHostsAndProjectorsLocked(moduleID, ServerMessage{
		Type:    "state:update",
		Payload: session,
	})
}

func (h *QuizHub) processPlayerSignal(client *Client, msg ClientMessage) {
	if client.quizBinding == nil || client.quizBinding.moduleID != client.ModuleID || client.quizBinding.playerID != client.PlayerID {
		DebugLog("player:signal rejected - client is host")
		return
	}
	// Payload: { "questionIndex": 0, "answerIndex": 0, "submissionId": "..." }
	data, ok := msg.Payload.(map[string]interface{})
	if !ok {
		DebugLog("player:signal rejected - invalid payload type")
		return
	}
	questionIndexFloat, ok := data["questionIndex"].(float64)
	if !ok || questionIndexFloat != float64(int(questionIndexFloat)) {
		DebugLog("player:signal rejected - questionIndex not found or wrong type")
		return
	}

	questionIndex := int(questionIndexFloat)
	submissionID, _ := data["submissionId"].(string)
	DebugLog("player:signal: User=%s, Q=%d", client.PlayerID, questionIndex)

	session := h.sessions[client.ModuleID]
	if session != nil && client.quizBinding.eventID != session.EventID {
		return
	}
	if session != nil && session.State == "finishing" {
		return
	}
	if session != nil {
		ack := func(accepted bool, reason string, awarded int) {
			if client.send == nil || !h.clients[client] {
				return
			}
			total := 0
			if player := session.Players[client.PlayerID]; player != nil {
				total = player.Score
			}
			h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "answer:ack", Payload: map[string]any{
				"submissionId": submissionID, "runId": session.RunID,
				"questionIndex": questionIndex, "accepted": accepted,
				"reason": reason, "awarded": awarded, "total": total,
				"streak": session.Streaks[client.PlayerID],
			}}))
		}
		if session.RunID != "" {
			_, idErr := uuid.Parse(submissionID)
			runID, _ := data["runId"].(string)
			if idErr != nil || runID == "" {
				if client.send != nil && h.clients[client] {
					h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]any{
						"code": "upgrade_required", "message": "This game has been updated. Reload the page to continue.",
					}}))
				}
				return
			}
		}
		if runID, _ := data["runId"].(string); runID != "" && runID != session.RunID {
			ack(false, "wrong_run", 0)
			return
		}
		DebugLog("player:signal - Session found, state: %s, currentIndex: %d, questionWindows: %d",
			session.State, session.CurrentIndex, len(session.QuestionWindows))

		// A2: one answer per player per question. Without this, re-sent
		// (or replayed) signals for the same question keep adding to the
		// score and inflate the answer histogram.
		if client.PlayerID == "" || session.Players[client.PlayerID] == nil {
			log.Printf("Answer rejected (no player) for module %s: Question %d",
				client.ModuleID, questionIndex)
			ack(false, "no_player", 0)
			return
		}
		for _, prev := range session.PlayerSignals {
			if prev.PlayerID == client.PlayerID && prev.QuestionIndex == questionIndex {
				log.Printf("Duplicate answer ignored for module %s: player %s already answered question %d",
					client.ModuleID, client.PlayerID, questionIndex)
				ack(true, "already_accepted", prev.ScoreEarned)
				return
			}
		}

		// A4: only the active question accepts answers. Anything else —
		// future, past, negative, or with no window yet — is dropped before
		// it can score or pollute statistics. This also closes the
		// out-of-range indexing panic on crafted questionIndex values.
		if questionIndex < 0 || questionIndex != session.CurrentIndex ||
			questionIndex >= len(session.QuestionWindows) {
			log.Printf("Answer rejected (invalid question) for module %s: Question %d (current %d, windows %d)",
				client.ModuleID, questionIndex, session.CurrentIndex, len(session.QuestionWindows))
			ack(false, "wrong_question", 0)
			return
		}

		// Check if signal is within acceptance window.
		// A3: honour AcceptanceResult.Scored — grace-period answers count
		// for statistics (Valid) but earn no points (Scored == false).
		qWindow := session.QuestionWindows[questionIndex]
		acceptance := CanAcceptSignal(qWindow, time.Now().Unix())
		status := acceptance.Reason

		if questionIndex >= len(session.Questions) || session.State != "question-active" || !acceptance.Valid {
			ack(false, status, 0)
			return
		}
		aspect := session.WrongPictureAspects[questionIndex]
		if session.Questions[questionIndex].Type != models.QuestionTypeWrongPictures {
			aspect = 1
		}
		awarded, answerIndices, answerValue, guesses, validAnswer := scoreQuizAnswer(
			session.Questions[questionIndex], data, qWindow, time.Now(), session.Streaks[client.PlayerID], aspect,
		)
		if !validAnswer {
			ack(false, "invalid_answer", 0)
			return
		}
		if session.Streaks == nil {
			session.Streaks = make(map[string]int)
		}
		if awarded > 0 && acceptance.Scored {
			session.Streaks[client.PlayerID]++
		} else {
			session.Streaks[client.PlayerID] = 0
		}
		if !acceptance.Scored {
			awarded = 0
		}
		timeSpent := int(time.Since(time.Unix(qWindow.StartTime, 0)).Milliseconds())
		// Create and store signal
		signal := models.PlayerSignal{
			PlayerID:      client.PlayerID,
			QuestionIndex: questionIndex,
			AnswerIndices: answerIndices,
			AnswerValue:   answerValue,
			Guesses:       guesses,
			ScoreEarned:   awarded,
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

		// Update answer counts for statistics (Valid) and score (Scored)
		// as two separate gates.
		if acceptance.Valid {
			for _, ansIdx := range answerIndices {
				session.AnswerCounts[ansIdx]++
			}

			// UPDATE PLAYER SCORE — only when inside the scoring window.
			if acceptance.Scored {
				if player, exists := session.Players[client.PlayerID]; exists {
					player.Score += awarded
				}
			}

			if acceptance.Scored {
				log.Printf("Answer counted for module %s: Question %d, Answers %v. Total answers received/updated.",
					client.ModuleID, questionIndex, answerIndices)
			} else {
				log.Printf("Answer counted without score (grace_period) for module %s: Question %d, Answers %v.",
					client.ModuleID, questionIndex, answerIndices)
			}
		} else {
			log.Printf("Answer rejected (too_late) for module %s: Question %d, Status %s",
				client.ModuleID, questionIndex, status)
		}
		ack(true, status, awarded)

		// Track unique players for the active question instead of scanning the
		// complete answer history on every submission.
		if acceptance.Valid && questionIndex == session.CurrentIndex {
			if session.AnsweredPlayerIDs == nil {
				session.AnsweredPlayerIDs = make(map[string]struct{})
			}
			session.AnsweredPlayerIDs[client.PlayerID] = struct{}{}
		}
		answeredCount := len(session.AnsweredPlayerIDs)

		// Send minimal sync message (answer counts for host histogram and projector)
		h.broadcastToHostsAndProjectorsLocked(session.ModuleID, ServerMessage{
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
}

func (h *QuizHub) processPlayerBatchRequest(client *Client, msg ClientMessage) {
	if client.IsHost {
		return
	}
	// Payload: { "currentBatchIndex": 0 }. The index identifies the exact
	// batch the client needs, rather than the batch after it.
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
	if session != nil && currentBatchIndex >= 0 && currentBatchIndex < len(session.Batches) {
		// Send the requested batch.
		h.sendToClientLocked(client, publicBatchMessage(session.Batches[currentBatchIndex], session.CurrentIndex, len(session.Questions)))
	}
}

func (h *QuizHub) processPlayerReconnect(client *Client, msg ClientMessage) {
	if client.quizBinding == nil || client.quizBinding.moduleID != client.ModuleID || client.quizBinding.playerID != client.PlayerID {
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
	if client.ModuleID != moduleID || client.PlayerID == "" || lastKnownIndex < -1 {
		return
	}

	session := h.sessions[moduleID]
	if session != nil && client.quizBinding.eventID != session.EventID {
		return
	}
	if session == nil {
		h.sendToClientLocked(client, mustMarshal(ServerMessage{
			Type:    "sync:quiz_ended",
			Payload: map[string]interface{}{"reason": "quiz_not_found"},
		}))
		return
	}
	_ = nickname
	player := session.Players[client.PlayerID]
	if player == nil {
		return
	}

	// Calculate what user missed
	currentIndex := session.CurrentIndex
	missedQuestions := []int{}

	for i := lastKnownIndex + 1; i < currentIndex; i++ {
		missedQuestions = append(missedQuestions, i)

		// Do not duplicate an answer or an earlier missed marker.
		seen := false
		for _, previous := range session.PlayerSignals {
			if previous.PlayerID == player.ID && previous.QuestionIndex == i {
				seen = true
				break
			}
		}
		if !seen {
			if session.Streaks == nil {
				session.Streaks = make(map[string]int)
			}
			session.Streaks[player.ID] = 0
			session.PlayerSignals = append(session.PlayerSignals, models.PlayerSignal{
				PlayerID:      player.ID,
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
	h.sendToClientLocked(client, mustMarshal(ServerMessage{
		Type: "sync:catchup",
		Payload: map[string]interface{}{
			"currentState":         session.State,
			"currentQuestionIndex": currentIndex,
			"missedQuestions":      missedQuestions,
			"score":                player.Score,
			"isActive":             isActive,
			"batchesNeeded":        batchesNeeded,
		},
	}))
}
