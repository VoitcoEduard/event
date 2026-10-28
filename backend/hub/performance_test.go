package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

// These fixtures exercise the existing production handlers, not an imitation
// of scoring. Logging is discarded so results isolate handler CPU/allocations;
// the remote load test includes the deployment's actual logging overhead.
func quietBenchmark(b *testing.B) {
	previous := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(previous) })
}

func quizBenchmarkFixture(players, history int, moduleID string) (*QuizHub, []*Client, *Client) {
	h := NewQuizHub(Dependencies{})
	now := time.Now().Unix()
	s := &models.QuizSession{
		EventID: "bench-event", ModuleID: moduleID, State: "question-active", CurrentIndex: history,
		Players: map[string]*models.Player{}, AnswerCounts: map[int]int{}, AnsweredPlayerIDs: map[string]struct{}{},
		Questions: make([]models.QuizQuestion, history+1), QuestionWindows: make([]models.QuestionWindow, history+1),
		PlayerSignals: make([]models.PlayerSignal, 0, players*(history+1)),
	}
	for q := range s.Questions {
		s.Questions[q] = models.QuizQuestion{ID: fmt.Sprintf("q-%d", q), Text: "Which option is correct?", Type: "classic", TimeLimit: 20, CorrectIndex: 0, Answers: []string{"A", "B", "C", "D"}}
		s.QuestionWindows[q] = models.QuestionWindow{StartTime: now, EndTime: now + 86400, GraceEndTime: now + 86402}
	}
	clients := make([]*Client, players)
	for p := range clients {
		id := fmt.Sprintf("player-%03d", p)
		s.Players[id] = &models.Player{ID: id, Nickname: id}
		clients[p] = &Client{ModuleID: moduleID, PlayerID: id, send: make(chan []byte, 4)}
		h.clients[clients[p]] = true
		for q := 0; q < history; q++ {
			s.PlayerSignals = append(s.PlayerSignals, models.PlayerSignal{PlayerID: id, QuestionIndex: q, ScoreEarned: 625, TimeSpent: 10000, AnswerIndices: []int{0}, Status: "on_time"})
		}
	}
	s.Batches = CalculateBatches(s.Questions)
	h.sessions[moduleID] = s
	observer := &Client{ModuleID: moduleID, IsProjector: true, send: make(chan []byte, players+8)}
	h.clients[observer] = true
	return h, clients, observer
}

func signalBytes(question int) []byte {
	return []byte(fmt.Sprintf(`{"type":"player:signal","payload":{"questionIndex":%d,"answerIndex":0,"scoreEarned":625,"timeSpent":10000}}`, question))
}

func resetBurst(s *models.QuizSession, history int) {
	s.PlayerSignals = s.PlayerSignals[:history]
	clear(s.AnswerCounts)
	clear(s.AnsweredPlayerIDs)
	for _, player := range s.Players {
		player.Score = 0
	}
}

// One operation is an entire question's answer burst, not one answer.
func BenchmarkQuizAnswerBurst(b *testing.B) {
	quietBenchmark(b)
	for _, players := range []int{50, 150, 300} {
		for _, history := range []int{0, 25, 100} {
			b.Run(fmt.Sprintf("players=%d/history=%d", players, history), func(b *testing.B) {
				h, clients, observer := quizBenchmarkFixture(players, history, "quiz")
				s := h.sessions["quiz"]
				payload := signalBytes(history)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					resetBurst(s, players*history)
					for _, client := range clients {
						h.ProcessMessage(client, payload)
						<-observer.send
					}
				}
				b.StopTimer()
				b.ReportMetric(float64(players), "answers/op")
				if len(s.AnsweredPlayerIDs) != players || len(h.clients) != players+1 {
					b.Fatal("burst did not reach all players or evicted the observer")
				}
			})
		}
	}
}

func BenchmarkQuizFullStateJSON(b *testing.B) {
	for _, history := range []int{0, 25, 100} {
		b.Run(fmt.Sprintf("players=150/history=%d", history), func(b *testing.B) {
			h, _, _ := quizBenchmarkFixture(150, history, "quiz")
			message := ServerMessage{Type: "state:update", Payload: h.sessions["quiz"]}
			data, err := json.Marshal(message)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := json.Marshal(message); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(len(data)), "payload-B")
		})
	}
}

// Shows unrelated connections scanned for a tiny host/projector update.
func BenchmarkQuizObserverBroadcast(b *testing.B) {
	for _, connections := range []int{150, 600, 1500} {
		b.Run(fmt.Sprintf("connections=%d", connections), func(b *testing.B) {
			h, _, observer := quizBenchmarkFixture(connections, 0, "quiz")
			message := ServerMessage{Type: "sync:minimal", Payload: map[string]any{"questionIndex": 0, "answeredCount": 100, "totalPlayers": 150}}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h.mu.Lock()
				h.broadcastToHostsAndProjectorsLocked("quiz", message)
				h.mu.Unlock()
				<-observer.send
			}
		})
	}
}

// Parallel handlers in separate games still contend on the same hub mutex.
// Each worker owns one session. State reset is included in measured time.
func BenchmarkQuizConcurrentGames(b *testing.B) {
	quietBenchmark(b)
	for _, games := range []int{1, 4} {
		b.Run(fmt.Sprintf("games=%d", games), func(b *testing.B) {
			h := NewQuizHub(Dependencies{})
			clients := make(chan *Client, games)
			for g := 0; g < games; g++ {
				id := fmt.Sprintf("quiz-%d", g)
				fixture, players, _ := quizBenchmarkFixture(150, 25, id)
				h.sessions[id] = fixture.sessions[id]
				clients <- players[0]
			}
			payload := signalBytes(25)
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					client := <-clients
					h.mu.Lock()
					resetBurst(h.sessions[client.ModuleID], 150*25)
					h.mu.Unlock()
					h.ProcessMessage(client, payload)
					clients <- client
				}
			})
		})
	}
}

func TestPerformanceFixtureExercisesRealHandler(t *testing.T) {
	h, players, observer := quizBenchmarkFixture(3, 2, "quiz")
	for _, player := range players {
		h.ProcessMessage(player, signalBytes(2))
	}
	s := h.sessions["quiz"]
	if len(s.AnsweredPlayerIDs) != 3 || len(s.PlayerSignals) != 9 || len(observer.send) != 3 {
		t.Fatal("fixture failed to submit real answers")
	}
	for _, player := range s.Players {
		if player.Score != 625 {
			t.Fatalf("score = %d", player.Score)
		}
	}
}
