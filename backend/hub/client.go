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
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 65536
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return r.Context().Value(allowedOriginKey{}) == true },
}

type allowedOriginKey struct{}

type Client struct {
	hub *QuizHub

	conn *websocket.Conn

	send chan []byte

	ModuleID     string
	EventID      string
	PlayerID     string
	IsHost       bool
	IsProjector  bool
	IsTeamsAdmin bool
	AdminToken   string
	quizBinding  *quizPlayerBinding
}

type quizPlayerBinding struct{ eventID, moduleID, playerID string }

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
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			w.Write([]byte("\n"))

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

	go client.writePump()
	go client.readPump()
}
