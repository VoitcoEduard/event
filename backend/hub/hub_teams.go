package hub

import (
	"context"
	"encoding/json"
	"log"
)

func (h *QuizHub) processTeamsAdminJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
	data, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	eventID, _ := data["eventId"].(string)

	DebugLog("teams:admin_join received for event: %s", eventID)

	client.EventID = eventID
	client.IsTeamsAdmin = true

	jsonMsg, _ := json.Marshal(ServerMessage{
		Type:    "teams:admin_joined",
		Payload: map[string]interface{}{"eventId": eventID, "status": "connected"},
	})
	client.send <- jsonMsg
}

func (h *QuizHub) processHostEndTeams(client *Client, msg ClientMessage) {
	if !client.IsHost {
		return
	}
	data, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	eventID, _ := data["eventId"].(string)
	moduleID, _ := data["moduleId"].(string)
	moduleName, _ := data["moduleName"].(string)

	DebugLog("host:end_teams received for event: %s, module: %s", eventID, moduleID)

	h.mu.Unlock()
	result, err := h.saveTeamResults(eventID, moduleID, moduleName)
	h.mu.Lock()

	if err != nil {
		log.Printf("Error saving team results for module %s: %v", moduleID, err)
	}

	jsonMsg, _ := json.Marshal(ServerMessage{
		Type:    "teams:ended",
		Payload: map[string]interface{}{"eventId": eventID, "moduleId": moduleID, "result": result},
	})
	h.BroadcastToTeamsAdmins(eventID, jsonMsg)
}

func (h *QuizHub) processParticipantJoin(client *Client, msg ClientMessage) {
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
	data, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}
	eventID, _ := data["eventId"].(string)
	userID, _ := data["userId"].(string)
	displayName, _ := data["displayName"].(string)

	DebugLog("participant:join received for event: %s, user: %s", eventID, userID)

	client.EventID = eventID
	client.PlayerID = userID

	if eventID != "" {
		h.mu.Unlock()
		teams, err := getTeamRepo().GetByEventID(context.TODO(), eventID)
		h.mu.Lock()

		if err == nil && len(teams) > 0 {
			jsonMsg, _ := json.Marshal(ServerMessage{
				Type:    "teams:available",
				Payload: map[string]interface{}{"eventId": eventID, "teams": teams},
			})
			client.send <- jsonMsg
		}
	}

	if userID != "" && displayName != "" {
		jsonMsg, _ := json.Marshal(ServerMessage{
			Type:    "participant:joined",
			Payload: map[string]interface{}{"userId": userID, "displayName": displayName},
		})
		h.BroadcastToTeamsAdmins(eventID, jsonMsg)
	}
}
