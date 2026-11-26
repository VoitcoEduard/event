package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
)

var (
	ErrQuizNotActive       = errors.New("quiz session not active")
	ErrQuizEventMismatch   = errors.New("quiz session belongs to another event")
	ErrQuizFinishing       = errors.New("quiz results are being saved")
	ErrQuizFinished        = errors.New("quiz session already finished")
	ErrQuizNoMoreQuestions = errors.New("no more questions")
	ErrInvalidImageMeta    = errors.New("invalid image metadata")
)

func (h *QuizHub) processHostJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
	data, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	moduleID, _ := data["moduleId"].(string)

	DebugLog("host:join received for module: %s", moduleID)

	client.ModuleID = moduleID
	client.IsHost = true

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

	jsonMsg, _ := json.Marshal(ServerMessage{
		Type: "cmd:state",
		Payload: map[string]interface{}{
			"state":        h.sessions[moduleID].State,
			"currentIndex": h.sessions[moduleID].CurrentIndex,
			"runId":        h.sessions[moduleID].RunID,
		},
	})
	if !h.sendToClientLocked(client, jsonMsg) {
		return
	}

	jsonMsg2, _ := json.Marshal(ServerMessage{
		Type:    "state:update",
		Payload: h.sessions[moduleID],
	})
	if !h.sendToClientLocked(client, jsonMsg2) {
		return
	}
	DebugLog("host:join: Sent state:update to host for module: %s", moduleID)
}

func (h *QuizHub) WrongPictureAspects(ctx context.Context, questions []models.QuizQuestion) (map[int]float64, error) {
	aspects := make(map[int]float64)
	for i, question := range questions {
		if question.Type != models.QuestionTypeWrongPictures {
			continue
		}
		if h.db == nil {
			aspects[i] = 1
			continue
		}
		if question.ImageID == nil || *question.ImageID == "" {
			return nil, fmt.Errorf("%w: question %d missing image", ErrInvalidImageMeta, i)
		}
		media, mediaErr := h.db.Media(ctx, *question.ImageID)
		if mediaErr != nil || media.Width <= 0 || media.Height <= 0 {
			return nil, fmt.Errorf("%w: question %d missing dimensions: %v", ErrInvalidImageMeta, i, mediaErr)
		}
		aspects[i] = float64(media.Width) / float64(media.Height)
	}
	return aspects, nil
}

func (h *QuizHub) startQuizSessionLocked(session *models.QuizSession, moduleID, eventID string, questions []models.QuizQuestion, aspects map[int]float64) error {
	if session == nil {
		return ErrQuizNotActive
	}
	if session.EventID != "" && eventID != "" && session.EventID != eventID {
		return ErrQuizEventMismatch
	}
	if session.State == "finishing" {
		return ErrQuizFinishing
	}
	if len(questions) == 0 {
		return ErrQuizNoMoreQuestions
	}

	session.RunID = uuid.NewString()
	session.EventID = eventID
	session.State = "question-presenting"
	session.CurrentIndex = 0
	session.PlayerSignals = []models.PlayerSignal{}
	session.AnswerCounts = make(map[int]int)
	session.AnsweredPlayerIDs = make(map[string]struct{})
	session.Streaks = make(map[string]int)
	for _, player := range session.Players {
		if player != nil {
			player.Score = 0
			player.JoinedQuestionIndex = 0
		}
	}
	session.WrongPictureAspects = aspects
	session.Questions = questions
	session.Batches = CalculateBatches(questions)
	session.StartTime = time.Now().Unix()

	firstLimit := EffectiveTimeLimit(questions[0])
	qWindow := models.QuestionWindow{
		QuestionIndex: int64(session.CurrentIndex),
		StartTime:     session.StartTime,
		EndTime:       session.StartTime + firstLimit,
		GraceEndTime:  session.StartTime + firstLimit + GracePeriodSeconds,
		State:         "active",
	}
	session.QuestionWindows = []models.QuestionWindow{qWindow}

	if len(session.Batches) > 0 {
		h.broadcastToSessionLocked(moduleID, ServerMessage{
			Type:    "data:batch",
			Payload: session.Batches[0],
		})
	}

	explainTime := session.Questions[session.CurrentIndex].ExplainTime
	if explainTime == 0 {
		explainTime = 5
	}
	h.broadcastToSessionLocked(moduleID, ServerMessage{
		Type:    "cmd:state",
		Payload: map[string]interface{}{"state": "question-presenting", "currentIndex": session.CurrentIndex, "totalQuestions": len(session.Questions), "timeLimit": explainTime, "runId": session.RunID},
	})
	h.broadcastToSessionLocked(moduleID, ServerMessage{
		Type:    "state:update",
		Payload: session,
	})

	h.scheduleQuizTransitionLocked(session, time.Duration(explainTime)*time.Second)
	return nil
}

