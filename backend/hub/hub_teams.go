package hub

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/user/go-expo-sample/backend/models"
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
	h.sendToClientLocked(client, jsonMsg)
}

func (h *QuizHub) processHostEndTeams(client *Client, msg ClientMessage) {
	h.mu.RLock()
	isHost := client.IsHost
	h.mu.RUnlock()
	if !isHost {
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

	result, err := h.saveTeams(eventID, moduleID, moduleName)

	if err != nil {
		log.Printf("Error saving team results for module %s: %v", moduleID, err)
	}

	message := ServerMessage{
		Type:    "teams:ended",
		Payload: map[string]interface{}{"eventId": eventID, "moduleId": moduleID, "result": result},
	}
	h.BroadcastToTeamsAdmins(eventID, message)
}

func (h *QuizHub) processParticipantJoin(client *Client, msg ClientMessage) {
	data, ok := msg.Payload.(map[string]interface{})
	if !ok {
		h.mu.Lock()
		defer h.mu.Unlock()
		client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
		return
	}
	eventID, _ := data["eventId"].(string)
	userID, _ := data["userId"].(string)
	displayName, _ := data["displayName"].(string)

	DebugLog("participant:join received for event: %s, user: %s", eventID, userID)

	var teamsAvailable []models.Team
	if eventID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		teams, err := h.teamsByEvent(ctx, eventID)
		if err == nil && len(teams) > 0 {
			teamsAvailable = teams
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	client.IsHost, client.IsTeamsAdmin, client.IsProjector = false, false, false
	client.EventID = eventID
	client.PlayerID = userID
	if len(teamsAvailable) > 0 {
		jsonMsg, _ := json.Marshal(ServerMessage{
			Type:    "teams:available",
			Payload: map[string]interface{}{"eventId": eventID, "teams": teamsAvailable},
		})
		// The admin broadcast below skips evicted clients, so the join is
		// still announced even when this client is too stalled to inform.
		h.sendToClientLocked(client, jsonMsg)
	}

	if userID != "" && displayName != "" {
		message := ServerMessage{
			Type:    "participant:joined",
			Payload: map[string]interface{}{"userId": userID, "displayName": displayName},
		}
		h.broadcastToTeamsAdminsLocked(eventID, message)
	}
}
