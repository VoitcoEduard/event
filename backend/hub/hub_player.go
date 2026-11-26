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
			if errors.Is(err, postgres.ErrNotFound) {
				h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_closed", "message": "This quiz is no longer available."}}))
			} else {
				h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_failed", "message": "Could not join the quiz. Please try again."}}))
			}
			return
		}
		quizModule := false
		for _, module := range event.Modules {
			if module.ID == moduleID && (module.Type == models.ModuleTypeQuiz || module.Type == models.ModuleTypeWrongPictures) {
				quizModule = true
				break
			}
		}
		if !quizModule {
			h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_closed", "message": "This module is not a live quiz."}}))
			return
		}
		if !event.IsActive {
			h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_closed", "message": "This event is not accepting players."}}))
			return
		}
		hasProof, validProof, proofErr := h.playerIdentity.CheckParticipantProof(ctx, event.ID, userID, participantSecret)
		if proofErr != nil && !errors.Is(proofErr, postgres.ErrNotFound) {
			h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_failed", "message": "Could not verify player identity. Please try again."}}))
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

func (h *QuizHub) canJoinQuizLocked(moduleID, playerID string) bool {
	session := h.sessions[moduleID]
	if session == nil || (session.State != "finishing" && session.State != "finished") {
		return true
	}
	return session.Players[playerID] != nil
}

func (h *QuizHub) processPlayerJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
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

	playerID := nickname
	if userID != "" {
		playerID = userID
	}
	if !h.canJoinQuizLocked(moduleID, playerID) {
		h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "join_closed", "message": "This quiz is not accepting new players."}}))
		return
	}

	client.ModuleID = moduleID
	client.IsHost = false
	client.PlayerID = playerID

	session, exists := h.sessions[moduleID]
	if !exists {
		session = &models.QuizSession{
			ModuleID:      moduleID,
			State:         "lobby",
			Players:       make(map[string]*models.Player),
			PlayerSignals: []models.PlayerSignal{},
			AnswerCounts:  make(map[int]int),
		}
		h.sessions[moduleID] = session
	}

	if player, exists := session.Players[playerID]; exists {
		if session.State != "finishing" && session.State != "finished" {
			player.Nickname = nickname
		}
	} else {
		joinedAt := 0
		if session.State != "lobby" && session.CurrentIndex > 0 {
			joinedAt = session.CurrentIndex
		}
		session.Players[playerID] = &models.Player{
			ID:                  playerID,
			Nickname:            nickname,
			Score:               0,
			JoinedQuestionIndex: joinedAt,
		}
	}

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
		if !h.sendToClientLocked(client, jsonMsg) {
			return
		}
	}
	if session.CurrentIndex >= 0 && session.CurrentIndex/MaxQuestionsPerBatch < len(session.Batches) {
		batch := session.Batches[session.CurrentIndex/MaxQuestionsPerBatch]
		if !h.sendToClientLocked(client, publicBatchMessage(batch, session.CurrentIndex, len(session.Questions), session.RunID)) {
			return
		}
	}
	if (session.State == "results" || session.State == "leaderboard" || session.State == "finished") && session.CurrentIndex >= 0 && session.CurrentIndex < len(session.Questions) {
		if !h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "data:reveal", Payload: revealQuestionPayload(session.Questions[session.CurrentIndex], session.CurrentIndex)})) {
			return
		}
	}
	public := publicSessionMessage(ServerMessage{Type: "state:update", Payload: session}, nil, session, client)
	if len(public) > 0 {
		h.sendToClientLocked(client, public)
	}

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
		if session.State == "finishing" {
			ack(false, "quiz_finishing", 0)
			return
		}
		DebugLog("player:signal - Session found, state: %s, currentIndex: %d, questionWindows: %d",
			session.State, session.CurrentIndex, len(session.QuestionWindows))

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

		if questionIndex < 0 || questionIndex != session.CurrentIndex ||
			questionIndex >= len(session.QuestionWindows) {
			log.Printf("Answer rejected (invalid question) for module %s: Question %d (current %d, windows %d)",
				client.ModuleID, questionIndex, session.CurrentIndex, len(session.QuestionWindows))
			ack(false, "wrong_question", 0)
			return
		}

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
		if len(answerIndices) == 1 {
			signal.AnswerIndex = answerIndices[0]
		} else {
			signal.AnswerIndex = -1
		}

		session.PlayerSignals = append(session.PlayerSignals, signal)

		if acceptance.Valid {
			for _, ansIdx := range answerIndices {
				session.AnswerCounts[ansIdx]++
			}

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

		if acceptance.Valid && questionIndex == session.CurrentIndex {
			if session.AnsweredPlayerIDs == nil {
				session.AnsweredPlayerIDs = make(map[string]struct{})
			}
			session.AnsweredPlayerIDs[client.PlayerID] = struct{}{}
		}
		answeredCount := len(session.AnsweredPlayerIDs)

		h.broadcastToHostsAndProjectorsLocked(session.ModuleID, ServerMessage{
			Type: "sync:minimal",
			Payload: map[string]interface{}{
				"questionIndex": questionIndex,
				"answeredCount": answeredCount,
				"totalPlayers":  len(session.Players),
				"answerCounts":  session.AnswerCounts,
			},
		})

		log.Printf("Question %d ended - answeredCount: %d / %d", questionIndex, answeredCount, len(session.Players))
	}
}

