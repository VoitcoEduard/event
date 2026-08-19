// Command loadtest exercises the same WebSocket protocol used by quiz players.
//
// Run this from a machine other than the NAS. It intentionally requires an
// explicit confirmation flag because it creates real players and answers in the
// selected quiz session.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
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

func (s *sampleSet) summary(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.values) == 0 {
		fmt.Printf("%-22s no successful samples (%d failures)\n", name+":", s.failures)
		return
	}
	values := append([]time.Duration(nil), s.values...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	p := func(percent float64) time.Duration {
		index := int(math.Ceil(percent*float64(len(values)))) - 1
		if index < 0 {
			index = 0
		}
		return values[index]
	}
	fmt.Printf("%-22s n=%d failures=%d min=%s p50=%s p95=%s p99=%s max=%s\n",
		name+":", len(values), s.failures, values[0].Round(time.Millisecond), p(.50).Round(time.Millisecond),
		p(.95).Round(time.Millisecond), p(.99).Round(time.Millisecond), values[len(values)-1].Round(time.Millisecond))
}

func newClient(wsURL string, timeout time.Duration) (*client, error) {
	dialer := websocket.Dialer{HandshakeTimeout: timeout}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}
	c := &client{conn: conn, messages: make(chan message, 128), done: make(chan struct{})}
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
					default: // State updates are disposable for a load generator.
					}
				}
			}
		}
	}()
	return c, nil
}

