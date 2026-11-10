package hub

import (
	"errors"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func TestSimpleTimerRejectsAbandonedOpening(t *testing.T) {
	for _, action := range []string{"lock", "untimed", "reset", "replace", "shutdown"} {
		t.Run(action, func(t *testing.T) {
			h := NewQuizHub(Dependencies{})
			now := time.Now().UTC()
			s := &models.SimpleQuizSession{ModuleID: "m", CreatedAt: now, QuestionOpenedAt: now, Status: models.SimpleQuizStateQuestionOpen, DeadlineUnix: now.Add(time.Hour).Unix()}
			h.SetSimpleQuiz(s)
			h.mu.RLock()
			generation := h.simpleQuizTimerGeneration["m"]
			h.mu.RUnlock()
			switch action {
			case "lock":
				h.UpdateSimpleQuiz("m", func(s *models.SimpleQuizSession) { s.Status = models.SimpleQuizStateAnswersLocked; s.DeadlineUnix = 0 })
			case "untimed":
				h.UpdateSimpleQuiz("m", func(s *models.SimpleQuizSession) {
					s.CurrentIndex++
					s.QuestionOpenedAt = now.Add(time.Second)
					s.DeadlineUnix = 0
				})
			case "reset":
				h.DeleteSimpleQuiz("m")
			case "replace":
				h.SetSimpleQuiz(&models.SimpleQuizSession{ModuleID: "m", CreatedAt: now.Add(time.Second), Status: models.SimpleQuizStateQuestionOpen})
			case "shutdown":
				h.Shutdown()
			}
			before := h.GetSimpleQuiz("m")
			h.autoLockSimpleQuiz("m", generation, s.CreatedAt, s.QuestionOpenedAt, s.CurrentIndex, s.DeadlineUnix)
			after := h.GetSimpleQuiz("m")
			if (before == nil) != (after == nil) || (before != nil && (before.Status != after.Status || before.CurrentIndex != after.CurrentIndex)) {
				t.Fatalf("old callback changed session: before=%+v after=%+v", before, after)
			}
			h.Shutdown()
		})
	}
}

func TestJudgeUndoRejectsStaleChangesAndPreservesHistory(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	h.SetSimpleQuiz(&models.SimpleQuizSession{ModuleID: "m", Answers: []models.SimpleQuizAnswer{{ID: "a"}, {ID: "b"}}, Grades: map[string]*models.SimpleQuizGrade{}})
	grade := func(judge, answer, verdict string) {
		t.Helper()
		if _, err := h.TryApplySimpleGrade("m", "token", judge, answer, verdict); err != nil {
			t.Fatal(err)
		}
	}
	grade("one", "a", "right")
	grade("two", "b", "wrong")
	grade("one", "a", "wrong")
	if _, _, _, err := h.UndoSimpleGrade("m", "token", "one"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := h.UndoSimpleGrade("m", "token", "one"); err != nil {
		t.Fatal(err)
	}
	g := h.GetSimpleQuiz("m").Grades["a"]
	if g.Verdict != "" || len(g.History) != 4 {
		t.Fatalf("grade after chained undo: %+v", g)
	}
	grade("one", "a", "right")
	grade("two", "a", "wrong")
	if _, _, _, err := h.UndoSimpleGrade("m", "token", "one"); !errors.Is(err, ErrStaleSimpleGrade) {
		t.Fatalf("stale undo: %v", err)
	}
	if got := h.GetSimpleQuiz("m").Grades["a"].Verdict; got != "wrong" {
		t.Fatalf("stale undo changed verdict to %q", got)
	}
	if _, _, _, err := h.UndoSimpleGrade("m", "token", "one"); err == nil {
		t.Fatal("stale entry was not discarded")
	}
}

func TestSimplePlayerSubscriptionRequiresSecret(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h, "m", models.SimpleQuizStateLobby, 0)
	for _, secret := range []string{"", "wrong", "s1"} {
		c := &Client{send: make(chan []byte, 2)}
		h.processSimplePlayerJoin(c, ClientMessage{Payload: map[string]interface{}{"moduleId": "m", "playerId": "p1", "secret": secret}})
		if (c.ModuleID == "m") != (secret == "s1") {
			t.Fatalf("secret %q: module = %q", secret, c.ModuleID)
		}
	}
}
