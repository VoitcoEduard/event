package hub

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func seedSimpleQuizSession(h *QuizHub, moduleID string, status string, deadline int64) {
	s := &models.SimpleQuizSession{
		ModuleID:     moduleID,
		Status:       status,
		CurrentIndex: 0,
		Questions: []models.SimpleQuizQuestionSnapshot{
			{ID: "q1", Text: "Q1"},
		},
		Players: map[string]*models.SimpleQuizPlayer{
			"p1": {ID: "p1", Nickname: "Alex", Secret: "s1", Connected: true},
		},
		Answers:      []models.SimpleQuizAnswer{},
		Grades:       map[string]*models.SimpleQuizGrade{},
		Revealed:     map[int]bool{},
		DeadlineUnix: deadline,
	}
	h.mu.Lock()
	h.simpleQuizSessions[moduleID] = s
	h.mu.Unlock()
}

func answerCount(h *QuizHub, moduleID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.simpleQuizSessions[moduleID].Answers)
}

// Same answerId submitted concurrently: exactly one stored, rest duplicates.
func TestSubmitSimpleQuizAnswer_ConcurrentSameAnswerID(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h, "m1", models.SimpleQuizStateQuestionOpen, 0)

	const n = 20
	var wg sync.WaitGroup
	codes := make([]string, n)
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, code := h.SubmitSimpleQuizAnswer("m1", "p1", "s1", "a-same", 0, "text", time.Now())
			codes[i], ids[i] = code, id
		}(i)
	}
	wg.Wait()

	if got := answerCount(h, "m1"); got != 1 {
		t.Fatalf("stored answers = %d, want 1", got)
	}
	ok, dup := 0, 0
	for i := 0; i < n; i++ {
		switch codes[i] {
		case SimpleQuizSubmitOK:
			ok++
			if ids[i] != "a-same" {
				t.Fatalf("ok returned id %q, want %q", ids[i], "a-same")
			}
		case SimpleQuizSubmitDuplicate:
			dup++
			if ids[i] != "a-same" {
				t.Fatalf("duplicate returned id %q, want %q", ids[i], "a-same")
			}
		default:
			t.Fatalf("unexpected code %q", codes[i])
		}
	}
	if ok != 1 || dup != n-1 {
		t.Fatalf("ok=%d dup=%d, want ok=1 dup=%d", ok, dup, n-1)
	}
}

// Same player+question with distinct answerIds: exactly one stored.
func TestSubmitSimpleQuizAnswer_ConcurrentSamePlayerQuestion(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h, "m2", models.SimpleQuizStateQuestionOpen, 0)

	const n = 20
	var wg sync.WaitGroup
	codes := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, code := h.SubmitSimpleQuizAnswer("m2", "p1", "s1", fmt.Sprintf("a-%d", i), 0, "text", time.Now())
			codes[i] = code
		}(i)
	}
	wg.Wait()

	if got := answerCount(h, "m2"); got != 1 {
		t.Fatalf("stored answers = %d, want 1", got)
	}
	ok := 0
	for _, c := range codes {
		if c == SimpleQuizSubmitOK {
			ok++
		} else if c != SimpleQuizSubmitDuplicate {
			t.Fatalf("unexpected code %q", c)
		}
	}
	if ok != 1 {
		t.Fatalf("ok=%d, want 1", ok)
	}
}

func TestSubmitSimpleQuizAnswer_Validation(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h, "m3", models.SimpleQuizStateQuestionOpen, 0)

	if _, code := h.SubmitSimpleQuizAnswer("m3", "p1", "wrong", "a1", 0, "t", time.Now()); code != SimpleQuizSubmitUnknownPlayer {
		t.Fatalf("bad secret code = %q", code)
	}
	if _, code := h.SubmitSimpleQuizAnswer("m3", "p1", "s1", "a1", 5, "t", time.Now()); code != SimpleQuizSubmitWrongQuestion {
		t.Fatalf("wrong index code = %q", code)
	}
	if id, code := h.SubmitSimpleQuizAnswer("m3", "p1", "s1", "a1", 0, "t", time.Now()); code != SimpleQuizSubmitOK || id != "a1" {
		t.Fatalf("first submit = %q/%q", id, code)
	}
	if id, code := h.SubmitSimpleQuizAnswer("m3", "p1", "s1", "a1", 0, "t", time.Now()); code != SimpleQuizSubmitDuplicate || id != "a1" {
		t.Fatalf("retry same id = %q/%q", id, code)
	}
	if id, code := h.SubmitSimpleQuizAnswer("m3", "p1", "s1", "a2", 0, "t", time.Now()); code != SimpleQuizSubmitDuplicate || id != "a1" {
		t.Fatalf("second answer same player = %q/%q", id, code)
	}

	h2 := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h2, "m4", models.SimpleQuizStateAnswersLocked, 0)
	if _, code := h2.SubmitSimpleQuizAnswer("m4", "p1", "s1", "a9", 0, "t", time.Now()); code != SimpleQuizSubmitClosed {
		t.Fatalf("locked code = %q", code)
	}

	h3 := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h3, "m5", models.SimpleQuizStateQuestionOpen, time.Now().Add(-time.Minute).Unix())
	if _, code := h3.SubmitSimpleQuizAnswer("m5", "p1", "s1", "a9", 0, "t", time.Now()); code != SimpleQuizSubmitDeadline {
		t.Fatalf("deadline code = %q", code)
	}

	if _, code := h.SubmitSimpleQuizAnswer("nope", "p1", "s1", "a9", 0, "t", time.Now()); code != SimpleQuizSubmitNoSession {
		t.Fatalf("missing session code = %q", code)
	}
}