func (h *QuizHub) nextQuizQuestionLocked(session *models.QuizSession, eventID string) error {
	if session == nil {
		return ErrQuizNotActive
	}
	if session.EventID != "" && eventID != "" && session.EventID != eventID {
		return ErrQuizEventMismatch
	}
	if h.closed || session.State == "finishing" {
		return ErrQuizFinishing
	}
	if session.State == "finished" {
		return ErrQuizFinished
	}
	if session.CurrentIndex+1 >= len(session.Questions) {
		return ErrQuizNoMoreQuestions
	}
	if session.Streaks == nil {
		session.Streaks = make(map[string]int)
	}
	for playerID := range session.Players {
		if _, answered := session.AnsweredPlayerIDs[playerID]; !answered {
			session.Streaks[playerID] = 0
		}
	}
	session.CurrentIndex++
	session.AnswerCounts = make(map[int]int)
	session.AnsweredPlayerIDs = make(map[string]struct{})
	session.StartTime = time.Now().Unix()

	limit := EffectiveTimeLimit(session.Questions[session.CurrentIndex])
	qWindow := models.QuestionWindow{
		QuestionIndex: int64(session.CurrentIndex),
		StartTime:     session.StartTime,
		EndTime:       session.StartTime + limit,
		GraceEndTime:  session.StartTime + limit + GracePeriodSeconds,
		State:         "active",
	}
	session.QuestionWindows = append(session.QuestionWindows, qWindow)
	batchIndex := session.CurrentIndex / MaxQuestionsPerBatch
	if batchIndex < len(session.Batches) {
		h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
			Type: "data:batch", Payload: session.Batches[batchIndex],
		})
	}

	session.State = "question-presenting"
	explainTime := session.Questions[session.CurrentIndex].ExplainTime
	if explainTime == 0 {
		explainTime = 5
	}
	h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
		Type:    "cmd:state",
		Payload: map[string]interface{}{"state": "question-presenting", "currentIndex": session.CurrentIndex, "totalQuestions": len(session.Questions), "timeLimit": explainTime, "runId": session.RunID},
	})

	h.scheduleQuizTransitionLocked(session, time.Duration(explainTime)*time.Second)
	return nil
}

func (h *QuizHub) StartQuizSession(moduleID, eventID string, questions []models.QuizQuestion, aspects map[int]float64) (*models.QuizSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, ErrQuizFinishing
	}
	session, exists := h.sessions[moduleID]
	if !exists || session == nil {
		session = &models.QuizSession{
			EventID:       eventID,
			ModuleID:      moduleID,
			State:         "lobby",
			Players:       make(map[string]*models.Player),
			PlayerSignals: []models.PlayerSignal{},
			AnswerCounts:  make(map[int]int),
		}
		h.sessions[moduleID] = session
	}
	if err := h.startQuizSessionLocked(session, moduleID, eventID, questions, aspects); err != nil {
		return nil, err
	}
	return snapshotSession(session), nil
}

func (h *QuizHub) NextQuizQuestion(moduleID, eventID string) (*models.QuizSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session := h.sessions[moduleID]
	if err := h.nextQuizQuestionLocked(session, eventID); err != nil {
		return nil, err
	}
	return snapshotSession(session), nil
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
		h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]any{"code": "start_failed", "message": "Could not load quiz event. Please try again."}}))
		return
	}
	questions, err := questionsForModule(event, moduleID)
	if err != nil {
		log.Printf("Cannot start quiz for module %s: %v", moduleID, err)
		h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]any{"code": "start_failed", "message": "Cannot start quiz: " + err.Error()}}))
		return
	}
	aspects, err := h.WrongPictureAspects(ctx, questions)
	if err != nil {
		log.Printf("Cannot start Wrong Pictures for module %s: %v", moduleID, err)
		h.sendToClient(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]any{"code": "invalid_image_metadata", "message": "This image has no dimensions. Reupload it before starting the quiz."}}))
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || !client.IsHost || client.ModuleID != moduleID {
		return
	}

	if err := h.startQuizSessionLocked(h.sessions[moduleID], client.ModuleID, event.ID, questions, aspects); err != nil {
		if errors.Is(err, ErrQuizFinishing) {
			h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]any{
				"code": "quiz_finishing", "message": "Quiz results are being saved. Wait a moment and try again.",
			}}))
		} else {
			h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "cmd:error", Payload: map[string]any{
				"code": "start_failed", "message": err.Error(),
			}}))
		}
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
	if err := h.nextQuizQuestionLocked(h.sessions[client.ModuleID], ""); err != nil {
		DebugLog("host:next_question ignored for module %s: %v", client.ModuleID, err)
	}
}

func (h *QuizHub) processHostShowResults(client *Client, msg ClientMessage) {
	if !client.IsHost {
		return
	}
	session := h.sessions[client.ModuleID]
	if session != nil {
		if session.State == "finishing" || session.State == "finished" {
			return
		}
		h.cancelQuizTransitionLocked(client.ModuleID)
		session.State = "results"
		if session.CurrentIndex >= 0 && session.CurrentIndex < len(session.Questions) {
			q := session.Questions[session.CurrentIndex]
			h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
				Type: "data:reveal", Payload: revealQuestionPayload(q, session.CurrentIndex),
			})
		}

		h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
			Type:    "cmd:state",
			Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex, "totalQuestions": len(session.Questions), "runId": session.RunID},
		})

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
		if session.State == "finishing" || session.State == "finished" {
			return
		}
		h.cancelQuizTransitionLocked(client.ModuleID)
		session.State = "leaderboard"
		h.broadcastToSessionLocked(session.ModuleID, ServerMessage{
			Type:    "cmd:state",
			Payload: map[string]interface{}{"state": session.State, "currentIndex": session.CurrentIndex, "totalQuestions": len(session.Questions), "runId": session.RunID},
		})

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
	h.broadcastToSessionLocked(moduleID, ServerMessage{Type: "cmd:state", Payload: map[string]interface{}{"state": "finished", "currentIndex": current.CurrentIndex, "runId": current.RunID}})
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