func (c *client) send(kind string, payload any) error {
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

func joinPlayer(wsURL, moduleID, id string, timeout time.Duration) (*client, time.Duration, error) {
	started := time.Now()
	c, err := newClient(wsURL, timeout)
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
	wsURL := flag.String("ws-url", "", "public WebSocket URL, for example wss://quiz.example.com/ws")
	moduleID := flag.String("module", "", "dedicated test quiz module ID")
	players := flag.Int("players", 100, "number of simulated player connections")
	question := flag.Int("question", 0, "active question index used in player:signal")
	timeout := flag.Duration("timeout", 15*time.Second, "connection and join timeout")
	warmup := flag.Duration("warmup", 3*time.Second, "settling time after all players join")
	pollURL := flag.String("poll-url", "", "optional HTTP URL to poll; {id} is replaced per player")
	pollInterval := flag.Duration("poll-interval", 3*time.Second, "poll interval per simulated player")
	requireActive := flag.Bool("require-active", true, "require host state question-active before sending answers")
	confirm := flag.String("confirm", "", "must equal I_UNDERSTAND_THIS_WRITES_TEST_ANSWERS")
	flag.Parse()

	if *wsURL == "" || *moduleID == "" || *players < 1 || *confirm != confirmation {
		fmt.Fprintln(os.Stderr, "Refusing to run. Set -ws-url, -module, a positive -players value, and -confirm "+confirmation)
		flag.PrintDefaults()
		os.Exit(2)
	}
	parsed, err := url.Parse(*wsURL)
	if err != nil || (parsed.Scheme != "ws" && parsed.Scheme != "wss") || parsed.Host == "" {
		log.Fatalf("invalid -ws-url %q", *wsURL)
	}

	fmt.Printf("Load test target: %s | module: %s | players: %d\n", parsed.Host, *moduleID, *players)
	fmt.Println("This will create test players and submit one answer per player. Use a disposable test module.")

	host, err := newClient(*wsURL, *timeout)
	if err != nil {
		log.Fatalf("host connection failed: %v", err)
	}
	defer host.close()
	if err := host.send("host:join", map[string]any{"moduleId": *moduleID}); err != nil {
		log.Fatalf("host join failed: %v", err)
	}
	hostState, err := waitForState(host, *timeout)
	if err != nil {
		log.Fatalf("host state failed: %v", err)
	}
	if *requireActive && hostState != "question-active" {
		log.Fatalf("module state is %q; start a question in the test quiz first, or use -require-active=false only for protocol testing", hostState)
	}

	var joins sampleSet
	clients := make([]*client, *players)
	var clientsMu sync.Mutex
	start := make(chan struct{})
	var joinWG sync.WaitGroup
	for i := 0; i < *players; i++ {
		joinWG.Add(1)
		go func(index int) {
			defer joinWG.Done()
			<-start
			c, elapsed, err := joinPlayer(*wsURL, *moduleID, fmt.Sprintf("loadtest-%03d", index+1), *timeout)
			if err != nil {
				joins.fail()
				return
			}
			joins.add(elapsed)
			clientsMu.Lock()
			clients[index] = c
			clientsMu.Unlock()
		}(i)
	}
	close(start)
	joinWG.Wait()
	joins.summary("WebSocket join")

	active := make([]*client, 0, *players)
	for _, c := range clients {
		if c != nil {
			active = append(active, c)
			defer c.close()
		}
	}
	if len(active) == 0 {
		log.Fatal("no players joined")
	}

	stopPolls := make(chan struct{})
	var polls sampleSet
	if *pollURL != "" {
		httpClient := &http.Client{Timeout: *timeout}
		for i := range active {
			id := fmt.Sprintf("loadtest-%03d", i+1)
			go func() {
				ticker := time.NewTicker(*pollInterval)
				defer ticker.Stop()
				for {
					requestURL := strings.ReplaceAll(*pollURL, "{id}", id)
					started := time.Now()
					resp, err := httpClient.Get(requestURL)
					if err != nil || resp.StatusCode >= 400 {
						polls.fail()
					} else {
						polls.add(time.Since(started))
					}
					if resp != nil {
						_, _ = io.Copy(io.Discard, resp.Body)
						_ = resp.Body.Close()
					}
					select {
					case <-stopPolls:
						return
					case <-ticker.C:
					}
				}
			}()
		}
	}

	fmt.Printf("%d players joined. Waiting %s before synchronized answer burst...\n", len(active), *warmup)
	time.Sleep(*warmup)

	// The host receives one sync:minimal message for every accepted answer.
	// Keep reading from the current point so the reported time reflects the burst.
	burstStart := time.Now()
	var sent atomic.Int64
	var sendWG sync.WaitGroup
	for _, c := range active {
		sendWG.Add(1)
		go func(player *client) {
			defer sendWG.Done()
			err := player.send("player:signal", map[string]any{
				"questionIndex": *question,
				"answerIndex":   0,
				"scoreEarned":   100,
				"timeSpent":     1000,
			})
			if err == nil {
				sent.Add(1)
			}
		}(c)
	}
	sendWG.Wait()

	// The first matching message may start above one if a test module was reused.
	// We count unique increasing answer counts until the expected total is reached.
	deadline := time.NewTimer(*timeout)
	maxAnswered := 0
	burstDone := time.Duration(0)
	for maxAnswered < len(active) {
		select {
		case <-deadline.C:
			fmt.Printf("Answer burst: sent=%d/%d, host observed=%d/%d before timeout\n", sent.Load(), len(active), maxAnswered, len(active))
			goto report
		case msg, ok := <-host.messages:
			if !ok {
				fmt.Println("Answer burst: host connection closed")
				goto report
			}
			if msg.Type != "sync:minimal" {
				continue
			}
			var syncState struct {
				QuestionIndex int `json:"questionIndex"`
				AnsweredCount int `json:"answeredCount"`
			}
			if json.Unmarshal(msg.Payload, &syncState) == nil && syncState.QuestionIndex == *question && syncState.AnsweredCount > maxAnswered {
				maxAnswered = syncState.AnsweredCount
				if maxAnswered >= len(active) {
					burstDone = time.Since(burstStart)
				}
			}
		}
	}
	deadline.Stop()
	fmt.Printf("Answer burst: sent=%d/%d, host observed=%d answers in %s\n", sent.Load(), len(active), maxAnswered, burstDone.Round(time.Millisecond))

report:
	close(stopPolls)
	// Give an in-flight HTTP request a moment to return before printing its summary.
	time.Sleep(100 * time.Millisecond)
	if *pollURL != "" {
		polls.summary("HTTP polling")
	}
}
