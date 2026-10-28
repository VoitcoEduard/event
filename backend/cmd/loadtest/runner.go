package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type options struct {
	wsURL, origin, module, pollURL, mediaURL, mode, output string
	registerURL                                            string
	players, question                                      int
	timeout, warmup, hold, pollInterval, spread            time.Duration
	maxBurst, maxJoinP95, maxPollP95                       time.Duration
	requireActive                                          bool
}

type metric struct {
	Count    int     `json:"count"`
	Failures int     `json:"failures"`
	P50MS    float64 `json:"p50Ms"`
	P95MS    float64 `json:"p95Ms"`
	P99MS    float64 `json:"p99Ms"`
	MaxMS    float64 `json:"maxMs"`
}

func (s *sampleSet) metric() metric {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := metric{Count: len(s.values), Failures: s.failures}
	if len(s.values) == 0 {
		return m
	}
	v := append([]time.Duration(nil), s.values...)
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	p := func(q float64) float64 {
		return float64(v[int(math.Ceil(q*float64(len(v))))-1]) / float64(time.Millisecond)
	}
	m.P50MS, m.P95MS, m.P99MS, m.MaxMS = p(.50), p(.95), p(.99), p(1)
	return m
}

type report struct {
	StartedAt      time.Time `json:"startedAt"`
	Target         string    `json:"target"`
	Module         string    `json:"module"`
	Mode           string    `json:"mode"`
	Players        int       `json:"players"`
	Joined         int       `json:"joined"`
	Sent           int       `json:"sent"`
	Observed       int       `json:"observed"`
	BurstMS        float64   `json:"burstMs"`
	ElapsedMS      float64   `json:"elapsedMs"`
	Disconnects    int       `json:"disconnects"`
	DroppedUpdates int64     `json:"generatorDroppedUpdates"`
	Joins          metric    `json:"joins"`
	Polls          metric    `json:"polls"`
	Media          metric    `json:"media"`
	Passed         bool      `json:"passed"`
	Problems       []string  `json:"problems"`
}

func command() error {
	var o options
	flag.StringVar(&o.wsURL, "ws-url", "", "WebSocket URL")
	flag.StringVar(&o.origin, "origin", "", "PUBLIC_BASE_URL; defaults to the HTTP equivalent of ws-url")
	flag.StringVar(&o.module, "module", "", "disposable test quiz module ID")
	flag.IntVar(&o.players, "players", 100, "simulated players")
	flag.IntVar(&o.question, "question", 0, "active question index")
	flag.DurationVar(&o.timeout, "timeout", 15*time.Second, "join/request/burst timeout")
	flag.DurationVar(&o.warmup, "warmup", 3*time.Second, "settling period before answers")
	flag.DurationVar(&o.hold, "hold", 30*time.Second, "keep connections and polling alive after the burst (or in idle mode)")
	flag.DurationVar(&o.spread, "join-spread", 0, "spread joins over this duration; zero tests a simultaneous join storm")
	flag.StringVar(&o.pollURL, "poll-url", "", "event GET endpoint; {id} is replaced by the actual player's unique ID")
	flag.StringVar(&o.registerURL, "register-url", "", "participant join endpoint; inferred from poll-url when it contains {id}")
	flag.DurationVar(&o.pollInterval, "poll-interval", 3*time.Second, "poll interval per player")
	flag.StringVar(&o.mediaURL, "media-url", "", "optional existing image/media URL fetched once by every player")
	flag.StringVar(&o.mode, "mode", "burst", "burst or idle (connections/polling only)")
	flag.StringVar(&o.output, "output", "", "write a JSON report to this file")
	flag.DurationVar(&o.maxBurst, "max-burst", time.Second, "maximum time until observer sees all answers")
	flag.DurationVar(&o.maxJoinP95, "max-join-p95", 5*time.Second, "maximum 95th percentile join latency")
	flag.DurationVar(&o.maxPollP95, "max-poll-p95", time.Second, "maximum 95th percentile polling latency")
	flag.BoolVar(&o.requireActive, "require-active", true, "require question-active state (use false for an idle lobby test)")
	confirm := flag.String("confirm", "", "must equal "+confirmation)
	flag.Parse()
	if *confirm != confirmation {
		return errors.New("use a disposable module and pass -confirm " + confirmation)
	}
	if err := o.validate(); err != nil {
		return err
	}
	r := runLoad(o)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	if o.output != "" {
		if err := os.WriteFile(o.output, append(data, '\n'), 0600); err != nil {
			return err
		}
	}
	if !r.Passed {
		return errors.New("load test FAILED; see problems in the report")
	}
	return nil
}

