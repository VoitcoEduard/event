package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
)

func (h *QuizHub) processHostJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
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
	if !h.sendToClientLocked(client, jsonMsg) {
		return
	}

	// Also send full session data via state:update (for player list)
	jsonMsg2, _ := json.Marshal(ServerMessage{
		Type:    "state:update",
		Payload: h.sessions[moduleID],
	})
	if !h.sendToClientLocked(client, jsonMsg2) {
		return
	}
	DebugLog("host:join: Sent state:update to host for module: %s", moduleID)
}

func (h *QuizHub) processHostStartQuiz(client *Client, msg ClientMessage) {
	h.mu.RLock()
	isHost, moduleID := client.IsHost, client.ModuleID
	h.mu.RUnlock()
	if !isHost {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	event, err := h.eventByModule(ctx, moduleID)
	if err != nil || event == nil {
		log.Printf("Error finding event for module %s: %v", moduleID, err)
		return
	}
	questions, err := questionsForModule(event, moduleID)
	if err != nil {
		log.Printf("Cannot start quiz for module %s: %v", moduleID, err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || !client.IsHost || client.ModuleID != moduleID {
		return
	}

	session := h.sessions[moduleID]
	if session != nil && session.State != "finishing" {
		session.RunID = uuid.NewString()
		session.EventID = event.ID // Ensure EventID is set
		session.State = "question-presenting"
		session.CurrentIndex = 0
		session.PlayerSignals = []models.PlayerSignal{}
		session.AnswerCounts = make(map[int]int)
		session.AnsweredPlayerIDs = make(map[string]struct{})
		session.Questions = questions // Assigned loaded questions
		session.Batches = CalculateBatches(questions)
		session.StartTime = time.Now().Unix() // Reset timer
		// session.Players = make(map[string]*models.Player) // REMOVED: Do not reset players, keep lobby players

		// Create question window for first question
		// A8: EffectiveTimeLimit keeps old questions without a timeLimit
		// on the same 20s window every client displays.
		firstLimit := EffectiveTimeLimit(questions[0])
		qWindow := models.QuestionWindow{
			QuestionIndex: int64(session.CurrentIndex),
			StartTime:     session.StartTime,
			EndTime:       session.StartTime + firstLimit,
			GraceEndTime:  session.StartTime + firstLimit + GracePeriodSeconds,
			State:         "active",
		}
		session.QuestionWindows = []models.QuestionWindow{qWindow}

		// Send first batch to clients
		if len(session.Batches) > 0 {
			h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
				Type:    "data:batch",
				Payload: session.Batches[0],
			})
		}

		// Notify state change - send 'question-presenting' for presentation
		explainTime := session.Questions[session.CurrentIndex].ExplainTime
		if explainTime == 0 {
			explainTime = 5 // default
		}
		h.broadcastToSessionLocked(client.ModuleID, ServerMessage{
			Type:    "cmd:state",
			Payload: map[string]interface{}{"state": "question-presenting", "currentIndex": session.CurrentIndex, "timeLimit": explainTime},
		})

		h.scheduleQuizTransitionLocked(session, time.Duration(explainTime)*time.Second)
	}
}

func questionsForModule(event *models.Event, moduleID string) ([]models.QuizQuestion, error) {
	for _, mod := range event.Modules {
		if mod.ID != moduleID {
			continue
		}
		data, err := json.Marshal(mod.Data)
		if err != nil {
			return nil, err
		}
		var quiz models.QuizModuleData
		if err := json.Unmarshal(data, &quiz); err != nil {
			return nil, err
		}
		if len(quiz.Questions) == 0 {
			return nil, fmt.Errorf("no questions")
		}
		return quiz.Questions, nil
	}
	return nil, fmt.Errorf("module not found in event")
}

func (h *QuizHub) processHostNextQuestion(client *Client, msg ClientMessage) {
	if !client.IsHost {
		return
	}
	session := h.sessions[client.ModuleID]
	if session != nil {
		if h.closed || session.State == "finishing" || session.State == "finished" {
			return
		}
		// Advancing past the final question is invalid — the host UI only sends
		// next_question while another question remains (it sends end/leaderboard
		// actions past the last one). A double-click, a racing client, or a
		// session that was never started must be ignored, never index out of range.
		if session.CurrentIndex+1 >= len(session.Questions) {
			DebugLog("host:next_question ignored for module %s: already on question %d of %d",
				client.ModuleID, session.CurrentIndex+1, len(session.Questions))
			return
		}
		// session.State = "question-active" // Will be set after delay
		session.CurrentIndex++
		session.AnswerCounts = make(map[int]int)
		session.AnsweredPlayerIDs = make(map[string]struct{})
		session.StartTime = time.Now().Unix()

		// Create question window for new question
		limit := EffectiveTimeLimit(session.Questions[session.CurrentIndex])
		qWindow := models.QuestionWindow{
			QuestionIndex: int64(session.CurrentIndex),
			StartTime:     session.StartTime,
			EndTime:       session.StartTime + limit,
			GraceEndTime:  session.StartTime + limit + GracePeriodSeconds,
			State:         "active",
		}
		session.QuestionWindows = append(session.QuestionWindows, qWindow)

		// Notify state change - send 'question-presenting' first
		session.State = "question-presenting"
		explainTime := session.Questions[session.CurrentIndex].ExplainTime
		if explainTime == 0 {
			explainTime = 5 // default
		}
		h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
			Type:    "cmd:state",
			Payload: map[string]interface{}{"state": "question-presenting", "currentIndex": session.CurrentIndex, "timeLimit": explainTime},
		})

		h.scheduleQuizTransitionLocked(session, time.Duration(explainTime)*time.Second)
	}
}

