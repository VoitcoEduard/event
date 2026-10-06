package hub

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

func playerCount(h *QuizHub, moduleID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.simpleQuizSessions[moduleID].Players)
}

// Concurrent joins with the same nickname (mixed case): exactly one wins.
func TestJoinSimpleQuiz_ConcurrentSameNickname(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h, "join-race", models.SimpleQuizStateLobby, 0)

	const n = 30
	var wg sync.WaitGroup
	codes := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nick := "Racer"
			if i%2 == 1 {
				nick = "rAcEr"
			}
			_, _, code := h.JoinSimpleQuiz("join-race", nick, "", fmt.Sprintf("sq_race_%d", i), fmt.Sprintf("sec-%d", i), time.Now())
			codes[i] = code
		}(i)
	}
	wg.Wait()

	ok, taken := 0, 0
	for _, c := range codes {
		switch c {
		case SimpleQuizJoinOK:
			ok++
		case SimpleQuizJoinTaken:
			taken++
		default:
			t.Fatalf("unexpected join code %q", c)
		}
	}
	if ok != 1 || taken != n-1 {
		t.Fatalf("ok=%d taken=%d, want ok=1 taken=%d", ok, taken, n-1)
	}
	// 1 seeded player + 1 winner.
	if got := playerCount(h, "join-race"); got != 2 {
		t.Fatalf("players = %d, want 2", got)
	}
}

func TestJoinSimpleQuiz_Validation(t *testing.T) {
	h := NewQuizHub(Dependencies{})

	if _, _, code := h.JoinSimpleQuiz("missing", "Bob", "", "sq_1", "s", time.Now()); code != SimpleQuizJoinNoSession {
		t.Fatalf("missing session code = %q", code)
	}

	seedSimpleQuizSession(h, "join-closed", models.SimpleQuizStateResultsPublished, 0)
	if _, _, code := h.JoinSimpleQuiz("join-closed", "Bob", "", "sq_1", "s", time.Now()); code != SimpleQuizJoinClosed {
		t.Fatalf("closed code = %q", code)
	}

	seedSimpleQuizSession(h, "join-ok", models.SimpleQuizStateLobby, 0)
	p, reconnected, code := h.JoinSimpleQuiz("join-ok", "Bob", "", "sq_bob", "sec-bob", time.Now())
	if code != SimpleQuizJoinOK || reconnected || p.ID != "sq_bob" || p.Nickname != "Bob" {
		t.Fatalf("join = %+v/%v/%q", p, reconnected, code)
	}
	// Case-insensitive duplicate.
	if _, _, code := h.JoinSimpleQuiz("join-ok", "bOB", "", "sq_bob2", "sec-x", time.Now()); code != SimpleQuizJoinTaken {
		t.Fatalf("duplicate code = %q", code)
	}
	// Distinct nickname succeeds.
	if _, _, code := h.JoinSimpleQuiz("join-ok", "Cara", "", "sq_cara", "sec-cara", time.Now()); code != SimpleQuizJoinOK {
		t.Fatalf("second nickname code = %q", code)
	}
	// Reconnect with secret flips Connected and returns same identity.
	h.mu.Lock()
	h.simpleQuizSessions["join-ok"].Players["sq_bob"].Connected = false
	h.mu.Unlock()
	rp, reconnected, code := h.JoinSimpleQuiz("join-ok", "BOB", "sec-bob", "sq_new", "sec-new", time.Now())
	if code != SimpleQuizJoinOK || !reconnected || rp.ID != "sq_bob" {
		t.Fatalf("reconnect = %+v/%v/%q", rp, reconnected, code)
	}
	if got := playerCount(h, "join-ok"); got != 3 { // Alex(seed) + Bob + Cara, reconnect adds none
		t.Fatalf("players = %d, want 3", got)
	}
}
