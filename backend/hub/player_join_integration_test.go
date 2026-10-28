//go:build integration

package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
	"github.com/user/go-expo-sample/backend/testutils"
)

func TestPlayerJoinValidatesEventAndQuizState(t *testing.T) {
	db := testutils.SetupPostgres(t)
	ctx := context.Background()
	for i, tc := range []struct {
		name, state                 string
		active, reconnect, accepted bool
	}{
		{"inactive", "lobby", false, false, false},
		{"inactive reconnect", "finished", false, true, false},
		{"lobby", "lobby", true, false, true},
		{"playing", "question", true, false, true},
		{"finishing", "finishing", true, false, false},
		{"finished", "finished", true, false, false},
		{"finishing reconnect", "finishing", true, true, true},
		{"finished reconnect", "finished", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			eventID, moduleID := uuid.NewString(), uuid.NewString()
			if err := db.CreateEvent(ctx, &models.Event{ID: eventID, Pin: fmt.Sprintf("%06d", i+1), IsActive: tc.active, CreatedAt: time.Now().UTC(), Modules: []models.Module{{ID: moduleID, Type: models.ModuleTypeQuiz}}}); err != nil {
				t.Fatal(err)
			}
			h := NewQuizHub(Dependencies{DB: db})
			h.CreateSession(moduleID, eventID)
			h.UpdateSession(moduleID, func(s *models.QuizSession) {
				if tc.reconnect {
					s.Players["player"] = &models.Player{ID: "player", Nickname: "Original", Score: 42}
				}
				s.State = tc.state
			})
			client := &Client{send: make(chan []byte, 16)}
			h.Register(client)
			h.processPlayerJoinWithPersistence(client, ClientMessage{Type: "player:join", Payload: map[string]interface{}{"moduleId": moduleID, "userId": "player", "nickname": "Changed"}})
			p, err := db.Participant(ctx, eventID, "player")
			if tc.accepted {
				wantName := "Changed"
				if tc.reconnect {
					wantName = "Original"
				}
				if err != nil || p.DisplayName != wantName || client.ModuleID != moduleID {
					t.Fatalf("accepted join: participant=%+v error=%v module=%q", p, err, client.ModuleID)
				}
			} else if !errors.Is(err, postgres.ErrNotFound) || client.ModuleID != "" {
				t.Fatalf("rejected join persisted/associated: participant=%+v error=%v module=%q", p, err, client.ModuleID)
			}
			if !tc.accepted {
				select {
				case raw := <-client.send:
					var message struct {
						Type    string `json:"type"`
						Payload struct {
							Code string `json:"code"`
						} `json:"payload"`
					}
					if err := json.Unmarshal(raw, &message); err != nil || message.Type != "cmd:error" || message.Payload.Code != "join_closed" {
						t.Fatalf("rejection message=%s error=%v", raw, err)
					}
				default:
					t.Fatal("rejected join did not inform the player")
				}
			}
			session := h.GetSession(moduleID)
			if tc.reconnect {
				if player := session.Players["player"]; player.Nickname != "Original" || player.Score != 42 {
					t.Fatalf("final result changed: %+v", player)
				}
			} else if !tc.accepted && len(session.Players) != 0 {
				t.Fatal("rejected join changed roster")
			}
		})
	}
}
