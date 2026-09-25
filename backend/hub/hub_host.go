package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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
	client.send <- jsonMsg

	// Also send full session data via state:update (for player list)
	jsonMsg2, _ := json.Marshal(ServerMessage{
		Type:    "state:update",
		Payload: h.sessions[moduleID],
	})
	client.send <- jsonMsg2
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
	if !client.IsHost || client.ModuleID != moduleID {
		return
	}

	session := h.sessions[moduleID]
	if session != nil {
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

		// Auto-transition to question-active after explainTime seconds
		go func(modID string, idx int, delay int) {
			time.Sleep(time.Duration(delay) * time.Second)
			h.mu.Lock()
			defer h.mu.Unlock()

			sess := h.sessions[modID]
			if sess != nil && sess.CurrentIndex == idx && sess.State == "question-presenting" {
				sess.State = "question-active"
				sess.StartTime = time.Now().Unix()

				// Update the QuestionWindow start time to now
				if idx < len(sess.QuestionWindows) {
					limit := EffectiveTimeLimit(sess.Questions[idx])
					sess.QuestionWindows[idx].StartTime = sess.StartTime
					sess.QuestionWindows[idx].EndTime = sess.StartTime + limit
					sess.QuestionWindows[idx].GraceEndTime = sess.QuestionWindows[idx].EndTime + GracePeriodSeconds
				}

				h.broadcastToSessionLocked(modID, ServerMessage{
					Type:    "cmd:state",
					Payload: map[string]interface{}{"state": "question-active", "currentIndex": idx, "timeLimit": EffectiveTimeLimit(sess.Questions[idx])},
				})
			}
		}(moduleID, session.CurrentIndex, explainTime)
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
		// session.State = "question-active" // Will be set after delay
		session.CurrentIndex++
		session.AnswerCounts = make(map[int]int)
		session.AnsweredPlayerIDs = make(map[string]struct{})
		session.StartTime = time.Now().Unix()

		// Create question window for new question
		if session.CurrentIndex < len(session.Questions) {
			limit := EffectiveTimeLimit(session.Questions[session.CurrentIndex])
			qWindow := models.QuestionWindow{
				QuestionIndex: int64(session.CurrentIndex),
				StartTime:     session.StartTime,
				EndTime:       session.StartTime + limit,
				GraceEndTime:  session.StartTime + limit + GracePeriodSeconds,
				State:         "active",
			}
			session.QuestionWindows = append(session.QuestionWindows, qWindow)
		}

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

		// Auto-transition to question-active after explainTime seconds
		go func(modID string, idx int, delay int) {
			time.Sleep(time.Duration(delay) * time.Second)
			h.mu.Lock()
			defer h.mu.Unlock()

			sess := h.sessions[modID]
			if sess != nil && sess.CurrentIndex == idx && sess.State == "question-presenting" {
				sess.State = "question-active"
				sess.StartTime = time.Now().Unix()

				// Update the QuestionWindow start time to now
				if idx < len(sess.QuestionWindows) {
					limit := EffectiveTimeLimit(sess.Questions[idx])
					sess.QuestionWindows[idx].StartTime = sess.StartTime
					sess.QuestionWindows[idx].EndTime = sess.StartTime + limit
					sess.QuestionWindows[idx].GraceEndTime = sess.QuestionWindows[idx].EndTime + GracePeriodSeconds
				}

				h.broadcastToSessionLocked(modID, ServerMessage{
					Type:    "cmd:state",
					Payload: map[string]interface{}{"state": "question-active", "currentIndex": idx, "timeLimit": EffectiveTimeLimit(sess.Questions[idx])},
				})
			}
		}(client.ModuleID, session.CurrentIndex, explainTime)
	}
}

func (h *QuizHub) processHostShowResults(client *Client, msg ClientMessage) {
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
}

func (h *QuizHub) processHostShowLeaderboard(client *Client, msg ClientMessage) {
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
}

func (h *QuizHub) processHostEndQuiz(client *Client, msg ClientMessage) {
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
