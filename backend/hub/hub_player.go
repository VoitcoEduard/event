package hub

import (
	"encoding/json"
	"github.com/user/go-expo-sample/backend/models"
	"log"
	"time"
)

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
		player.Nickname = nickname
	} else {
		session.Players[playerID] = &models.Player{
			ID:       playerID,
			Nickname: nickname,
			Score:    0,
		}
	}

	// Send state directly to the newly joined player
	if jsonMsg, err := json.Marshal(ServerMessage{
		Type:    "cmd:state",
		Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex},
	}); err == nil {
		select {
		case client.send <- jsonMsg:
		default:
		}
	}

	// Notify session update (state:update) to refresh player list for host & projector only
	h.broadcastToHostsAndProjectorsLocked(moduleID, ServerMessage{
		Type:    "state:update",
		Payload: session,
	})
}

func (h *QuizHub) processPlayerSignal(client *Client, msg ClientMessage) {
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

	// Extract AnswerValue for range (optional)
	var answerValue float64
	if val, ok := data["answerValue"].(float64); ok {
		answerValue = val
	}

	// Extract Guesses for wrong_pictures type (optional)
	var guesses []models.PlayerGuess
	if guessesInterface, ok := data["guesses"].([]interface{}); ok {
		guessesBytes, _ := json.Marshal(guessesInterface)
		json.Unmarshal(guessesBytes, &guesses)
	}

	session := h.sessions[client.ModuleID]
	if session != nil {
		DebugLog("player:signal - Session found, state: %s, currentIndex: %d, questionWindows: %d",
			session.State, session.CurrentIndex, len(session.QuestionWindows))

		// A2: one answer per player per question. Without this, re-sent
		// (or replayed) signals for the same question keep adding to the
		// score and inflate the answer histogram.
		if client.PlayerID == "" {
			log.Printf("Answer rejected (no player) for module %s: Question %d",
				client.ModuleID, questionIndex)
			return
		}
		for _, prev := range session.PlayerSignals {
			if prev.PlayerID == client.PlayerID && prev.QuestionIndex == questionIndex {
				log.Printf("Duplicate answer ignored for module %s: player %s already answered question %d",
					client.ModuleID, client.PlayerID, questionIndex)
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
			return
		}

		// Check if signal is within acceptance window.
		// A3: honour AcceptanceResult.Scored — grace-period answers count
		// for statistics (Valid) but earn no points (Scored == false).
		acceptance := AcceptanceResult{Valid: true, Scored: true, Reason: "on_time"}
		status := acceptance.Reason
		if questionIndex < len(session.QuestionWindows) {
			qWindow := session.QuestionWindows[questionIndex]
			acceptance = CanAcceptSignal(qWindow, time.Now().Unix())
			status = acceptance.Reason
			DebugLog("player:signal - QuestionWindow check: index=%d, startTime=%d, endTime=%d, graceEndTime=%d, now=%d, status=%s",
				questionIndex, qWindow.StartTime, qWindow.EndTime, qWindow.GraceEndTime, time.Now().Unix(), status)
		} else {
			DebugLog("player:signal - No QuestionWindow for index %d (windows count: %d)", questionIndex, len(session.QuestionWindows))
		}

		// Store the points actually awarded, not the client claim, so the
		// history and persisted results stay consistent with balances.
		awarded := scoreEarned
		if !acceptance.Scored {
			awarded = 0
		}
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
					player.Score += scoreEarned
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
		client.send <- mustMarshal(ServerMessage{
			Type:    "data:batch",
			Payload: session.Batches[currentBatchIndex],
		})
	}
}

func (h *QuizHub) processPlayerReconnect(client *Client, msg ClientMessage) {
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
	player := session.Players[client.PlayerID]
	if player == nil {
		// Support clients that reconnect without a persistent user ID.
		player = session.Players[nickname]
	}
	if player == nil {
		return
	}

	// Calculate what user missed
	currentIndex := session.CurrentIndex
	missedQuestions := []int{}

	for i := lastKnownIndex + 1; i < currentIndex; i++ {
		missedQuestions = append(missedQuestions, i)

		// Mark as skipped (0 points, no answer)
		if player != nil {
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
	client.send <- mustMarshal(ServerMessage{
		Type: "sync:catchup",
		Payload: map[string]interface{}{
			"currentState":         session.State,
			"currentQuestionIndex": currentIndex,
			"missedQuestions":      missedQuestions,
			"score":                player.Score,
			"isActive":             isActive,
			"batchesNeeded":        batchesNeeded,
		},
	})
}
