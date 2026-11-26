package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
)

func TestPublicQuizStateOmitsIndividualPlayersAndAnswers(t *testing.T) {
	qh := hub.NewQuizHub(hub.Dependencies{})
	qh.CreateSession("m", "e")
	qh.UpdateSession("m", func(s *models.QuizSession) {
		s.State, s.CurrentIndex = "question-active", 2
		s.Players["private-id"] = &models.Player{ID: "private-id", Nickname: "private-name", Score: 100}
		s.PlayerSignals = []models.PlayerSignal{{PlayerID: "private-id", AnswerValue: 12345}}
		s.AnsweredPlayerIDs = map[string]struct{}{"private-id": {}}
	})
	h := New(Dependencies{QuizHub: qh})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/{eventID}/modules/{moduleID}/quiz/state", h.GetQuizState)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/events/e/modules/m/quiz/state", nil))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("public state response: %d %s", w.Code, w.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"players", "playerSignals", "questions", "answerCounts"} {
		if _, exists := state[field]; exists {
			t.Fatalf("public state exposes %s", field)
		}
	}
	if state["state"] != "question-active" || state["currentIndex"] != float64(2) || state["totalPlayers"] != float64(1) || state["answeredCount"] != float64(1) {
		t.Fatalf("public progress missing: %v", state)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/events/other/modules/m/quiz/state", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-event session returned %d", w.Code)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/events/e/modules/unstarted/quiz/state", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("unstarted session returned %d", w.Code)
	}
	var inactive map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &inactive); err != nil {
		t.Fatal(err)
	}
	if inactive["state"] != "inactive" {
		t.Fatalf("unstarted session state = %v, want inactive", inactive["state"])
	}
}

func TestGetEventRejectsOversizedUserIDBeforeStorage(t *testing.T) {
	h := New(Dependencies{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events/{eventID}", h.GetEvent)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/events/e?userId="+strings.Repeat("x", 121), nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("oversized user ID returned %d", w.Code)
	}
}

func TestSimpleQuizQueueOptionsRejectsMalformedAndOutOfRangeNumbers(t *testing.T) {
	for _, tc := range []struct {
		query string
		index int
		limit int
		valid bool
	}{
		{"", -1, 20, true},
		{"questionIndex=-1&limit=100", -1, 100, true},
		{"questionIndex=1&limit=1", 1, 1, true},
		{"questionIndex=0junk", 0, 0, false},
		{"questionIndex=text", 0, 0, false},
		{"questionIndex=-2", 0, 0, false},
		{"questionIndex=2", 0, 0, false},
		{"questionIndex=9999999999999999999999999999", 0, 0, false},
		{"limit=20junk", 0, 0, false},
		{"limit=-1", 0, 0, false},
		{"limit=0", 0, 0, false},
		{"limit=101", 0, 0, false},
		{"limit=9999999999999999999999999999", 0, 0, false},
	} {
		t.Run(tc.query, func(t *testing.T) {
			query, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			index, limit, err := simpleQuizQueueOptions(query, 2)
			if (err == nil) != tc.valid {
				t.Fatalf("validation error = %v, valid = %v", err, tc.valid)
			}
			if tc.valid && (index != tc.index || limit != tc.limit) {
				t.Fatalf("options = %d/%d, want %d/%d", index, limit, tc.index, tc.limit)
			}
		})
	}
}

func TestRESTStartAndNextAlignWithHub(t *testing.T) {
	qh := hub.NewQuizHub(hub.Dependencies{})
	t.Cleanup(qh.Shutdown)
	qh.CreateSession("m1", "e1")
	qh.UpdateSession("m1", func(s *models.QuizSession) {
		s.Players["p1"] = &models.Player{ID: "p1", Nickname: "Alice", Score: 500}
		s.Players["p2"] = &models.Player{ID: "p2", Nickname: "Bob", Score: 300}
		s.Streaks = map[string]int{"p1": 2, "p2": 3}
	})

	h := New(Dependencies{QuizHub: qh})
	h.getEvent = func(_ context.Context, id string) (*models.Event, error) {
		if id != "e1" {
			return nil, nil
		}
		return &models.Event{
			ID: "e1",
			Modules: []models.Module{{
				ID:   "m1",
				Type: models.ModuleTypeQuiz,
				Data: map[string]any{
					"questions": []map[string]any{
						{"id": "q1", "type": "classic", "text": "Q1", "answers": []string{"A", "B"}, "correctIndex": 0, "explainTime": 60},
						{"id": "q2", "type": "classic", "text": "Q2", "answers": []string{"C", "D"}, "correctIndex": 1, "explainTime": 60},
					},
				},
			}},
		}, nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /events/{eventID}/modules/{moduleID}/quiz/start", h.StartQuiz)
	mux.HandleFunc("POST /events/{eventID}/modules/{moduleID}/quiz/next", h.NextQuestion)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/events/e1/modules/m1/quiz/start", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("StartQuiz status = %d: %s", w.Code, w.Body.String())
	}
	sess := qh.GetSession("m1")
	if sess.State != "question-presenting" || sess.CurrentIndex != 0 || len(sess.Players) != 2 || sess.Players["p1"].Score != 0 {
		t.Fatalf("unexpected started session: %+v", sess)
	}

	qh.UpdateSession("m1", func(s *models.QuizSession) {
		s.Streaks["p1"] = 1
		s.Streaks["p2"] = 3
		s.AnsweredPlayerIDs = map[string]struct{}{"p1": {}}
	})

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/events/other/modules/m1/quiz/next", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("cross-event NextQuestion status = %d, want 404", w.Code)
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/events/e1/modules/m1/quiz/next", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("NextQuestion status = %d: %s", w.Code, w.Body.String())
	}
	sess = qh.GetSession("m1")
	if sess.State != "question-presenting" || sess.CurrentIndex != 1 || len(sess.AnsweredPlayerIDs) != 0 {
		t.Fatalf("unexpected next session: %+v", sess)
	}
	if sess.Streaks["p1"] != 1 || sess.Streaks["p2"] != 0 {
		t.Fatalf("unexpected streaks after NextQuestion: %+v", sess.Streaks)
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/events/e1/modules/m1/quiz/next", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("past-end NextQuestion status = %d, want 400", w.Code)
	}
	if qh.GetSession("m1").CurrentIndex != 1 {
		t.Fatalf("CurrentIndex mutated past end")
	}

	qh.UpdateSession("m1", func(s *models.QuizSession) {
		s.State = "finished"
	})
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/events/e1/modules/m1/quiz/next", nil))
	if w.Code != http.StatusConflict {
		t.Fatalf("finished NextQuestion status = %d, want 409", w.Code)
	}
}
