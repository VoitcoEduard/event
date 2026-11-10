package hub

import (
	"encoding/json"
	"github.com/user/go-expo-sample/backend/models"
)

func (h *QuizHub) processProjectorJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
	// Payload: { "moduleId": "...", "eventId": "..." }
	data, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	moduleID, _ := data["moduleId"].(string)
	eventID, _ := data["eventId"].(string)

	DebugLog("projector:join received for module: %s, event: %s", moduleID, eventID)

	client.ModuleID = moduleID
	client.EventID = eventID
	client.IsProjector = true

	if moduleID != "" {
		// Ensure session exists (same logic as host)
		if _, exists := h.sessions[moduleID]; !exists {
			DebugLog("projector:join: Creating new session for module: %s", moduleID)
			h.sessions[moduleID] = &models.QuizSession{
				ModuleID:      moduleID,
				State:         "lobby",
				Players:       make(map[string]*models.Player),
				PlayerSignals: []models.PlayerSignal{},
				AnswerCounts:  make(map[int]int),
			}
		}

		// Send current state to projector via cmd:state (same as host)
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

		// Also send full session data via state:update (for player list)
		jsonMsg2, _ := json.Marshal(ServerMessage{
			Type: "state:update",
			Payload: map[string]any{
				"eventId": h.sessions[moduleID].EventID, "moduleId": moduleID,
				"state": h.sessions[moduleID].State, "currentIndex": h.sessions[moduleID].CurrentIndex,
				"players": h.sessions[moduleID].Players, "answerCounts": h.sessions[moduleID].AnswerCounts,
				"questionWindows": h.sessions[moduleID].QuestionWindows, "startTime": h.sessions[moduleID].StartTime,
			},
		})
		if !h.sendToClientLocked(client, jsonMsg2) {
			return
		}

		// Send data:batch with questions to projector
		if len(h.sessions[moduleID].Batches) > 0 && h.sessions[moduleID].CurrentIndex/MaxQuestionsPerBatch < len(h.sessions[moduleID].Batches) {
			jsonMsg3 := publicBatchMessage(h.sessions[moduleID].Batches[h.sessions[moduleID].CurrentIndex/MaxQuestionsPerBatch], h.sessions[moduleID].CurrentIndex)
			if !h.sendToClientLocked(client, jsonMsg3) {
				return
			}
			DebugLog("projector:join: Sent data:batch to projector for module: %s", moduleID)
		}
		s := h.sessions[moduleID]
		if (s.State == "results" || s.State == "leaderboard" || s.State == "finished") && s.CurrentIndex >= 0 && s.CurrentIndex < len(s.Questions) {
			if !h.sendToClientLocked(client, mustMarshal(ServerMessage{Type: "data:reveal", Payload: revealQuestionPayload(s.Questions[s.CurrentIndex], s.CurrentIndex)})) {
				return
			}
		}

		DebugLog("projector:join: Sent state:update to projector for module: %s", moduleID)
	} else {
		DebugLog("projector:join: No module ID provided, connected to event %s", eventID)
	}
}