func (o *options) validate() error {
	u, err := url.Parse(o.wsURL)
	if err != nil || u.Host == "" || (u.Scheme != "ws" && u.Scheme != "wss") || u.User != nil || u.RawQuery != "" {
		return errors.New("ws-url must be a ws/wss URL without credentials or query parameters")
	}
	if o.origin == "" {
		scheme := "http"
		if u.Scheme == "wss" {
			scheme = "https"
		}
		o.origin = scheme + "://" + u.Host
	}
	if o.module == "" || o.players < 1 || o.question < 0 || (o.mode != "burst" && o.mode != "idle") {
		return errors.New("provide a module, positive players, nonnegative question and mode burst/idle")
	}
	if o.timeout <= 0 || o.pollInterval <= 0 || o.warmup < 0 || o.hold < 0 || o.spread < 0 || o.maxBurst <= 0 || o.maxJoinP95 <= 0 || o.maxPollP95 <= 0 {
		return errors.New("timeouts, polling interval and latency limits must be positive; warmup/hold/join-spread must be nonnegative")
	}
	if strings.Contains(o.pollURL, "{id}") && o.registerURL == "" {
		poll, err := url.Parse(o.pollURL)
		if err != nil || !strings.Contains(poll.Path, "/events/") || poll.Query().Get("userId") != "{id}" {
			return errors.New("personal polling requires an /events/EVENT_ID?userId={id} URL or an explicit register-url")
		}
		poll.Path = strings.TrimRight(poll.Path, "/") + "/participants/join"
		poll.RawQuery = ""
		o.registerURL = poll.String()
	}
	for _, value := range []string{o.pollURL, o.mediaURL, o.registerURL} {
		if value == "" {
			continue
		}
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
			return errors.New("poll-url and media-url must be HTTP(S) URLs without embedded credentials")
		}
	}
	return nil
}

type quizSnapshot struct {
	State         string `json:"state"`
	CurrentIndex  int    `json:"currentIndex"`
	PlayerSignals []struct {
		QuestionIndex int `json:"questionIndex"`
	} `json:"playerSignals"`
	QuestionWindows []struct {
		EndTime int64 `json:"endTime"`
	} `json:"questionWindows"`
}

// The projector is a public observer, so this tests the real authorization
// rules without requiring administrator cookies or changing game state.
func observerSnapshot(c *client, module string, timeout time.Duration) (*quizSnapshot, error) {
	if err := c.send("projector:join", map[string]any{"moduleId": module}); err != nil {
		return nil, err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return nil, errors.New("observer snapshot timed out; check -origin against PUBLIC_BASE_URL")
		case msg, ok := <-c.messages:
			if !ok {
				return nil, errors.New("observer disconnected")
			}
			if msg.Type != "state:update" {
				continue
			}
			var s quizSnapshot
			if err := json.Unmarshal(msg.Payload, &s); err != nil {
				return nil, err
			}
			return &s, nil
		}
	}
}

