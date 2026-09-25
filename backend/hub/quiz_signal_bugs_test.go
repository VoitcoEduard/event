// Package hub — bug reproduction tests for quiz mode.
//
// Every test here asserts the CURRENT (buggy) behaviour on purpose, so the
// suite stays green. When the corresponding bug is fixed, the assertion is
// marked "// BUG:" must be flipped to the correct expectation.
package hub

import (
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func newSignalHub(t *testing.T) (*QuizHub, *Client) {
	t.Helper()
	h := NewQuizHub(Dependencies{})
	h.CreateSession("m1", "e1")
	h.mu.Lock()
	s := h.sessions["m1"]
	s.State = "question-active"
	s.CurrentIndex = 0
	s.Questions = []models.QuizQuestion{{ID: "q0", Type: "classic", TimeLimit: 20, CorrectIndex: 1}}
	s.QuestionWindows = []models.QuestionWindow{{
		QuestionIndex: 0,
		StartTime:     time.Now().Unix(),
		EndTime:       time.Now().Unix() + 20,
		GraceEndTime:  time.Now().Unix() + 22,
		State:         "active",
	}}
	s.Players["p1"] = &models.Player{ID: "p1", Nickname: "P"}
	h.mu.Unlock()
	return h, &Client{ModuleID: "m1", PlayerID: "p1"}
}

func send(h *QuizHub, c *Client, payload string) {
	h.ProcessMessage(c, []byte(`{"type":"player:signal","payload":`+payload+`}`))
}

// Bug 1: the score is taken verbatim from the client.
func TestReproClientSuppliedScoreIsTrusted(t *testing.T) {
	h, c := newSignalHub(t)
	// Wrong answer (correctIndex is 1), but claims 999999 points.
	send(h, c, `{"questionIndex":0,"answerIndex":0,"scoreEarned":999999,"timeSpent":1000}`)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if got := h.sessions["m1"].Players["p1"].Score; got != 999999 {
		t.Fatalf("server accepted client-declared score: %d", got)
	}
}

// Bug 2 (FIXED A2): repeated signals for the same question are ignored —
// only the first answer counts.
func TestReproDuplicateSignalsAccumulateScore(t *testing.T) {
	h, c := newSignalHub(t)
	for i := 0; i < 3; i++ {
		send(h, c, `{"questionIndex":0,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if got := h.sessions["m1"].Players["p1"].Score; got != 1000 {
		t.Fatalf("score after 3 duplicate signals = %d, want 1000 (first answer only)", got)
	}
	if len(h.sessions["m1"].PlayerSignals) != 1 {
		t.Fatalf("duplicate signals recorded = %d, want 1", len(h.sessions["m1"].PlayerSignals))
	}
}

// Bug 3: grace-period answers are still awarded points (AcceptanceResult.Scored is ignored).
func TestReproGracePeriodStillScores(t *testing.T) {
	h, c := newSignalHub(t)
	h.mu.Lock()
	h.sessions["m1"].QuestionWindows[0].EndTime = time.Now().Unix() - 1
	h.sessions["m1"].QuestionWindows[0].GraceEndTime = time.Now().Unix() + 60
	h.mu.Unlock()
	send(h, c, `{"questionIndex":0,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
	h.mu.RLock()
	defer h.mu.RUnlock()
	sig := h.sessions["m1"].PlayerSignals[0]
	if sig.Status != "grace_period" {
		t.Fatalf("status = %q", sig.Status)
	}
	// BUG: AcceptanceResult.Scored is false for the grace period, but the hub
	// only checks `status != "too_late"`, so late answers are still paid.
	// When fixed: this must become `got != 0`.
	if got := h.sessions["m1"].Players["p1"].Score; got != 1000 {
		t.Fatalf("grace-period score = %d, expected the buggy 1000", got)
	}
}

// Bug 4: a signal for a question that has no window (a future question) is
// accepted with the default "on_time" status and scored.
func TestReproFutureQuestionSignalScores(t *testing.T) {
	h, c := newSignalHub(t)
	send(h, c, `{"questionIndex":42,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if got := h.sessions["m1"].Players["p1"].Score; got != 1000 {
		t.Fatalf("answer for a question that never started scored: %d", got)
	}
}

// Bug 5: negative questionIndex panics (index out of range) and takes the
// whole server down, because nothing recovers panics in the read loop.
func TestReproNegativeQuestionIndexPanics(t *testing.T) {
	h, c := newSignalHub(t)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for questionIndex = -1")
		}
	}()
	send(h, c, `{"questionIndex":-1,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
}

// Bug 6: host:next_question past the last question panics on
// session.Questions[session.CurrentIndex].
func TestReproNextQuestionPastEndPanics(t *testing.T) {
	h, c := newSignalHub(t)
	c.IsHost = true
	h.mu.Lock()
	h.sessions["m1"].CurrentIndex = len(h.sessions["m1"].Questions) // == len
	h.mu.Unlock()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when advancing past the last question")
		}
	}()
	h.processHostNextQuestion(c, ClientMessage{})
}

// Bug 7: too-late answers are still recorded in PlayerSignals, which is what
// the host histogram and the persisted results are built from.
func TestReproTooLateSignalStillRecorded(t *testing.T) {
	h, c := newSignalHub(t)
	h.mu.Lock()
	h.sessions["m1"].QuestionWindows[0].EndTime = time.Now().Unix() - 100
	h.sessions["m1"].QuestionWindows[0].GraceEndTime = time.Now().Unix() - 50
	h.mu.Unlock()
	send(h, c, `{"questionIndex":0,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.sessions["m1"].PlayerSignals) != 1 {
		t.Fatal("expected the rejected signal to still be appended (this documents the bug)")
	}
	if got := h.sessions["m1"].Players["p1"].Score; got != 0 {
		t.Fatalf("too_late signal scored: %d", got)
	}
}
