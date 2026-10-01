package handlers

import (
	"encoding/json"
	"github.com/user/go-expo-sample/backend/models"
	"strings"
	"testing"
)

func TestPublicTeamsProjectionDoesNotLeakDeviceIdentity(t *testing.T) {
	a := &models.TeamsActivity{EventID: "e", ModuleID: "m", Teams: []models.ActivityTeam{{ID: "blue", Name: "Blue"}}, Entrants: []models.TeamsEntrant{{ParticipantID: "secret-device", Name: "Anna", TeamID: "blue"}}, Awards: []models.TeamAward{{ID: "award", RequestID: "private-request", ActorID: "private-admin", TeamID: "blue", Points: 20, Recipients: []string{"secret-device"}}}}
	data, err := json.Marshal(publicTeamsState(a, "secret-device", 35))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"secret-device", "private-request", "private-admin", "recipients", "actorId"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("public state leaked %s", secret)
		}
	}
	if !strings.Contains(string(data), `"myPoints":20`) || !strings.Contains(string(data), `"forYou":true`) {
		t.Fatal("personal award projection missing")
	}
	anonymous, _ := json.Marshal(publicTeamsState(a, "", 0))
	if strings.Contains(string(anonymous), `"forYou":true`) {
		t.Fatal("projector claimed personal awards")
	}
}