func (h *QuizHub) processHostShowResults(client *Client, msg ClientMessage) {
	if !client.IsHost {
		return
	}
	session := h.sessions[client.ModuleID]
	if session != nil {
		h.cancelQuizTransitionLocked(client.ModuleID)
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
}

func (h *QuizHub) processHostShowLeaderboard(client *Client, msg ClientMessage) {
	if !client.IsHost {
		return
	}
	session := h.sessions[client.ModuleID]
	if session != nil {
		h.cancelQuizTransitionLocked(client.ModuleID)
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
}

func (h *QuizHub) processHostEndQuiz(client *Client, msg ClientMessage) {
	h.mu.Lock()
	if !client.IsHost {
		h.mu.Unlock()
		return
	}
	moduleID := client.ModuleID
	session := h.sessions[moduleID]
	if session == nil || session.State == "finishing" || session.State == "finished" {
		h.mu.Unlock()
		return
	}
	if session.RunID == "" {
		session.RunID = uuid.NewString()
	}
	runID := session.RunID
	previous := session.State
	pending := h.quizTimers[moduleID]
	h.cancelQuizTransitionLocked(moduleID)
	session.State = "finishing"
	snapshot := snapshotSession(session)
	h.mu.Unlock()
	var err error
	if h.quizResults == nil {
		err = fmt.Errorf("quiz result store unavailable")
	} else {
		err = h.saveQuizResults(snapshot)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	current := h.sessions[moduleID]
	if current != session || current.RunID != runID {
		return
	}
	if err != nil {
		current.State = previous
		if pending != nil && previous == "question-presenting" {
			h.scheduleQuizTransitionLocked(current, time.Until(pending.deadline))
		}
		log.Printf("Error saving quiz results for module %s: %v", moduleID, err)
		if _, ok := h.clients[client]; ok {
			h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]interface{}{"code": "save_failed", "message": "Could not save results. Retry ending the quiz."}}))
		}
		return
	}
	current.State = "finished"
	h.broadcastToSessionLocked(moduleID, ServerMessage{Type: "cmd:state", Payload: map[string]interface{}{"state": "finished", "currentIndex": current.CurrentIndex}})
	h.broadcastToSessionLocked(moduleID, ServerMessage{Type: "state:update", Payload: current})
}

func (h *QuizHub) processHostResetQuiz(client *Client, msg ClientMessage) {
	if !client.IsHost {
		DebugLog("host:reset_quiz rejected - not a host")
		return
	}
	DebugLog("host:reset_quiz received for module: %s", client.ModuleID)
	h.resetSessionLocked(client.ModuleID)
	DebugLog("host:reset_quiz completed for module: %s", client.ModuleID)
}