func runLoad(o options) (r report) {
	started := time.Now()
	r = report{StartedAt: started.UTC(), Target: o.wsURL, Module: o.module, Mode: o.mode, Players: o.players, Problems: []string{}}
	var joins, polls, media sampleSet
	defer func() {
		r.ElapsedMS = float64(time.Since(started)) / float64(time.Millisecond)
		r.Joins, r.Polls, r.Media = joins.metric(), polls.metric(), media.metric()
		if r.Joined != o.players || r.Joins.Failures > 0 {
			r.Problems = append(r.Problems, "not all players joined")
		}
		if r.Joins.P95MS > float64(o.maxJoinP95)/float64(time.Millisecond) {
			r.Problems = append(r.Problems, "join p95 exceeded limit")
		}
		if r.Polls.Failures > 0 || r.Media.Failures > 0 {
			r.Problems = append(r.Problems, "HTTP request failures")
		}
		if o.pollURL != "" && r.Joined > 0 && r.Polls.Count == 0 {
			r.Problems = append(r.Problems, "no successful polling samples")
		}
		if r.Polls.P95MS > float64(o.maxPollP95)/float64(time.Millisecond) {
			r.Problems = append(r.Problems, "poll p95 exceeded limit")
		}
		if r.Disconnects > 0 || r.DroppedUpdates > 0 {
			r.Problems = append(r.Problems, "connections lost or load generator dropped updates")
		}
		r.Passed = len(r.Problems) == 0
	}()
	observer, err := newClient(o.wsURL, o.origin, o.timeout)
	if err != nil {
		r.Problems = append(r.Problems, "observer connection: "+err.Error())
		return
	}
	defer observer.close()
	snapshot, err := observerSnapshot(observer, o.module, o.timeout)
	if err != nil {
		r.Problems = append(r.Problems, err.Error())
		return
	}
	if (o.requireActive || o.mode == "burst") && (snapshot.State != "question-active" || snapshot.CurrentIndex != o.question) {
		r.Problems = append(r.Problems, "start the specified question in the disposable quiz before testing")
		return
	}
	if o.mode == "burst" {
		if len(snapshot.QuestionWindows) <= o.question || time.Until(time.Unix(snapshot.QuestionWindows[o.question].EndTime, 0)) < o.timeout+o.warmup+o.spread {
			r.Problems = append(r.Problems, "question scoring window is too short; start a fresh question with a longer time limit")
			return
		}
		for _, signal := range snapshot.PlayerSignals {
			if signal.QuestionIndex == o.question {
				r.Problems = append(r.Problems, "question already has answers; use a fresh question to avoid a false pass")
				return
			}
		}
	}
	type entrant struct {
		id string
		c  *client
	}
	entrants := make([]entrant, o.players)
	runID := fmt.Sprintf("loadtest-%d", time.Now().UnixNano())
	registrationClient := &http.Client{Timeout: o.timeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	var wg sync.WaitGroup
	for i := range entrants {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if o.spread > 0 {
				time.Sleep(time.Duration(int64(o.spread) * int64(i) / int64(o.players)))
			}
			id := fmt.Sprintf("%s-%03d", runID, i)
			joinStart := time.Now()
			if o.registerURL != "" {
				var err error
				id, err = registerParticipant(registrationClient, o.registerURL, id)
				if err != nil {
					joins.fail()
					return
				}
			}
			c, elapsed, err := joinPlayer(o.wsURL, o.origin, o.module, id, o.timeout)
			if err != nil {
				joins.fail()
				return
			}
			entrants[i] = entrant{id, c}
			if o.registerURL != "" {
				elapsed = time.Since(joinStart)
			}
			joins.add(elapsed)
		}(i)
	}
	wg.Wait()
	active := []entrant{}
	for _, e := range entrants {
		if e.c != nil {
			active = append(active, e)
			defer e.c.close()
		}
	}
	r.Joined = len(active)
	if len(active) == 0 {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	transport := &http.Transport{MaxIdleConns: o.players * 2, MaxIdleConnsPerHost: o.players * 2}
	defer transport.CloseIdleConnections()
	httpClient := &http.Client{Timeout: o.timeout, Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	var pollWG sync.WaitGroup
	defer func() { cancel(); pollWG.Wait() }()
	for _, e := range active {
		if o.pollURL != "" {
			pollWG.Add(1)
			go func(e entrant) {
				defer pollWG.Done()
				ticker := time.NewTicker(o.pollInterval)
				defer ticker.Stop()
				for {
					requestSample(ctx, httpClient, strings.ReplaceAll(o.pollURL, "{id}", e.id), &polls)
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
					}
				}
			}(e)
		}
		if o.mediaURL != "" {
			wg.Add(1)
			go func() { defer wg.Done(); requestSample(ctx, httpClient, o.mediaURL, &media) }()
		}
	}
	wg.Wait()
	time.Sleep(o.warmup)
	if o.mode == "burst" {
		start := time.Now()
		var sent atomic.Int64
		for _, e := range active {
			wg.Add(1)
			go func(c *client) {
				defer wg.Done()
				if err := c.send("player:signal", map[string]any{"questionIndex": o.question, "answerIndex": 0, "scoreEarned": 100, "timeSpent": 1000}); err == nil {
					sent.Add(1)
				}
			}(e.c)
		}
		wg.Wait()
		r.Sent = int(sent.Load())
		r.Observed, err = observeBurst(observer, o.question, len(active), o.timeout)
		r.BurstMS = float64(time.Since(start)) / float64(time.Millisecond)
		if err != nil || r.Sent != o.players || r.Observed != o.players {
			r.Problems = append(r.Problems, "answer burst incomplete (question may have expired)")
		}
		if time.Since(start) > o.maxBurst {
			r.Problems = append(r.Problems, "answer burst exceeded latency limit")
		}
	}
	// Drain observer updates during the hold period so longer runs also exercise
	// keepalives and cache expiry without filling the generator's queue.
	timer := time.NewTimer(o.hold)
	defer timer.Stop()
hold:
	for {
		select {
		case <-timer.C:
			break hold
		case _, ok := <-observer.messages:
			if !ok {
				break hold
			}
		}
	}
	for _, e := range active {
		r.DroppedUpdates += e.c.dropped.Load()
		select {
		case <-e.c.done:
			r.Disconnects++
		default:
		}
	}
	r.DroppedUpdates += observer.dropped.Load()
	select {
	case <-observer.done:
		r.Disconnects++
	default:
	}
	return
}

func observeBurst(c *client, question, expected int, timeout time.Duration) (int, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	observed := 0
	for observed < expected {
		select {
		case <-timer.C:
			return observed, errors.New("answer observation timed out")
		case msg, ok := <-c.messages:
			if !ok {
				return observed, errors.New("observer disconnected")
			}
			var state struct {
				QuestionIndex int `json:"questionIndex"`
				AnsweredCount int `json:"answeredCount"`
			}
			if msg.Type == "sync:minimal" && json.Unmarshal(msg.Payload, &state) == nil && state.QuestionIndex == question && state.AnsweredCount > observed {
				observed = state.AnsweredCount
			}
		}
	}
	return observed, nil
}

func requestSample(ctx context.Context, client *http.Client, endpoint string, samples *sampleSet) {
	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		samples.fail()
		return
	}
	response, err := client.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			samples.fail()
		}
		return
	}
	var readErr error
	if req.URL.Query().Get("userId") != "" {
		var event struct {
			ParticipantStatus json.RawMessage `json:"participantStatus"`
		}
		readErr = json.NewDecoder(io.LimitReader(response.Body, 32*1024*1024)).Decode(&event)
		if len(event.ParticipantStatus) == 0 || string(event.ParticipantStatus) == "null" {
			readErr = errors.New("personal event response omitted participantStatus")
		}
	}
	_, drainErr := io.Copy(io.Discard, response.Body)
	if drainErr != nil {
		readErr = drainErr
	}
	_ = response.Body.Close()
	if ctx.Err() != nil {
		return
	}
	if readErr != nil || response.StatusCode < 200 || response.StatusCode >= 300 || !validResponse(response.Header) {
		samples.fail()
		return
	}
	samples.add(time.Since(started))
}

// SPA fallbacks and login redirects must not count as successful API/media tests.
func validResponse(headers http.Header) bool {
	contentType := headers.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return false
	}
	return contentType != "" // includes JSON, image, audio and binary media types
}

func registerParticipant(client *http.Client, endpoint, name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"joinKey": name, "displayName": name})
	response, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("participant registration: HTTP %d", response.StatusCode)
	}
	var p struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&p); err != nil {
		return "", err
	}
	if p.ID == "" {
		return "", errors.New("participant registration returned no ID")
	}
	return p.ID, nil
}
