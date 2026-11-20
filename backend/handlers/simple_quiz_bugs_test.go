package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
)

func seedHandlerSimpleQuiz(status string) (*Handler, *models.SimpleQuizSession) {
	qh := hub.NewQuizHub(hub.Dependencies{})
	now := time.Now().UTC()
	s := &models.SimpleQuizSession{
		EventID:      "e1",
		ModuleID:     "m1",
		Status:       status,
		CurrentIndex: 0,
		AllowReveal:  true,
		Questions: []models.SimpleQuizQuestionSnapshot{
			{ID: "q1", Text: "Q1", Points: 50, CorrectAnswerText: "Paris", JudgeNotes: "Accept Pariz"},
			{ID: "q2", Text: "Q2", Points: 50, CorrectAnswerText: "4"},
		},
		Players: map[string]*models.SimpleQuizPlayer{
			"p3": {ID: "p3", Nickname: "zoe", Secret: "s3"},
			"p1": {ID: "p1", Nickname: "Beta", Secret: "s1"},
			"p2": {ID: "p2", Nickname: "alpha", Secret: "s2"},
		},
		Answers: []models.SimpleQuizAnswer{
			{ID: "a1", PlayerID: "p1", QuestionIndex: 0, QuestionID: "q1", Text: "one", CreatedAt: now},
			{ID: "a2", PlayerID: "p2", QuestionIndex: 0, QuestionID: "q1", Text: "two", CreatedAt: now},
			{ID: "a3", PlayerID: "p3", QuestionIndex: 0, QuestionID: "q1", Text: "three", CreatedAt: now},
		},
		Grades:   map[string]*models.SimpleQuizGrade{},
		Revealed: map[int]bool{0: true},
	}
	qh.SetSimpleQuiz(s)
	return New(Dependencies{QuizHub: qh}), s
}

func TestSimpleQuizTransitionStateGuards(t *testing.T) {
	for _, status := range []string{
		models.SimpleQuizStateWaitingGrading,
		models.SimpleQuizStateResultsPublished,
		models.SimpleQuizStateCompleted,
	} {
		t.Run("next_"+status, func(t *testing.T) {
			h, _ := seedHandlerSimpleQuiz(status)
			req := httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/next", nil)
			req.SetPathValue("eventID", "e1")
			req.SetPathValue("moduleID", "m1")
			rec := httptest.NewRecorder()
			h.NextSimpleQuizQuestion(rec, req)
			if rec.Code != http.StatusConflict {
				t.Fatalf("NextSimpleQuizQuestion in %s returned %d, want 409", status, rec.Code)
			}
		})
		t.Run("end_"+status, func(t *testing.T) {
			h, _ := seedHandlerSimpleQuiz(status)
			req := httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/end-answering", nil)
			req.SetPathValue("eventID", "e1")
			req.SetPathValue("moduleID", "m1")
			rec := httptest.NewRecorder()
			h.EndSimpleQuizAnswering(rec, req)
			if rec.Code != http.StatusConflict {
				t.Fatalf("EndSimpleQuizAnswering in %s returned %d, want 409", status, rec.Code)
			}
		})
	}

	for _, status := range []string{models.SimpleQuizStateLobby, models.SimpleQuizStateQuestionOpen} {
		t.Run("reveal_"+status, func(t *testing.T) {
			h, _ := seedHandlerSimpleQuiz(status)
			req := httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/reveal", nil)
			req.SetPathValue("eventID", "e1")
			req.SetPathValue("moduleID", "m1")
			rec := httptest.NewRecorder()
			h.RevealSimpleQuizAnswer(rec, req)
			if rec.Code != http.StatusConflict {
				t.Fatalf("RevealSimpleQuizAnswer in %s returned %d, want 409", status, rec.Code)
			}
		})
	}
}

func TestSimpleQuizHostViewSortsPlayersAndIncludesRevealedAnswer(t *testing.T) {
	_, s := seedHandlerSimpleQuiz(models.SimpleQuizStateAnswersLocked)
	state := publicSimpleQuizState(s, "host", "")
	players, ok := state["playerList"].([]map[string]any)
	if !ok || len(players) != 3 {
		t.Fatalf("unexpected playerList: %#v", state["playerList"])
	}
	gotOrder := []string{players[0]["nickname"].(string), players[1]["nickname"].(string), players[2]["nickname"].(string)}
	wantOrder := []string{"alpha", "Beta", "zoe"}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("playerList order = %v, want %v", gotOrder, wantOrder)
		}
	}
	qs := state["questions"].([]map[string]any)
	if qs[0]["correctAnswer"] != "Paris" {
		t.Fatalf("host revealed correctAnswer = %v, want Paris", qs[0]["correctAnswer"])
	}
	if _, leaked := qs[1]["correctAnswer"]; leaked {
		t.Fatalf("unrevealed question leaked correctAnswer: %#v", qs[1])
	}
}

func TestSimpleQuizQueueCursorResilienceAndMetadata(t *testing.T) {
	now := time.Now().UTC()
	s := &models.SimpleQuizSession{
		EventID:  "e1",
		ModuleID: "m1",
		Status:   models.SimpleQuizStateWaitingGrading,
		Questions: []models.SimpleQuizQuestionSnapshot{
			{ID: "q1", Text: "Q1", Points: 50, JudgeNotes: "accept pariz", CorrectAnswerText: "Paris"},
		},
		Answers: []models.SimpleQuizAnswer{
			{ID: "a1", PlayerID: "p1", QuestionIndex: 0, Text: "one", CreatedAt: now},
			{ID: "a2", PlayerID: "p2", QuestionIndex: 0, Text: "two", CreatedAt: now},
			{ID: "a3", PlayerID: "p3", QuestionIndex: 0, Text: "three", CreatedAt: now},
		},
		Grades: map[string]*models.SimpleQuizGrade{
			"a1": {AnswerID: "a1", Verdict: models.SimpleQuizVerdictRight},
		},
		Revealed: map[int]bool{},
	}

	// Cursor "a1" was graded right after page 1 was fetched; page 2 in pending filter must still return a2, a3.
	resp := buildSimpleQuizQueueResponse(s, -1, "pending", "a1", 2)
	rows := resp["answers"].([]simpleQuizQueueRow)
	if len(rows) != 2 || rows[0].AnswerID != "a2" || rows[1].AnswerID != "a3" {
		t.Fatalf("unexpected rows after graded cursor: %+v", rows)
	}
	// Exact match for limit with no remaining items -> nextCursor must be empty.
	if resp["nextCursor"] != "" {
		t.Fatalf("nextCursor = %q, want empty when no more matching items", resp["nextCursor"])
	}

	// With limit=1 and 2 remaining items, nextCursor must point to a2.
	respPage1 := buildSimpleQuizQueueResponse(s, -1, "pending", "", 1)
	if respPage1["nextCursor"] != "a2" {
		t.Fatalf("nextCursor = %q, want a2", respPage1["nextCursor"])
	}

	qs := resp["questions"].([]map[string]any)
	if qs[0]["correctAnswerText"] != "Paris" || qs[0]["judgeNotes"] != "accept pariz" {
		t.Fatalf("unexpected judge question metadata: %+v", qs[0])
	}
}
