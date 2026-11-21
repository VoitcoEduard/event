package hub

import (
	"encoding/json"
)

func (h *QuizHub) processProjectorJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector, client.PlayerID = false, false, false, ""
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
		s, exists := h.sessions[moduleID]
		if !exists || s == nil {
			// Do not allocate phantom sessions for unauthenticated projector joins.
			// Send an initial lobby cmd:state so the projector renders lobby while
			// remaining subscribed for updates once a host creates/starts a session.
			jsonMsg, _ := json.Marshal(ServerMessage{
				Type: "cmd:state",
				Payload: map[string]interface{}{
					"state":        "lobby",
					"currentIndex": 0,
					"runId":        "",
				},
			})
			h.sendToClientLocked(client, jsonMsg)
			DebugLog("projector:join: No active session for module: %s; sent initial lobby state", moduleID)
			return
		}

		// Send current state to projector via cmd:state (same as host)
		jsonMsg, _ := json.Marshal(ServerMessage{
			Type: "cmd:state",
			Payload: map[string]interface{}{
				"state":        s.State,
				"currentIndex": s.CurrentIndex,
				"runId":        s.RunID,
			},
		})
		if !h.sendToClientLocked(client, jsonMsg) {
			return
		}

		// Also send full session data via state:update (for player list)
		jsonMsg2, _ := json.Marshal(ServerMessage{
			Type: "state:update",
			Payload: map[string]any{
				"eventId": s.EventID, "moduleId": moduleID,
				"state": s.State, "currentIndex": s.CurrentIndex,
				"players": s.Players, "answerCounts": s.AnswerCounts,
				"questionWindows": s.QuestionWindows, "startTime": s.StartTime,
			},
		})
		if !h.sendToClientLocked(client, jsonMsg2) {
			return
		}

		// Send data:batch with questions to projector
		if len(s.Batches) > 0 && s.CurrentIndex/MaxQuestionsPerBatch < len(s.Batches) {
			jsonMsg3 := publicBatchMessage(s.Batches[s.CurrentIndex/MaxQuestionsPerBatch], s.CurrentIndex)
			if !h.sendToClientLocked(client, jsonMsg3) {
				return
			}
			DebugLog("projector:join: Sent data:batch to projector for module: %s", moduleID)
		}
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
