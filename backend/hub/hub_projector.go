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

		// Send data:batch with questions to projector
		if len(h.sessions[moduleID].Batches) > 0 {
			jsonMsg3, _ := json.Marshal(ServerMessage{
				Type:    "data:batch",
				Payload: h.sessions[moduleID].Batches[0],
			})
			if !h.sendToClientLocked(client, jsonMsg3) {
				return
			}
			DebugLog("projector:join: Sent data:batch to projector for module: %s", moduleID)
		}

		DebugLog("projector:join: Sent state:update to projector for module: %s", moduleID)
	} else {
		DebugLog("projector:join: No module ID provided, connected to event %s", eventID)
	}
}
