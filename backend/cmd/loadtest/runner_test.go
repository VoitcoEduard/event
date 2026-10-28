package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/auth"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
)

func fixtureOptions(wsURL string) options {
	return options{wsURL: wsURL, module: "test", mode: "burst", players: 5,
		timeout: 2 * time.Second, pollInterval: 5 * time.Millisecond, hold: 20 * time.Millisecond,
		maxBurst: time.Second, maxJoinP95: time.Second, maxPollP95: time.Second, requireActive: true}
}

func loadTestServer(t *testing.T) (*httptest.Server, *hub.QuizHub) {
	t.Helper()
	service := &auth.Service{}
	h := hub.NewQuizHub(hub.Dependencies{AuthService: service})
	h.CreateSession("test", "event")
	h.UpdateSession("test", func(s *models.QuizSession) {
		s.State = "question-active"
		s.Questions = []models.QuizQuestion{{TimeLimit: 120, CorrectIndex: 0}}
		s.QuestionWindows = []models.QuestionWindow{{StartTime: time.Now().Unix(), EndTime: time.Now().Unix() + 120, GraceEndTime: time.Now().Unix() + 122}}
	})
	var participants sync.Map
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) { hub.ServeWs(h, w, r) })
	mux.HandleFunc("POST /events/event/participants/join", func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			DisplayName string `json:"displayName"`
		}
		_ = json.NewDecoder(r.Body).Decode(&p)
		id := "persistent-" + p.DisplayName
		participants.Store(id, true)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
	})
	mux.HandleFunc("GET /events/event", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, ok := participants.Load(r.URL.Query().Get("userId")); !ok {
			_, _ = w.Write([]byte(`{"id":"event"}`))
			return
		}
		_, _ = w.Write([]byte(`{"participantStatus":{"totalPoints":0}}`))
	})
	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		_, _ = w.Write([]byte("fixture-image"))
	})
	server := httptest.NewServer(mux)
	service.Config.BaseURL = server.URL
	t.Cleanup(func() { h.Shutdown(); server.Close() })
	return server, h
}

func TestLoadRunUsesRealWebSocketProtocolAndRegisteredPolling(t *testing.T) {
	server, _ := loadTestServer(t)
	o := fixtureOptions("ws" + strings.TrimPrefix(server.URL, "http") + "/ws")
	o.pollURL, o.mediaURL = server.URL+"/events/event?userId={id}", server.URL+"/image"
	if err := o.validate(); err != nil {
		t.Fatal(err)
	}
	r := runLoad(o)
	if !r.Passed || r.Observed != o.players || r.Polls.Count == 0 || r.Media.Count != o.players {
		t.Fatalf("load report: %+v", r)
	}
	// Reusing the same question must fail, even with fresh player identifiers.
	r = runLoad(o)
	if r.Passed || !strings.Contains(strings.Join(r.Problems, " "), "already has answers") {
		t.Fatalf("reused question was not rejected: %+v", r)
	}
}

func TestIncompleteBurstFails(t *testing.T) {
	server, h := loadTestServer(t)
	h.UpdateSession("test", func(s *models.QuizSession) {
		s.QuestionWindows[0].EndTime = time.Now().Unix() - 10
		s.QuestionWindows[0].GraceEndTime = time.Now().Unix() - 8
	})
	o := fixtureOptions("ws" + strings.TrimPrefix(server.URL, "http") + "/ws")
	o.timeout = 100 * time.Millisecond
	if err := o.validate(); err != nil {
		t.Fatal(err)
	}
	r := runLoad(o)
	if r.Passed || r.Observed != 0 {
		t.Fatalf("expired answers passed: %+v", r)
	}
}

func TestIdleModeDoesNotSubmitAnswers(t *testing.T) {
	server, h := loadTestServer(t)
	h.UpdateSession("test", func(s *models.QuizSession) { s.State = "lobby" })
	o := fixtureOptions("ws" + strings.TrimPrefix(server.URL, "http") + "/ws")
	o.mode = "idle"
	o.requireActive = false
	if err := o.validate(); err != nil {
		t.Fatal(err)
	}
	r := runLoad(o)
	if !r.Passed || len(h.GetSession("test").PlayerSignals) != 0 {
		t.Fatalf("idle report: %+v", r)
	}
}

func TestLatencyLimitMakesRunFail(t *testing.T) {
	server, _ := loadTestServer(t)
	o := fixtureOptions("ws" + strings.TrimPrefix(server.URL, "http") + "/ws")
	o.maxBurst = time.Nanosecond
	if err := o.validate(); err != nil {
		t.Fatal(err)
	}
	r := runLoad(o)
	if r.Passed || !strings.Contains(strings.Join(r.Problems, " "), "latency limit") {
		t.Fatalf("latency limit not enforced: %+v", r)
	}
}

func TestHTTPFailuresCannotPassAsSuccessfulPolling(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
	}{
		{"server error", "application/json", `{}`, 500},
		{"SPA fallback", "text/html", `<html>login</html>`, 200},
		{"missing participant", "application/json", `{"id":"event"}`, 200},
		{"truncated JSON", "application/json", `{"participantStatus":`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			var samples sampleSet
			requestSample(context.Background(), server.Client(), server.URL+"?userId=p", &samples)
			if m := samples.metric(); m.Count != 0 || m.Failures != 1 {
				t.Fatalf("invalid response passed: %+v", m)
			}
		})
	}
}

func TestInvalidOptions(t *testing.T) {
	for _, mutate := range []func(*options){
		func(o *options) { o.players = 0 }, func(o *options) { o.pollInterval = 0 },
		func(o *options) { o.wsURL = "https://example.test/ws" },
		func(o *options) { o.wsURL = "ws://user:password@example.test/ws" },
		func(o *options) { o.mode = "unknown" }, func(o *options) { o.question = -1 },
	} {
		o := fixtureOptions("ws://example.test/ws")
		mutate(&o)
		if o.validate() == nil {
			t.Fatalf("invalid options accepted: %+v", o)
		}
	}
}

func TestSamplePercentiles(t *testing.T) {
	var s sampleSet
	for i := 1; i <= 100; i++ {
		s.add(time.Duration(i) * time.Millisecond)
	}
	s.fail()
	m := s.metric()
	if m.P50MS != 50 || m.P95MS != 95 || m.P99MS != 99 || m.MaxMS != 100 || m.Failures != 1 {
		t.Fatalf("percentiles: %+v", m)
	}
}
