//go:build integration

package hub

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/testutils"
	"go.mongodb.org/mongo-driver/bson"
)

func TestWebSocketAdminCannotBeSelfAssigned(t *testing.T) {
	mongoDB := testutils.SetupMongoDB(t)
	defer testutils.TeardownMongoDB(t, mongoDB)
	store := auth.Store{DB: mongoDB.Client.Database("websocket_auth")}
	if err := store.EnsureIndexes(context.Background()); err != nil {
		t.Fatal(err)
	}
	user, err := store.UpsertGoogleUser(context.Background(), "issuer", "admin", "admin@example.test", "Admin")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Users().UpdateByID(context.Background(), user.ID, bson.M{"$set": bson.M{"status": "active"}})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.NewSession(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	service := &auth.Service{Store: store, Config: auth.Config{Mode: "dev", Environment: "development"}}
	hub := NewQuizHub(Dependencies{AuthService: service})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ServeWs(hub, w, r) }))
	defer server.Close()
	service.Config.BaseURL = server.URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	connect := func(cookie string) *websocket.Conn {
		t.Helper()
		header := http.Header{"Origin": []string{server.URL}}
		if cookie != "" {
			header.Set("Cookie", "admin_session_dev="+cookie)
		}
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("dial: %v (%v)", err, resp)
		}
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		return conn
	}
	for _, kind := range []string{"host:join", "teams:admin_join"} {
		conn := connect("")
		payload := `{"type":"` + kind + `","payload":{"moduleId":"quiz-1","eventId":"event-1"}}`
		if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
			t.Fatal(err)
		}
		_, _, err := conn.ReadMessage()
		if !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
			t.Errorf("anonymous %s: %v", kind, err)
		}
		conn.Close()
	}
	for _, payload := range []string{
		`{"type":"player:join","payload":{"moduleId":"quiz-1","nickname":"Guest"}}`,
		`{"type":"projector:join","payload":{"moduleId":"quiz-1","eventId":"event-1"}}`,
	} {
		guest := connect("")
		if err := guest.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
			t.Fatal(err)
		}
		_, message, err := guest.ReadMessage()
		if err != nil || !strings.Contains(string(message), `"cmd:state"`) {
			t.Fatalf("public subscription rejected: %s: %v", message, err)
		}
		guest.Close()
	}
	conn := connect(token)
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"host:join","payload":{"moduleId":"quiz-1"}}`)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatalf("approved host rejected: %v", err)
	}
	checkRoles := func(wantHost, wantTeamsAdmin, wantProjector bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			hub.mu.RLock()
			matched := false
			for client := range hub.clients {
				if client.AdminToken == token && client.IsHost == wantHost && client.IsTeamsAdmin == wantTeamsAdmin && client.IsProjector == wantProjector {
					matched = true
					break
				}
			}
			hub.mu.RUnlock()
			if matched {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("socket roles did not change to host=%t teamsAdmin=%t projector=%t", wantHost, wantTeamsAdmin, wantProjector)
	}
	checkRoles(true, false, false)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"player:join","payload":{"moduleId":"quiz-1","nickname":"Player"}}`)); err != nil {
		t.Fatal(err)
	}
	checkRoles(false, false, false)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"projector:join","payload":{"moduleId":"quiz-1"}}`)); err != nil {
		t.Fatal(err)
	}
	checkRoles(false, false, true)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"teams:admin_join","payload":{"eventId":"event-1"}}`)); err != nil {
		t.Fatal(err)
	}
	checkRoles(false, true, false)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"participant:join","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	checkRoles(false, false, false)
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"host:join","payload":{"moduleId":"quiz-1"}}`)); err != nil {
		t.Fatal(err)
	}
	checkRoles(true, false, false)
	if err := store.Revoke(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"host:next_question","payload":{}}`)); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		_, _, err := conn.ReadMessage()
		if websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
			return
		}
		if err != nil {
			t.Fatalf("revocation not enforced: %v", err)
		}
	}
}
