package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestSimpleQuizPublicStateNeverLeaksUpcomingOrLobbyQuestions(t *testing.T) {
	hiddenStates := []string{
		models.SimpleQuizStateLobby,
		models.SimpleQuizStateBetweenQuestions,
		models.SimpleQuizStatePaused,
		models.SimpleQuizStateWaitingGrading,
		models.SimpleQuizStateAnsweringFinished,
		models.SimpleQuizStateResultsPublished,
		models.SimpleQuizStateCompleted,
	}
	for _, status := range hiddenStates {
		for _, viewQuery := range []string{"", "projector", "player"} {
			t.Run(status+"_"+viewQuery, func(t *testing.T) {
				h, _ := seedHandlerSimpleQuiz(status)
				url := "/events/e1/modules/m1/simple-quiz/state"
				if viewQuery != "" {
					url += "?view=" + viewQuery
				}
				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.SetPathValue("eventID", "e1")
				req.SetPathValue("moduleID", "m1")
				rec := httptest.NewRecorder()
				h.GetSimpleQuizState(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200", rec.Code)
				}
				body := rec.Body.String()
				if strings.Contains(body, `"Q1"`) || strings.Contains(body, `"Q2"`) || strings.Contains(body, "Accept Pariz") {
					t.Fatalf("public state in %s (view=%q) leaked question content: %s", status, viewQuery, body)
				}
				var got map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("decode json: %v", err)
				}
				if _, hasQuestions := got["questions"]; hasQuestions {
					t.Fatalf("public state in %s (view=%q) should not include questions array: %s", status, viewQuery, body)
				}
				if got["question"] != nil {
					t.Fatalf("public state in %s (view=%q) question = %#v, want nil", status, viewQuery, got["question"])
				}
			})
		}
	}

	activeStates := []string{
		models.SimpleQuizStateQuestionOpen,
		models.SimpleQuizStateAnswersLocked,
	}
	for _, status := range activeStates {
		for _, viewQuery := range []string{"", "projector", "player"} {
			t.Run("active_"+status+"_"+viewQuery, func(t *testing.T) {
				h, _ := seedHandlerSimpleQuiz(status)
				url := "/events/e1/modules/m1/simple-quiz/state"
				if viewQuery != "" {
					url += "?view=" + viewQuery
				}
				req := httptest.NewRequest(http.MethodGet, url, nil)
				req.SetPathValue("eventID", "e1")
				req.SetPathValue("moduleID", "m1")
				rec := httptest.NewRecorder()
				h.GetSimpleQuizState(rec, req)
				if rec.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200", rec.Code)
				}
				body := rec.Body.String()
				if strings.Contains(body, `"Q2"`) || strings.Contains(body, "Accept Pariz") {
					t.Fatalf("public state in %s (view=%q) leaked future question or notes: %s", status, viewQuery, body)
				}
				var got map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("decode json: %v", err)
				}
				if _, hasQuestions := got["questions"]; hasQuestions {
					t.Fatalf("public state in %s (view=%q) should not include questions array", status, viewQuery)
				}
				q, ok := got["question"].(map[string]any)
				if !ok || q["text"] != "Q1" {
					t.Fatalf("public state in %s (view=%q) question = %#v, want Q1", status, viewQuery, got["question"])
				}
			})
		}
	}
}

