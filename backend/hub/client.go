package hub

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/go-expo-sample/backend/auth"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 65536
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return r.Context().Value(allowedOriginKey{}) == true },
}

type allowedOriginKey struct{}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *QuizHub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Session tracking
	ModuleID     string // The quiz/module ID this client is connected to
	EventID      string // The event ID this client is connected to (for big screen)
	PlayerID     string // The player ID (if player)
	IsHost       bool   // True if this client is the host
	IsProjector  bool   // True if this client is a projector display
	IsTeamsAdmin bool   // True if this client is a teams admin
	AdminToken   string
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				c.hub.mu.RLock()
				privileged := c.IsHost || c.IsTeamsAdmin
				c.hub.mu.RUnlock()
				if privileged && !c.hub.AdminActive(c.AdminToken) {
					c.Reject()
					return
				}
			}
		}
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		// Process message through Hub logic
		c.hub.ProcessMessage(c, message)
	}
}

func (c *Client) Reject() {
	if c.conn == nil {
		return
	}
	_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Administrator access required"), time.Now().Add(time.Second))
	_ = c.conn.Close()
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			w.Write([]byte("\n")) // Add newline delimiter between messages

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				msg := <-c.send
				w.Write(msg)
				w.Write([]byte("\n"))
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ...

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *QuizHub, w http.ResponseWriter, r *http.Request) {
	if hub.authService == nil {
		http.Error(w, "Authentication unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("Origin") != hub.authService.Config.BaseURL || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	token := hub.authService.Token(r)
	if token != "" {
		if _, _, err := hub.authService.Current(r); err != nil {
			// Expired sessions may reconnect as players, but database errors fail closed.
			if err != auth.ErrNoSession {
				http.Error(w, "Session unavailable", http.StatusServiceUnavailable)
				return
			}
			token = ""
		}
	}
	r = r.WithContext(context.WithValue(r.Context(), allowedOriginKey{}, true))
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), AdminToken: token}
	if !hub.Register(client) {
		return
	}

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