func (h *QuizHub) processPlayerBatchRequest(client *Client, msg ClientMessage) {
	if client.IsHost {
		return
	}
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
		h.sendToClientLocked(client, publicBatchMessage(session.Batches[currentBatchIndex], session.CurrentIndex, len(session.Questions), session.RunID))
	}
}

func (h *QuizHub) processPlayerReconnect(client *Client, msg ClientMessage) {
	if client.quizBinding == nil || client.quizBinding.moduleID != client.ModuleID || client.quizBinding.playerID != client.PlayerID {
		return
	}
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

	currentIndex := session.CurrentIndex
	lastAnsweredIndex := -1
	for _, previous := range session.PlayerSignals {
		if previous.PlayerID == player.ID && previous.Status != "missed" && previous.QuestionIndex > lastAnsweredIndex {
			lastAnsweredIndex = previous.QuestionIndex
		}
	}
	startIdx := lastKnownIndex + 1
	if player.JoinedQuestionIndex > startIdx {
		startIdx = player.JoinedQuestionIndex
	}
	if lastAnsweredIndex+1 > startIdx {
		startIdx = lastAnsweredIndex + 1
	}

	missedQuestions := []int{}

	for i := startIdx; i < currentIndex; i++ {
		seen := false
		for _, previous := range session.PlayerSignals {
			if previous.PlayerID == player.ID && previous.QuestionIndex == i {
				seen = true
				break
			}
		}
		if !seen {
			missedQuestions = append(missedQuestions, i)
			if session.Streaks == nil {
				session.Streaks = make(map[string]int)
			}
			session.Streaks[player.ID] = 0
			session.PlayerSignals = append(session.PlayerSignals, models.PlayerSignal{
				PlayerID:      player.ID,
				QuestionIndex: i,
				AnswerIndex:   -1,
				ScoreEarned:   0,
				TimeSpent:     0,
				Timestamp:     time.Now().Unix(),
				Status:        "missed",
			})
		}
	}

	batchesNeeded := []int{}
	for batchIdx, batch := range session.Batches {
		batchStart := batch.StartIndex
		batchEnd := batch.EndIndex

		if (batchStart <= currentIndex && batchEnd >= currentIndex) ||
			(batchStart > currentIndex) {
			batchesNeeded = append(batchesNeeded, batchIdx)
		}
	}

	isActive := len(missedQuestions) <= 5

	h.sendToClientLocked(client, mustMarshal(ServerMessage{
		Type: "sync:catchup",
		Payload: map[string]interface{}{
			"currentState":         session.State,
			"currentQuestionIndex": currentIndex,
			"missedQuestions":      missedQuestions,
			"score":                player.Score,
			"streak":               session.Streaks[player.ID],
			"isActive":             isActive,
			"batchesNeeded":        batchesNeeded,
		},
	}))
}
