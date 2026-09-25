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

// Bug 3 (FIXED A3): grace-period answers count for statistics but earn no
// points — AcceptanceResult.Scored is honoured.
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
	if got := h.sessions["m1"].Players["p1"].Score; got != 0 {
		t.Fatalf("grace-period signal scored: %d, want 0", got)
	}
	if sig.ScoreEarned != 0 {
		t.Fatalf("grace-period history ScoreEarned = %d, want 0", sig.ScoreEarned)
	}
	if got := h.sessions["m1"].AnswerCounts[1]; got != 1 {
		t.Fatalf("grace-period answer missing from stats: counts = %v", h.sessions["m1"].AnswerCounts)
	}
}

// Bug 4 (FIXED A4): a signal for a question with no window (e.g. a future
// question) is rejected — it scores nothing and is not recorded.
func TestReproFutureQuestionSignalScores(t *testing.T) {
	h, c := newSignalHub(t)
	send(h, c, `{"questionIndex":42,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if got := h.sessions["m1"].Players["p1"].Score; got != 0 {
		t.Fatalf("answer for a question that never started scored: %d, want 0", got)
	}
	if len(h.sessions["m1"].PlayerSignals) != 0 {
		t.Fatalf("invalid-question signal recorded = %d, want 0", len(h.sessions["m1"].PlayerSignals))
	}
}

// A4 also covers stale answers: once the host advances, the previous
// question no longer accepts signals.
func TestStaleQuestionSignalRejected(t *testing.T) {
	h, c := newSignalHub(t)
	h.mu.Lock()
	h.sessions["m1"].CurrentIndex = 1
	h.sessions["m1"].QuestionWindows = append(h.sessions["m1"].QuestionWindows, models.QuestionWindow{
		QuestionIndex: 1,
		StartTime:     time.Now().Unix(),
		EndTime:       time.Now().Unix() + 20,
		GraceEndTime:  time.Now().Unix() + 22,
		State:         "active",
	})
	h.mu.Unlock()
	send(h, c, `{"questionIndex":0,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if got := h.sessions["m1"].Players["p1"].Score; got != 0 {
		t.Fatalf("stale-question signal scored: %d, want 0", got)
	}
	if len(h.sessions["m1"].PlayerSignals) != 0 {
		t.Fatalf("stale-question signal recorded = %d, want 0", len(h.sessions["m1"].PlayerSignals))
	}
}
// Bug 5 (FIXED as a side effect of A4): a negative questionIndex is rejected
// instead of panicking the whole server on out-of-range indexing.
func TestReproNegativeQuestionIndexPanics(t *testing.T) {
	h, c := newSignalHub(t)
	send(h, c, `{"questionIndex":-1,"answerIndex":1,"scoreEarned":1000,"timeSpent":1000}`)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if got := h.sessions["m1"].Players["p1"].Score; got != 0 {
		t.Fatalf("negative questionIndex scored: %d, want 0", got)
	}
	if len(h.sessions["m1"].PlayerSignals) != 0 {
		t.Fatalf("negative questionIndex recorded = %d, want 0", len(h.sessions["m1"].PlayerSignals))
	}
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
