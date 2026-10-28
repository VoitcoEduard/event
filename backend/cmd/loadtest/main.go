// Command loadtest exercises the same WebSocket protocol used by quiz players.
//
// Run this from a machine other than the NAS. It intentionally requires an
// explicit confirmation flag because it creates real players and answers in the
// selected quiz session.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const confirmation = "I_UNDERSTAND_THIS_WRITES_TEST_ANSWERS"

type message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type client struct {
	conn     *websocket.Conn
	messages chan message
	done     chan struct{}
	once     sync.Once
	dropped  atomic.Int64
	timeout  time.Duration
}

type sampleSet struct {
	mu       sync.Mutex
	values   []time.Duration
	failures int
}

func (s *sampleSet) add(d time.Duration) {
	s.mu.Lock()
	s.values = append(s.values, d)
	s.mu.Unlock()
}

func (s *sampleSet) fail() {
	s.mu.Lock()
	s.failures++
	s.mu.Unlock()
}

func newClient(wsURL, origin string, timeout time.Duration) (*client, error) {
	dialer := websocket.Dialer{HandshakeTimeout: timeout}
	conn, _, err := dialer.Dial(wsURL, http.Header{"Origin": []string{origin}})
	if err != nil {
		return nil, err
	}
	c := &client{conn: conn, messages: make(chan message, 4096), done: make(chan struct{}), timeout: timeout}
	go func() {
		defer close(c.done)
		defer close(c.messages)
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			for _, line := range strings.Split(string(payload), "\n") {
				if strings.TrimSpace(line) == "" {
					continue
				}
				var msg message
				if json.Unmarshal([]byte(line), &msg) == nil {
					select {
					case c.messages <- msg:
					default:
						c.dropped.Add(1) // A saturated generator must fail, not produce a false pass.
					}
				}
			}
		}
	}()
	return c, nil
}

func (c *client) send(kind string, payload any) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.conn.WriteJSON(map[string]any{"type": kind, "payload": payload})
}

func (c *client) close() {
	c.once.Do(func() {
		_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "load test finished"), time.Now().Add(time.Second))
		_ = c.conn.Close()
	})
}

func waitForState(c *client, timeout time.Duration) (string, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return "", fmt.Errorf("timed out waiting for cmd:state")
		case msg, ok := <-c.messages:
			if !ok {
				return "", fmt.Errorf("connection closed before cmd:state")
			}
			if msg.Type != "cmd:state" {
				continue
			}
			var state struct {
				State string `json:"state"`
			}
			if err := json.Unmarshal(msg.Payload, &state); err != nil {
				return "", err
			}
			return state.State, nil
		}
	}
}

func joinPlayer(wsURL, origin, moduleID, id string, timeout time.Duration) (*client, time.Duration, error) {
	started := time.Now()
	c, err := newClient(wsURL, origin, timeout)
	if err != nil {
		return nil, 0, err
	}
	if err := c.send("player:join", map[string]any{"moduleId": moduleID, "nickname": id, "userId": id}); err != nil {
		c.close()
		return nil, 0, err
	}
	if _, err := waitForState(c, timeout); err != nil {
		c.close()
		return nil, 0, err
	}
	return c, time.Since(started), nil
}

func main() {
	if err := command(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
