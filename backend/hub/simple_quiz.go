package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/user/go-expo-sample/backend/postgres"
	"log/slog"
	"sync"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

type simpleQuizUndoEntry struct {
	AnswerID   string    `json:"answerId"`
	Prev       string    `json:"prev"`
	PrevBy     string    `json:"prevBy"`
	PrevAt     time.Time `json:"prevAt"`
	NewVerdict string    `json:"newVerdict"`
	At         time.Time `json:"at"`
	By         string    `json:"by"`
}

// GenerateJudgeToken creates an unguessable token for secret judge links.
func GenerateJudgeToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (h *QuizHub) simpleLock(moduleID string) *sync.Mutex {
	v, _ := h.simpleQuizLocks.LoadOrStore(moduleID, &sync.Mutex{})
	return v.(*sync.Mutex)
}
func copySimple(s *models.SimpleQuizSession) (*models.SimpleQuizSession, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var out models.SimpleQuizSession
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Call only with the per-module lock held. Stored snapshots are immutable.
func (h *QuizHub) readSimpleLocked(ctx context.Context, moduleID string) (*models.SimpleQuizSession, error) {
	h.mu.RLock()
	s := h.simpleQuizSessions[moduleID]
	h.mu.RUnlock()
	if s != nil {
		return s, nil
	}
	if h.db == nil {
		return nil, postgres.ErrNotFound
	}
	restored, err := h.db.ReadSimpleQuiz(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	restored, err = h.db.PauseRestoredSimpleQuiz(ctx, restored)
	if err != nil {
		return nil, err
	}
	h.mu.Lock()
	h.simpleQuizSessions[moduleID] = restored
	h.mu.Unlock()
	return restored, nil
}
func (h *QuizHub) GetSimpleQuiz(moduleID string) *models.SimpleQuizSession {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	defer lock.Unlock()
	s, err := h.readSimpleLocked(ctx, moduleID)
	if err != nil {
		return nil
	}
	out, err := copySimple(s)
	if err != nil {
		return nil
	}
	return out
}

// TrySetSimpleQuiz commits the initial snapshot before making it visible.
func (h *QuizHub) TrySetSimpleQuiz(s *models.SimpleQuizSession) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(s.ModuleID)
	lock.Lock()
	next, err := copySimple(s)
	if err == nil && h.db != nil {
		err = h.db.SaveSimpleQuiz(ctx, nil, next)
	}
	if err == nil {
		h.mu.Lock()
		h.simpleQuizSessions[s.ModuleID] = next
		delete(h.simpleQuizUndo, s.ModuleID)
		h.mu.Unlock()
	}
	lock.Unlock()
	if err == nil {
		h.BroadcastSimpleQuiz(s.ModuleID)
	}
	return err
}
func (h *QuizHub) SetSimpleQuiz(s *models.SimpleQuizSession) { _ = h.TrySetSimpleQuiz(s) }
func (h *QuizHub) tryUpdateSimpleLocked(ctx context.Context, moduleID string, fn func(*models.SimpleQuizSession) error) (*models.SimpleQuizSession, error) {
	old, err := h.readSimpleLocked(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	next, err := copySimple(old)
	if err != nil {
		return nil, err
	}
	if err = fn(next); err != nil {
		return nil, err
	}
	next.UpdatedAt = time.Now().UTC()
	if h.db != nil {
		if err = h.db.SaveSimpleQuiz(ctx, old, next); err != nil {
			return nil, err
		}
	}
	h.mu.Lock()
	h.simpleQuizSessions[moduleID] = next
	h.mu.Unlock()
	return copySimple(next)
}

// TryUpdateSimpleQuizExpected rejects a transition based on an obsolete snapshot.
func (h *QuizHub) TryUpdateSimpleQuizExpected(moduleID string, expected *models.SimpleQuizSession, fn func(*models.SimpleQuizSession)) (*models.SimpleQuizSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	next, err := h.tryUpdateSimpleLocked(ctx, moduleID, func(current *models.SimpleQuizSession) error {
		if expected == nil || !current.CreatedAt.Equal(expected.CreatedAt) || !current.UpdatedAt.Equal(expected.UpdatedAt) {
			return postgres.ErrConflict
		}
		fn(current)
		return nil
	})
	lock.Unlock()
	if err == nil {
		h.BroadcastSimpleQuiz(moduleID)
	}
	return next, err
}

func (h *QuizHub) TryUpdateSimpleQuiz(moduleID string, fn func(*models.SimpleQuizSession)) (*models.SimpleQuizSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(moduleID)
	lock.Lock()
	next, err := h.tryUpdateSimpleLocked(ctx, moduleID, func(s *models.SimpleQuizSession) error { fn(s); return nil })
	lock.Unlock()
	if err == nil {
		h.BroadcastSimpleQuiz(moduleID)
	}
	return next, err
}
func (h *QuizHub) UpdateSimpleQuiz(moduleID string, fn func(*models.SimpleQuizSession)) *models.SimpleQuizSession {
	s, _ := h.TryUpdateSimpleQuiz(moduleID, fn)
	return s
}

// BroadcastSimpleQuiz sends counts-only state to all clients of the module.
func (h *QuizHub) BroadcastSimpleQuiz(moduleID string) {
	s := h.GetSimpleQuiz(moduleID)
	if s == nil {
		return
	}
	answered := 0
	for _, a := range s.Answers {
		if a.QuestionIndex == s.CurrentIndex {
			answered++
		}
	}
	progress := s.ProgressFor(s.CurrentIndex)
	h.BroadcastToSession(moduleID, ServerMessage{
		Type: "simple:state",
		Payload: map[string]interface{}{
			"status":           s.Status,
			"currentIndex":     s.CurrentIndex,
			"total":            len(s.Questions),
			"players":          len(s.Players),
			"answeredCount":    answered,
			"progress":         progress,
			"revealed":         s.Revealed,
			"deadlineUnix":     s.DeadlineUnix,
			"publishedVersion": s.PublishedVer,
			"updatedAt":        s.UpdatedAt,
		},
	})
}

func (h *QuizHub) scheduleSimpleQuizAutoLock(moduleID string, deadlineUnix int64) {
	h.mu.Lock()
	if t, ok := h.simpleQuizTimers[moduleID]; ok && t != nil {
		t.Stop()
	}
	delay := time.Until(time.Unix(deadlineUnix, 0))
	if delay <= 0 {
		h.mu.Unlock()
		go h.autoLockSimpleQuiz(moduleID)
		return
	}
	mID := moduleID
	h.simpleQuizTimers[moduleID] = time.AfterFunc(delay, func() {
		h.autoLockSimpleQuiz(mID)
	})
	h.mu.Unlock()
}

func (h *QuizHub) cancelSimpleQuizTimer(moduleID string) {
	h.mu.Lock()
	if t, ok := h.simpleQuizTimers[moduleID]; ok && t != nil {
		t.Stop()
		delete(h.simpleQuizTimers, moduleID)
	}
	h.mu.Unlock()
}

func (h *QuizHub) autoLockSimpleQuiz(moduleID string) {
	_, err := h.TryUpdateSimpleQuiz(moduleID, func(s *models.SimpleQuizSession) {
		if s.Status == models.SimpleQuizStateQuestionOpen {
			s.Status = models.SimpleQuizStateAnswersLocked
			s.DeadlineUnix = 0
		}
	})
	if err != nil && !errors.Is(err, postgres.ErrNotFound) {
		slog.Error("simple quiz auto-lock failed", "module", moduleID, "error", err)
		time.AfterFunc(time.Second, func() { h.autoLockSimpleQuiz(moduleID) })
	}
}

// SimpleQuizTemplate loads snapshot questions + reveal/token settings.
func (h *QuizHub) SimpleQuizTemplate(ctx context.Context, moduleID string) ([]models.SimpleQuizQuestionSnapshot, bool, string, string, error) {
	if h.eventByModule == nil {
		return nil, false, "", "", errSimpleNoLookup()
	}
	event, err := h.eventByModule(ctx, moduleID)
	if err != nil || event == nil {
		return nil, false, "", "", err
	}
	for _, mod := range event.Modules {
		if mod.ID != moduleID {
			continue
		}
		data, err := json.Marshal(mod.Data)
		if err != nil {
			return nil, false, "", "", err
		}
		var qd models.QuizModuleData
		if err := json.Unmarshal(data, &qd); err != nil {
			return nil, false, "", "", err
		}
		snaps := make([]models.SimpleQuizQuestionSnapshot, 0, len(qd.Questions))
		for _, q := range qd.Questions {
			if q.Type != "" && q.Type != models.QuestionTypeSimpleText {
				continue
			}
			// Require text or image; skip empty placeholders.
			if q.Text == "" && (q.ImageID == nil || *q.ImageID == "") {
				continue
			}
			snaps = append(snaps, models.SnapshotSimpleQuizQuestion(q))
		}
		return snaps, qd.AllowAnswerReveal, qd.JudgeToken, mod.Name, nil
	}
	return nil, false, "", "", errSimpleNotFound()
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errSimpleNoLookup() error { return simpleErr("event lookup unavailable") }
func errSimpleNotFound() error { return simpleErr("module not found in event") }

// ValidateSimpleJudgeToken checks the secret judge link token.
func (h *QuizHub) ValidateSimpleJudgeToken(ctx context.Context, moduleID, token string) bool {
	if token == "" || h.eventByModule == nil {
		return false
	}
	event, err := h.eventByModule(ctx, moduleID)
	if err != nil || event == nil {
		return false
	}
	for _, mod := range event.Modules {
		if mod.ID != moduleID {
			continue
		}
		data, _ := json.Marshal(mod.Data)
		var qd models.QuizModuleData
		_ = json.Unmarshal(data, &qd)
		if qd.JudgeToken == "" {
			return false
		}
		if len(qd.JudgeToken) != len(token) {
			return false
		}
		match := true
		for i := 0; i < len(token); i++ {
			if qd.JudgeToken[i] != token[i] {
				match = false
			}
		}
		return match
	}
	return false
}

// EnsureSimpleJudgeToken generates and persists a token if missing.
func (h *QuizHub) EnsureSimpleJudgeToken(ctx context.Context, eventID, moduleID string) string {
	if h.eventByModule == nil || h.db == nil {
		return ""
	}
	event, err := h.eventByModule(ctx, moduleID)
	if err != nil || event == nil {
		return ""
	}
	for _, mod := range event.Modules {
		if mod.ID != moduleID {
			continue
		}
		data, _ := json.Marshal(mod.Data)
		var qd models.QuizModuleData
		_ = json.Unmarshal(data, &qd)
		if qd.JudgeToken != "" {
			return qd.JudgeToken
		}
		token := GenerateJudgeToken()
		if token == "" {
			return ""
		}
		token, err = h.db.EnsureJudgeToken(ctx, eventID, moduleID, token)
		if err != nil {
			return ""
		}
		return token
	}
	return ""
}

// --- WebSocket joins for Simple Quiz (counts + live updates, no secrets leaked) ---

func (h *QuizHub) processSimpleHostJoin(client *Client, msg ClientMessage) {
	data, _ := msg.Payload.(map[string]interface{})
	moduleID, _ := data["moduleId"].(string)
	if moduleID == "" {
		return
	}
	client.ModuleID = moduleID
	client.IsHost = true
	client.IsProjector = false
	s := h.GetSimpleQuiz(moduleID)
	if s == nil {
		return
	}
	h.sendToClient(client, mustMarshal(ServerMessage{Type: "simple:state", Payload: simpleStatePayload(s)}))
}

func (h *QuizHub) processSimplePlayerJoin(client *Client, msg ClientMessage) {
	data, _ := msg.Payload.(map[string]interface{})
	moduleID, _ := data["moduleId"].(string)
	playerID, _ := data["playerId"].(string)
	if moduleID == "" {
		return
	}
	client.ModuleID = moduleID
	client.IsHost = false
	client.IsProjector = false
	client.PlayerID = playerID
	s := h.GetSimpleQuiz(moduleID)
	if s == nil {
		return
	}
	h.sendToClient(client, mustMarshal(ServerMessage{Type: "simple:state", Payload: simpleStatePayload(s)}))
}

func (h *QuizHub) processSimpleProjectorJoin(client *Client, msg ClientMessage) {
	data, _ := msg.Payload.(map[string]interface{})
	moduleID, _ := data["moduleId"].(string)
	if moduleID == "" {
		return
	}
	client.ModuleID = moduleID
	client.IsProjector = true
	client.IsHost = false
	s := h.GetSimpleQuiz(moduleID)
	if s == nil {
		return
	}
	h.sendToClient(client, mustMarshal(ServerMessage{Type: "simple:state", Payload: simpleStatePayload(s)}))
}

func (h *QuizHub) processSimpleJudgeJoin(client *Client, msg ClientMessage) {
	data, _ := msg.Payload.(map[string]interface{})
	moduleID, _ := data["moduleId"].(string)
	token, _ := data["judgeToken"].(string)
	if moduleID == "" || token == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !h.ValidateSimpleJudgeToken(ctx, moduleID, token) {
		return
	}
	client.ModuleID = moduleID
	client.IsHost = false
	client.IsProjector = false
	s := h.GetSimpleQuiz(moduleID)
	if s == nil {
		return
	}
	h.sendToClient(client, mustMarshal(ServerMessage{Type: "simple:state", Payload: simpleStatePayload(s)}))
}

func simpleStatePayload(s *models.SimpleQuizSession) map[string]interface{} {
	answered := 0
	for _, a := range s.Answers {
		if a.QuestionIndex == s.CurrentIndex {
			answered++
		}
	}
	return map[string]interface{}{
		"status":           s.Status,
		"currentIndex":     s.CurrentIndex,
		"total":            len(s.Questions),
		"players":          len(s.Players),
		"answeredCount":    answered,
		"progress":         s.ProgressFor(s.CurrentIndex),
		"revealed":         s.Revealed,
		"deadlineUnix":     s.DeadlineUnix,
		"publishedVersion": s.PublishedVer,
	}
}