func TestSimpleQuizResultsIncludesOnlyRevealedAnswers(t *testing.T) {
	h, _ := seedHandlerSimpleQuiz(models.SimpleQuizStateResultsPublished)
	req := httptest.NewRequest(http.MethodGet, "/events/e1/modules/m1/simple-quiz/results?view=player&playerId=p1", nil)
	req.Header.Set("X-Player-Secret", "s1")
	req.SetPathValue("eventID", "e1")
	req.SetPathValue("moduleID", "m1")
	rec := httptest.NewRecorder()
	h.GetSimpleQuizResults(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got struct {
		RevealedAnswers []struct {
			Index int    `json:"index"`
			Text  string `json:"text"`
		} `json:"revealedAnswers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if len(got.RevealedAnswers) != 1 || got.RevealedAnswers[0].Index != 0 || got.RevealedAnswers[0].Text != "Paris" {
		t.Fatalf("revealedAnswers = %+v, want [{Index:0 Text:Paris}]", got.RevealedAnswers)
	}
}

func TestJoinSimpleQuizRejectsDuplicateParticipantAndPreservesReconnect(t *testing.T) {
	h, _ := seedHandlerSimpleQuiz(models.SimpleQuizStateLobby)
	h.checkParticipantProof = func(_ context.Context, eventID, participantID, secret string) (bool, bool, error) {
		return true, eventID == "e1" && secret == "proof-"+participantID, nil
	}

	doJoin := func(payload string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/join", strings.NewReader(payload))
		req.SetPathValue("eventID", "e1")
		req.SetPathValue("moduleID", "m1")
		rec := httptest.NewRecorder()
		h.JoinSimpleQuiz(rec, req)
		return rec
	}

	// First join with valid participant proof succeeds.
	rec1 := doJoin(`{"nickname":"Alice","participantId":"part-1","participantSecret":"proof-part-1"}`)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first join status = %d body=%q, want 200", rec1.Code, rec1.Body.String())
	}
	var joined struct {
		PlayerID string `json:"playerId"`
		Secret   string `json:"secret"`
		Nickname string `json:"nickname"`
	}
	if err := json.Unmarshal(rec1.Body.Bytes(), &joined); err != nil {
		t.Fatalf("decode first join: %v", err)
	}
	if joined.PlayerID == "" || joined.Secret == "" {
		t.Fatalf("missing player identity: %+v", joined)
	}

	// Second join with same participantId and a different nickname must return 409 Conflict.
	recDup := doJoin(`{"nickname":"AliceClone","participantId":"part-1","participantSecret":"proof-part-1"}`)
	if recDup.Code != http.StatusConflict {
		t.Fatalf("duplicate participant join status = %d body=%q, want 409", recDup.Code, recDup.Body.String())
	}

	// Reconnect with original secret + nickname succeeds (with and without participantId).
	recRecon1 := doJoin(`{"nickname":"alice","secret":"` + joined.Secret + `"}`)
	if recRecon1.Code != http.StatusOK {
		t.Fatalf("reconnect without participantId status = %d body=%q, want 200", recRecon1.Code, recRecon1.Body.String())
	}
	var recon1 struct {
		PlayerID    string `json:"playerId"`
		Reconnected bool   `json:"reconnected"`
	}
	if err := json.Unmarshal(recRecon1.Body.Bytes(), &recon1); err != nil {
		t.Fatalf("decode recon1: %v", err)
	}
	if !recon1.Reconnected || recon1.PlayerID != joined.PlayerID {
		t.Fatalf("recon1 = %+v, want playerId=%q reconnected=true", recon1, joined.PlayerID)
	}

	recRecon2 := doJoin(`{"nickname":"Alice","secret":"` + joined.Secret + `","participantId":"part-1","participantSecret":"proof-part-1"}`)
	if recRecon2.Code != http.StatusOK {
		t.Fatalf("reconnect with participantId status = %d body=%q, want 200", recRecon2.Code, recRecon2.Body.String())
	}
	var recon2 struct {
		PlayerID    string `json:"playerId"`
		Reconnected bool   `json:"reconnected"`
	}
	if err := json.Unmarshal(recRecon2.Body.Bytes(), &recon2); err != nil {
		t.Fatalf("decode recon2: %v", err)
	}
	if !recon2.Reconnected || recon2.PlayerID != joined.PlayerID {
		t.Fatalf("recon2 = %+v, want playerId=%q reconnected=true", recon2, joined.PlayerID)
	}

	// Different participant joins cleanly.
	recOther := doJoin(`{"nickname":"Charlie","participantId":"part-2","participantSecret":"proof-part-2"}`)
	if recOther.Code != http.StatusOK {
		t.Fatalf("distinct participant join status = %d body=%q, want 200", recOther.Code, recOther.Body.String())
	}

	// Ensure only 2 new players (Alice + Charlie) were added to the 3 seeded players.
	sess := h.deps.QuizHub.GetSimpleQuiz("m1")
	if len(sess.Players) != 5 {
		t.Fatalf("player count = %d, want 5", len(sess.Players))
	}
}

func TestSimpleQuizBodyLimitsAndDecodeErrors(t *testing.T) {
	oversized := `{"pad":"` + strings.Repeat("x", 4<<20) + `"}`
	oversizedSpaces := strings.Repeat(" ", (4<<20)+16)

	t.Run("open", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			body       string
			nilBody    bool
			wantStatus int
			wantIndex  int
		}{
			{"oversized_json", oversized, false, http.StatusRequestEntityTooLarge, 0},
			{"oversized_whitespace", oversizedSpaces, false, http.StatusRequestEntityTooLarge, 0},
			{"malformed_json", `{bad`, false, http.StatusBadRequest, 0},
			{"wrong_field_type", `{"questionIndex":"one"}`, false, http.StatusBadRequest, 0},
			{"trailing_json", `{"questionIndex":0}{"questionIndex":1}`, false, http.StatusBadRequest, 0},
			{"nil_body", "", true, http.StatusOK, 0},
			{"empty_body", "", false, http.StatusOK, 0},
			{"empty_object", `{}`, false, http.StatusOK, 0},
			{"explicit_index", `{"questionIndex":1}`, false, http.StatusOK, 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				h, _ := seedHandlerSimpleQuiz(models.SimpleQuizStateLobby)
				var req *http.Request
				if tc.nilBody {
					req = httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/open", nil)
				} else {
					req = httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/open", strings.NewReader(tc.body))
				}
				req.SetPathValue("eventID", "e1")
				req.SetPathValue("moduleID", "m1")
				rec := httptest.NewRecorder()
				h.OpenSimpleQuizQuestion(rec, req)
				if rec.Code != tc.wantStatus {
					t.Fatalf("status = %d, want %d (body=%q)", rec.Code, tc.wantStatus, rec.Body.String())
				}
				if tc.wantStatus == http.StatusOK {
					sess := h.deps.QuizHub.GetSimpleQuiz("m1")
					if sess.CurrentIndex != tc.wantIndex || sess.Status != models.SimpleQuizStateQuestionOpen {
						t.Fatalf("session = status=%s idx=%d, want question_open idx=%d", sess.Status, sess.CurrentIndex, tc.wantIndex)
					}
				}
			})
		}
	})

	t.Run("reveal", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			body       string
			nilBody    bool
			wantStatus int
			wantIdx    int
		}{
			{"oversized_json", oversized, false, http.StatusRequestEntityTooLarge, 0},
			{"oversized_whitespace", oversizedSpaces, false, http.StatusRequestEntityTooLarge, 0},
			{"malformed_json", `{bad`, false, http.StatusBadRequest, 0},
			{"wrong_field_type", `{"questionIndex":"one"}`, false, http.StatusBadRequest, 0},
			{"trailing_json", `{"questionIndex":0}{"questionIndex":1}`, false, http.StatusBadRequest, 0},
			{"nil_body", "", true, http.StatusOK, 0},
			{"empty_body", "", false, http.StatusOK, 0},
			{"explicit_index", `{"questionIndex":1}`, false, http.StatusOK, 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				h, _ := seedHandlerSimpleQuiz(models.SimpleQuizStateAnswersLocked)
				var req *http.Request
				if tc.nilBody {
					req = httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/reveal", nil)
				} else {
					req = httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/reveal", strings.NewReader(tc.body))
				}
				req.SetPathValue("eventID", "e1")
				req.SetPathValue("moduleID", "m1")
				rec := httptest.NewRecorder()
				h.RevealSimpleQuizAnswer(rec, req)
				if rec.Code != tc.wantStatus {
					t.Fatalf("status = %d, want %d (body=%q)", rec.Code, tc.wantStatus, rec.Body.String())
				}
				if tc.wantStatus == http.StatusOK {
					sess := h.deps.QuizHub.GetSimpleQuiz("m1")
					if !sess.Revealed[tc.wantIdx] {
						t.Fatalf("question %d not marked revealed: %+v", tc.wantIdx, sess.Revealed)
					}
				}
			})
		}
	})

	t.Run("undo", func(t *testing.T) {
		for _, tc := range []struct {
			name       string
			body       string
			wantStatus int
		}{
			{"oversized_json", oversized, http.StatusRequestEntityTooLarge},
			{"malformed_json", `{bad`, http.StatusBadRequest},
			{"wrong_field_type", `{"judgeKey":123}`, http.StatusBadRequest},
			{"trailing_json", `{"judgeKey":"j1"}{"judgeKey":"j2"}`, http.StatusBadRequest},
			{"empty_body", ``, http.StatusBadRequest},
		} {
			t.Run(tc.name, func(t *testing.T) {
				h, _ := seedHandlerSimpleQuiz(models.SimpleQuizStateWaitingGrading)
				req := httptest.NewRequest(http.MethodPost, "/events/e1/modules/m1/simple-quiz/undo", strings.NewReader(tc.body))
				req.SetPathValue("eventID", "e1")
				req.SetPathValue("moduleID", "m1")
				rec := httptest.NewRecorder()
				h.UndoSimpleQuizGrade(rec, req)
				if rec.Code != tc.wantStatus {
					t.Fatalf("status = %d, want %d (body=%q)", rec.Code, tc.wantStatus, rec.Body.String())
				}
			})
		}
	})
}
