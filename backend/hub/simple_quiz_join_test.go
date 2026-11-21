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

func TestJoinSimpleQuiz_DuplicateParticipantRejected(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h, "join-part", models.SimpleQuizStateLobby, 0)

	// First join linked to participant "part-1" succeeds.
	p, reconnected, code := h.JoinSimpleQuizWithParticipant("join-part", "Bob", "", "sq_bob", "sec-bob", time.Now(), "part-1")
	if code != SimpleQuizJoinOK || reconnected || p.ID != "sq_bob" || p.ParticipantID != "part-1" {
		t.Fatalf("first join = %+v/%v/%q", p, reconnected, code)
	}

	// Second join with the same participantID under a different nickname must be rejected.
	if _, _, code := h.JoinSimpleQuizWithParticipant("join-part", "BobClone", "", "sq_clone", "sec-clone", time.Now(), "part-1"); code != SimpleQuizJoinParticipantTaken {
		t.Fatalf("duplicate participant code = %q, want %q", code, SimpleQuizJoinParticipantTaken)
	}
	if got := playerCount(h, "join-part"); got != 2 { // seeded Alex + Bob
		t.Fatalf("players = %d, want 2", got)
	}

	// Legitimate reconnect with secret (with or without participantID) succeeds.
	h.mu.Lock()
	h.simpleQuizSessions["join-part"].Players["sq_bob"].Connected = false
	h.mu.Unlock()
	rp, reconnected, code := h.JoinSimpleQuizWithParticipant("join-part", "bob", "sec-bob", "sq_ignored", "sec-ignored", time.Now(), "part-1")
	if code != SimpleQuizJoinOK || !reconnected || rp.ID != "sq_bob" {
		t.Fatalf("reconnect with participantID = %+v/%v/%q", rp, reconnected, code)
	}
	rp2, reconnected2, code2 := h.JoinSimpleQuizWithParticipant("join-part", "Bob", "sec-bob", "sq_ignored2", "sec-ignored2", time.Now(), "")
	if code2 != SimpleQuizJoinOK || !reconnected2 || rp2.ID != "sq_bob" {
		t.Fatalf("reconnect without participantID = %+v/%v/%q", rp2, reconnected2, code2)
	}

	// Distinct participantID succeeds.
	if _, _, code := h.JoinSimpleQuizWithParticipant("join-part", "Cara", "", "sq_cara", "sec-cara", time.Now(), "part-2"); code != SimpleQuizJoinOK {
		t.Fatalf("distinct participant code = %q, want %q", code, SimpleQuizJoinOK)
	}
	// Multiple unlinked (empty participantID) joins with distinct nicknames still succeed.
	if _, _, code := h.JoinSimpleQuizWithParticipant("join-part", "Anon1", "", "sq_a1", "sec-a1", time.Now(), ""); code != SimpleQuizJoinOK {
		t.Fatalf("anon1 code = %q", code)
	}
	if _, _, code := h.JoinSimpleQuizWithParticipant("join-part", "Anon2", "", "sq_a2", "sec-a2", time.Now(), ""); code != SimpleQuizJoinOK {
		t.Fatalf("anon2 code = %q", code)
	}
	if got := playerCount(h, "join-part"); got != 5 { // Alex(seed) + Bob + Cara + Anon1 + Anon2
		t.Fatalf("players = %d, want 5", got)
	}
}

func TestJoinSimpleQuiz_ConcurrentSameParticipant(t *testing.T) {
	h := NewQuizHub(Dependencies{})
	seedSimpleQuizSession(h, "join-part-race", models.SimpleQuizStateLobby, 0)

	const n = 20
	var wg sync.WaitGroup
	codes := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, code := h.JoinSimpleQuizWithParticipant(
				"join-part-race",
				fmt.Sprintf("Nick-%d", i),
				"",
				fmt.Sprintf("sq_p_%d", i),
				fmt.Sprintf("sec-%d", i),
				time.Now(),
				"shared-participant",
			)
			codes[i] = code
		}(i)
	}
	wg.Wait()

	ok, taken := 0, 0
	for _, c := range codes {
		switch c {
		case SimpleQuizJoinOK:
			ok++
		case SimpleQuizJoinParticipantTaken:
			taken++
		default:
			t.Fatalf("unexpected join code %q", c)
		}
	}
	if ok != 1 || taken != n-1 {
		t.Fatalf("ok=%d taken=%d, want ok=1 taken=%d", ok, taken, n-1)
	}
	if got := playerCount(h, "join-part-race"); got != 2 { // seeded Alex + 1 winner
		t.Fatalf("players = %d, want 2", got)
	}
}
