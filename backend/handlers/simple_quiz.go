package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/go-expo-sample/backend/hub"
	"github.com/user/go-expo-sample/backend/models"
	"github.com/user/go-expo-sample/backend/postgres"
)

// RequireSimpleQuizModule prevents a module from being accessed through another event's URL.
func (h *Handler) RequireSimpleQuizModule(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.deps.DB == nil {
			http.Error(w, "Storage unavailable", http.StatusServiceUnavailable)
			return
		}
		eventID, moduleID := simpleQuizIDs(r)
		module, err := h.deps.DB.GetModule(r.Context(), eventID, moduleID)
		if errors.Is(err, postgres.ErrNotFound) || (err == nil && module.Type != models.ModuleTypeSimpleQuiz) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "Storage unavailable", http.StatusServiceUnavailable)
			return
		}
		next(w, r)
	}
}

func simpleQuizIDs(r *http.Request) (eventID, moduleID string) {
	// Prefer PathValue (Go 1.22+ patterns), fall back to manual split.
	if v := r.PathValue("eventID"); v != "" {
		eventID = v
	}
	if v := r.PathValue("moduleID"); v != "" {
		moduleID = v
	}
	if eventID != "" && moduleID != "" {
		return eventID, moduleID
	}
	parts := strings.Split(r.URL.Path, "/")
	// /events/{eventID}/modules/{moduleID}/simple-quiz/...
	if len(parts) >= 5 {
		return parts[2], parts[4]
	}
	return "", ""
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (h *Handler) simpleQuizHub() *hub.QuizHub {
	return h.deps.QuizHub
}

// LaunchSimpleQuiz snapshots template questions into a new live session.
func (h *Handler) LaunchSimpleQuiz(w http.ResponseWriter, r *http.Request) {
	if h.simpleQuizHub() == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	eventID, moduleID := simpleQuizIDs(r)
	if eventID == "" || moduleID == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	snaps, allowReveal, _, title, err := h.simpleQuizHub().SimpleQuizTemplate(ctx, moduleID)
	if err != nil {
		http.Error(w, "No simple-text questions in template", http.StatusBadRequest)
		return
	}
	if len(snaps) == 0 {
		http.Error(w, "Add at least one text question", http.StatusBadRequest)
		return
	}
	token := h.simpleQuizHub().EnsureSimpleJudgeToken(ctx, eventID, moduleID)
	if token == "" {
		http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		return
	}
	now := time.Now()
	sess := &models.SimpleQuizSession{
		EventID:      eventID,
		ModuleID:     moduleID,
		Title:        title,
		Status:       models.SimpleQuizStateLobby,
		CurrentIndex: 0,
		Questions:    snaps,
		Players:      make(map[string]*models.SimpleQuizPlayer),
		Answers:      []models.SimpleQuizAnswer{},
		Grades:       make(map[string]*models.SimpleQuizGrade),
		Revealed:     make(map[int]bool),
		AllowReveal:  allowReveal,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.simpleQuizHub().TrySetSimpleQuiz(sess); err != nil {
		http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{
		"session":    publicSimpleQuizState(sess, "host", ""),
		"judgeToken": token,
	})
}

// OpenSimpleQuizQuestion opens the current (or specified) question.
func (h *Handler) OpenSimpleQuizQuestion(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "Launch the quiz first", http.StatusBadRequest)
		return
	}
	var body struct {
		QuestionIndex *int `json:"questionIndex"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	idx := s.CurrentIndex
	if body.QuestionIndex != nil {
		idx = *body.QuestionIndex
	}
	if idx < 0 || idx >= len(s.Questions) {
		http.Error(w, "Invalid question index", http.StatusBadRequest)
		return
	}
	if s.Status != models.SimpleQuizStateLobby &&
		s.Status != models.SimpleQuizStateBetweenQuestions &&
		s.Status != models.SimpleQuizStateAnswersLocked &&
		s.Status != models.SimpleQuizStatePaused {
		http.Error(w, "Cannot open question in state "+s.Status, http.StatusConflict)
		return
	}
	q := s.Questions[idx]
	var deadline int64
	if q.TimeLimit > 0 {
		deadline = time.Now().Add(time.Duration(q.TimeLimit) * time.Second).Unix()
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		sess.CurrentIndex = idx
		sess.Status = models.SimpleQuizStateQuestionOpen
		sess.DeadlineUnix = deadline
		sess.QuestionOpenedAt = time.Now()
		sess.PausedFrom = ""
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	if deadline > 0 {
		hub.BroadcastSimpleQuiz(moduleID)
		// arm timer via exported path: re-schedule by touching hub internals
		// (schedule is unexported; emulate by goroutine here is avoided —
		// call SetSimpleQuiz path already broadcast; use hub method via Update + timer)
		armSimpleQuizTimer(hub, moduleID, deadline)
	}
	writeJSON(w, publicSimpleQuizState(updated, "host", ""))
}

// LockSimpleQuizAnswers closes submissions for the active question.
func (h *Handler) LockSimpleQuizAnswers(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	if s.Status != models.SimpleQuizStateQuestionOpen {
		http.Error(w, "Question is not open", http.StatusConflict)
		return
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		sess.Status = models.SimpleQuizStateAnswersLocked
		sess.DeadlineUnix = 0
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, publicSimpleQuizState(updated, "host", ""))
}

// NextSimpleQuizQuestion advances to between-questions for the next index.
func (h *Handler) NextSimpleQuizQuestion(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	if s.CurrentIndex+1 >= len(s.Questions) {
		http.Error(w, "Last question reached; end answering instead", http.StatusBadRequest)
		return
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		sess.CurrentIndex++
		sess.Status = models.SimpleQuizStateBetweenQuestions
		sess.DeadlineUnix = 0
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, publicSimpleQuizState(updated, "host", ""))
}

// EndSimpleQuizAnswering moves the session into grading (no publish).
func (h *Handler) EndSimpleQuizAnswering(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		sess.Status = models.SimpleQuizStateWaitingGrading
		sess.DeadlineUnix = 0
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, publicSimpleQuizState(updated, "host", ""))
}

// ResumeSimpleQuiz resumes a paused session after restart.
func (h *Handler) ResumeSimpleQuiz(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	if s.Status != models.SimpleQuizStatePaused {
		http.Error(w, "Session is not paused", http.StatusConflict)
		return
	}
	back := s.PausedFrom
	if back == "" {
		back = models.SimpleQuizStateQuestionOpen
	}
	var deadline int64
	if back == models.SimpleQuizStateQuestionOpen && s.CurrentIndex < len(s.Questions) {
		if tl := s.Questions[s.CurrentIndex].TimeLimit; tl > 0 {
			deadline = time.Now().Add(time.Duration(tl) * time.Second).Unix()
		}
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		sess.Status = back
		sess.PausedFrom = ""
		sess.DeadlineUnix = deadline
		sess.QuestionOpenedAt = time.Now()
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	if deadline > 0 {
		armSimpleQuizTimer(hub, moduleID, deadline)
	}
	writeJSON(w, publicSimpleQuizState(updated, "host", ""))
}

// RevealSimpleQuizAnswer reveals the public correct answer if allowed.
func (h *Handler) RevealSimpleQuizAnswer(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	if !s.AllowReveal {
		http.Error(w, "Answer revealing is disabled for this quiz", http.StatusForbidden)
		return
	}
	var body struct {
		QuestionIndex *int `json:"questionIndex"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	idx := s.CurrentIndex
	if body.QuestionIndex != nil {
		idx = *body.QuestionIndex
	}
	if idx < 0 || idx >= len(s.Questions) {
		http.Error(w, "Invalid question index", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(s.Questions[idx].CorrectAnswerText) == "" {
		http.Error(w, "No public answer set for this question", http.StatusBadRequest)
		return
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		if sess.Revealed == nil {
			sess.Revealed = make(map[int]bool)
		}
		sess.Revealed[idx] = true
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, publicSimpleQuizState(updated, "host", ""))
}

// PublishSimpleQuizResults publishes working results (host only, gated).
func (h *Handler) PublishSimpleQuizResults(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	if s.Status != models.SimpleQuizStateWaitingGrading && s.Status != models.SimpleQuizStateAnsweringFinished {
		http.Error(w, "End answering before publishing", http.StatusConflict)
		return
	}
	if !s.GradingComplete() {
		http.Error(w, "Every submitted answer must be marked Right or Wrong", http.StatusConflict)
		return
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		rows := sess.ComputeWorkingResults()
		sess.PublishedVer++
		sess.Published = &models.SimpleQuizPublishedSnapshot{
			Version:     sess.PublishedVer,
			PublishedAt: time.Now(),
			Players:     rows,
		}
		sess.Status = models.SimpleQuizStateResultsPublished
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, map[string]any{
		"session":   publicSimpleQuizState(updated, "host", ""),
		"published": updated.Published,
	})
}

// UpdateSimpleQuizPublished recomputes the published snapshot after corrections.
func (h *Handler) UpdateSimpleQuizPublished(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil || s.Published == nil {
		http.Error(w, "Nothing published yet", http.StatusBadRequest)
		return
	}
	if !s.GradingComplete() {
		http.Error(w, "Every submitted answer must be marked Right or Wrong", http.StatusConflict)
		return
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		rows := sess.ComputeWorkingResults()
		sess.PublishedVer++
		sess.Published = &models.SimpleQuizPublishedSnapshot{
			Version:     sess.PublishedVer,
			PublishedAt: time.Now(),
			Players:     rows,
		}
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, map[string]any{
		"session":   publicSimpleQuizState(updated, "host", ""),
		"published": updated.Published,
	})
}

// CompleteSimpleQuiz marks the session completed.
func (h *Handler) CompleteSimpleQuiz(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	if s.Status != models.SimpleQuizStateResultsPublished {
		http.Error(w, "Publish results before completing", http.StatusConflict)
		return
	}
	updated, updateErr := hub.TryUpdateSimpleQuizExpected(moduleID, s, func(sess *models.SimpleQuizSession) {
		sess.Status = models.SimpleQuizStateCompleted
	})
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, publicSimpleQuizState(updated, "host", ""))
}

// ResetSimpleQuiz clears the session.
func (h *Handler) ResetSimpleQuiz(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	if err := hub.TryDeleteSimpleQuiz(moduleID); err != nil {
		http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]string{"message": "Simple quiz reset"})
}

// JoinSimpleQuiz registers a nickname and returns a private reconnect identity.
func (h *Handler) JoinSimpleQuiz(w http.ResponseWriter, r *http.Request) {
	qh := h.simpleQuizHub()
	if qh == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	var body struct {
		Nickname string `json:"nickname"`
		Secret   string `json:"secret"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	nick := strings.TrimSpace(body.Nickname)
	if nick == "" || len([]rune(nick)) > 40 {
		http.Error(w, "Nickname required (max 40 chars)", http.StatusBadRequest)
		return
	}
	id := "sq_" + randHex(8)
	secret := randHex(16)
	player, reconnected, code := qh.JoinSimpleQuiz(moduleID, nick, body.Secret, id, secret, time.Now())
	switch code {
	case hub.SimpleQuizJoinNoSession:
		http.Error(w, "Quiz is not launched yet", http.StatusNotFound)
		return
	case hub.SimpleQuizJoinClosed:
		http.Error(w, "Joining is closed", http.StatusConflict)
		return
	case hub.SimpleQuizJoinTaken:
		http.Error(w, "Nickname is already taken", http.StatusConflict)
		return
	case hub.SimpleQuizJoinStorageError:
		http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		return
	}
	if reconnected {
		writeJSON(w, map[string]any{"playerId": player.ID, "secret": player.Secret, "nickname": player.Nickname, "reconnected": true})
		return
	}
	writeJSON(w, map[string]any{"playerId": player.ID, "secret": player.Secret, "nickname": player.Nickname})
}

// SubmitSimpleQuizAnswer stores one locked answer per player per question.
func (h *Handler) SubmitSimpleQuizAnswer(w http.ResponseWriter, r *http.Request) {
	qh := h.simpleQuizHub()
	if qh == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	var body struct {
		PlayerID      string `json:"playerId"`
		Secret        string `json:"secret"`
		QuestionIndex int    `json:"questionIndex"`
		Text          string `json:"text"`
		AnswerID      string `json:"answerId"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if body.AnswerID == "" {
		if k := r.Header.Get("Idempotency-Key"); k != "" {
			body.AnswerID = k
		}
	}
	if body.AnswerID == "" {
		http.Error(w, "answerId (or Idempotency-Key) is required", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(body.AnswerID); err != nil {
		http.Error(w, "Invalid answer ID", http.StatusBadRequest)
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		http.Error(w, "Answer must not be empty", http.StatusBadRequest)
		return
	}
	if len([]rune(text)) > models.SimpleQuizMaxAnswerChars {
		http.Error(w, "Answer exceeds 2000 characters", http.StatusBadRequest)
		return
	}
	id, code := qh.SubmitSimpleQuizAnswer(moduleID, body.PlayerID, body.Secret, body.AnswerID, body.QuestionIndex, text, time.Now())
	switch code {
	case hub.SimpleQuizSubmitNoSession:
		http.Error(w, "No session", http.StatusNotFound)
	case hub.SimpleQuizSubmitUnknownPlayer:
		http.Error(w, "Unknown player", http.StatusForbidden)
	case hub.SimpleQuizSubmitDuplicate:
		writeJSON(w, map[string]any{"answerId": id, "duplicate": true})
	case hub.SimpleQuizSubmitClosed:
		http.Error(w, "Submissions are closed", http.StatusConflict)
	case hub.SimpleQuizSubmitWrongQuestion:
		http.Error(w, "Wrong question is active", http.StatusConflict)
	case hub.SimpleQuizSubmitDeadline:
		http.Error(w, "Deadline has passed", http.StatusConflict)
	case hub.SimpleQuizSubmitInvalid:
		http.Error(w, "Invalid question", http.StatusBadRequest)
	case hub.SimpleQuizSubmitStorageError:
		http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
	default:
		writeJSON(w, map[string]any{"answerId": id})
	}
}

// GetSimpleQuizState returns role-filtered state (never leaks notes/grades/secrets).
func (h *Handler) GetSimpleQuizState(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		writeJSON(w, map[string]string{"status": "inactive"})
		return
	}
	view := r.URL.Query().Get("view")
	playerID := r.URL.Query().Get("playerId")
	if view == "" {
		view = "projector"
	}
	writeJSON(w, publicSimpleQuizState(s, view, playerID))
}

// GetSimpleQuizQueue returns the anonymous grading queue (judge token required).
func (h *Handler) GetSimpleQuizQueue(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	token := r.URL.Query().Get("judgeToken")
	if token == "" {
		token = r.Header.Get("X-Judge-Token")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if !hub.ValidateSimpleJudgeToken(ctx, moduleID, token) {
		http.Error(w, "Invalid judge token", http.StatusForbidden)
		return
	}
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	q := r.URL.Query()
	qi, limit, err := simpleQuizQueueOptions(q, len(s.Questions))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	filter := q.Get("filter")
	if filter == "" {
		filter = "pending"
	}
	cursor := q.Get("cursor")
	type row struct {
		AnswerID      string `json:"answerId"`
		QuestionIndex int    `json:"questionIndex"`
		Text          string `json:"text"`
		Verdict       string `json:"verdict"`
		CreatedAt     string `json:"createdAt"`
	}
	rows := []row{}
	progress := []models.SimpleQuizGradingProgress{}
	for i := range s.Questions {
		progress = append(progress, s.ProgressFor(i))
	}
	started := cursor == ""
	for _, a := range s.Answers {
		if qi >= 0 && a.QuestionIndex != qi {
			continue
		}
		g, graded := s.Grades[a.ID]
		verdict := ""
		if graded {
			verdict = g.Verdict
		}
		if filter == "pending" && (graded && verdict != models.SimpleQuizVerdictReviewLater) {
			// pending = ungraded + review_later? Spec: review_later prevents publication,
			// judge can view only pending. Keep review_later in pending.
			if verdict != models.SimpleQuizVerdictReviewLater {
				continue
			}
		}
		// Hide review_later as unresolved for publish gate but show in queue.
		if !graded {
			verdict = "pending"
		}
		if !started {
			if a.ID == cursor {
				started = true
			}
			continue
		}
		rows = append(rows, row{
			AnswerID: a.ID, QuestionIndex: a.QuestionIndex, Text: a.Text,
			Verdict: verdict, CreatedAt: a.CreatedAt.Format(time.RFC3339),
		})
		if len(rows) >= limit {
			break
		}
	}
	nextCursor := ""
	if len(rows) == limit {
		nextCursor = rows[len(rows)-1].AnswerID
	}
	qs := make([]map[string]any, 0, len(s.Questions))
	for i, q := range s.Questions {
		qs = append(qs, map[string]any{
			"index": i, "id": q.ID, "text": q.Text,
			"imageId": q.ImageID, "imageAlt": q.ImageAlt,
			"points": q.Points, "timeLimit": q.TimeLimit,
			"judgeNotes":       q.JudgeNotes,
			"hasCorrectAnswer": strings.TrimSpace(q.CorrectAnswerText) != "",
			"revealed":         s.Revealed[i],
		})
	}
	writeJSON(w, map[string]any{
		"answers": rows, "progress": progress,
		"gradingComplete": s.GradingComplete(),
		"nextCursor":      nextCursor,
		"questions":       qs,
		"total":           len(s.Questions),
		"status":          s.Status,
		"currentIndex":    s.CurrentIndex,
	})
}

// GradeSimpleQuizAnswer marks one answer right/wrong/review_later.
func (h *Handler) GradeSimpleQuizAnswer(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	var body struct {
		AnswerID   string `json:"answerId"`
		Verdict    string `json:"verdict"`
		JudgeToken string `json:"judgeToken"`
		JudgeKey   string `json:"judgeKey"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeRequestError(w, err, http.StatusBadRequest)
		return
	}
	if body.JudgeToken == "" {
		body.JudgeToken = r.Header.Get("X-Judge-Token")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if !hub.ValidateSimpleJudgeToken(ctx, moduleID, body.JudgeToken) {
		http.Error(w, "Invalid judge token", http.StatusForbidden)
		return
	}
	if body.Verdict != models.SimpleQuizVerdictRight &&
		body.Verdict != models.SimpleQuizVerdictWrong &&
		body.Verdict != models.SimpleQuizVerdictReviewLater {
		http.Error(w, "Verdict must be right, wrong or review_later", http.StatusBadRequest)
		return
	}
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	found := false
	for _, a := range s.Answers {
		if a.ID == body.AnswerID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Answer not found", http.StatusNotFound)
		return
	}
	by := body.JudgeKey
	if by == "" {
		by = "judge"
	}
	updated, updateErr := hub.TryApplySimpleGrade(moduleID, body.JudgeToken, by, body.AnswerID, body.Verdict)
	if errors.Is(updateErr, postgres.ErrNotFound) {
		http.Error(w, "Answer not found", http.StatusNotFound)
		return
	}
	if updateErr != nil {
		if errors.Is(updateErr, postgres.ErrConflict) {
			http.Error(w, "Quiz changed; retry", http.StatusConflict)
		} else {
			http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	writeJSON(w, map[string]any{
		"answerId": body.AnswerID, "verdict": body.Verdict,
		"gradingComplete": updated.GradingComplete(),
		"progress":        progressAll(updated),
	})
}

// UndoSimpleQuizGrade reverts the latest grading action for one judge key.
func (h *Handler) UndoSimpleQuizGrade(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	var body struct {
		JudgeToken string `json:"judgeToken"`
		JudgeKey   string `json:"judgeKey"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.JudgeToken == "" {
		body.JudgeToken = r.Header.Get("X-Judge-Token")
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if !hub.ValidateSimpleJudgeToken(ctx, moduleID, body.JudgeToken) {
		http.Error(w, "Invalid judge token", http.StatusForbidden)
		return
	}
	by := body.JudgeKey
	if by == "" {
		by = "judge"
	}
	answerID, previous, updated, err := hub.UndoSimpleGrade(moduleID, body.JudgeToken, by)
	if errors.Is(err, postgres.ErrNotFound) {
		http.Error(w, "Nothing to undo", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Quiz storage unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]any{"undone": answerID, "restored": previous, "gradingComplete": updated.GradingComplete()})
}

// GetSimpleQuizResults returns working (host/judge) and published snapshots.
func (h *Handler) GetSimpleQuizResults(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	_, moduleID := simpleQuizIDs(r)
	s := hub.GetSimpleQuiz(moduleID)
	if s == nil {
		http.Error(w, "No session", http.StatusNotFound)
		return
	}
	view := r.URL.Query().Get("view")
	playerID := r.URL.Query().Get("playerId")
	resp := map[string]any{
		"status":           s.Status,
		"published":        s.Published,
		"publishedVersion": s.PublishedVer,
		"gradingComplete":  s.GradingComplete(),
	}
	// Working results only for host (admin) and judge (token).
	if view == "host" || view == "judge" {
		if view == "judge" {
			token := r.URL.Query().Get("judgeToken")
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if !hub.ValidateSimpleJudgeToken(ctx, moduleID, token) {
				http.Error(w, "Invalid judge token", http.StatusForbidden)
				return
			}
		}
		resp["working"] = s.ComputeWorkingResults()
		resp["progress"] = progressAll(s)
	}
	// Player personal block when published.
	if view == "player" && playerID != "" && s.Published != nil {
		for _, p := range s.Published.Players {
			if p.PlayerID == playerID {
				resp["me"] = p
				break
			}
		}
		// Include own answer texts.
		mine := []models.SimpleQuizAnswer{}
		for _, a := range s.Answers {
			if a.PlayerID == playerID {
				mine = append(mine, a)
			}
		}
		resp["myAnswers"] = mine
	}
	writeJSON(w, resp)
}

// RotateSimpleQuizJudgeToken creates a new secret link (host only).
func (h *Handler) RotateSimpleQuizJudgeToken(w http.ResponseWriter, r *http.Request) {
	hub := h.simpleQuizHub()
	if hub == nil {
		http.Error(w, "Quiz hub not initialized", http.StatusInternalServerError)
		return
	}
	eventID, moduleID := simpleQuizIDs(r)
	if h.deps.DB == nil {
		http.Error(w, "DB unavailable", http.StatusInternalServerError)
		return
	}
	token := hub.GenerateJudgeTokenForModule(r.Context(), eventID, moduleID)
	if token == "" {
		http.Error(w, "Failed to rotate token", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"judgeToken": token})
}

func progressAll(s *models.SimpleQuizSession) []models.SimpleQuizGradingProgress {
	out := make([]models.SimpleQuizGradingProgress, 0, len(s.Questions))
	for i := range s.Questions {
		out = append(out, s.ProgressFor(i))
	}
	return out
}

func simpleQuizQueueOptions(q url.Values, totalQuestions int) (index, limit int, err error) {
	index, limit = -1, 20
	if value := q.Get("questionIndex"); value != "" {
		index, err = strconv.Atoi(value)
		if err != nil || index < -1 || index >= totalQuestions {
			return 0, 0, errors.New("invalid questionIndex")
		}
	}
	if value := q.Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, errors.New("invalid limit")
		}
	}
	return index, limit, nil
}
