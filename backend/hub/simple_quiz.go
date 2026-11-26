package hub

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/user/go-expo-sample/backend/postgres"
	"hash/fnv"
	"log/slog"
	"sync"
	"time"

	"github.com/user/go-expo-sample/backend/models"
)

type simpleQuizUndoEntry struct {
	AnswerID       string    `json:"answerId"`
	Prev           string    `json:"prev"`
	PrevBy         string    `json:"prevBy"`
	PrevAt         time.Time `json:"prevAt"`
	PrevMutationID string    `json:"prevMutationId"`
	MutationID     string    `json:"mutationId"`
	NewVerdict     string    `json:"newVerdict"`
	At             time.Time `json:"at"`
	By             string    `json:"by"`
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
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(moduleID))
	return &h.simpleQuizLocks[hash.Sum32()%uint32(len(h.simpleQuizLocks))]
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
	return h.trySetSimpleQuiz(s, false)
}

// TryCreateSimpleQuiz starts a new session only if none currently exists.
func (h *QuizHub) TryCreateSimpleQuiz(s *models.SimpleQuizSession) error {
	return h.trySetSimpleQuiz(s, true)
}

func (h *QuizHub) trySetSimpleQuiz(s *models.SimpleQuizSession, requireAbsent bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock := h.simpleLock(s.ModuleID)
	lock.Lock()
	if requireAbsent {
		if _, existingErr := h.readSimpleLocked(ctx, s.ModuleID); existingErr == nil {
			lock.Unlock()
			return postgres.ErrConflict
		} else if !errors.Is(existingErr, postgres.ErrNotFound) {
			lock.Unlock()
			return existingErr
		}
	}
	next, err := copySimple(s)
	if err == nil && h.db != nil {
		err = h.db.SaveSimpleQuiz(ctx, nil, next)
	}
	if err == nil {
		h.mu.Lock()
		h.simpleQuizSessions[s.ModuleID] = next
		delete(h.simpleQuizUndo, s.ModuleID)
		h.mu.Unlock()
		h.reconcileSimpleQuizTimer(next)
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
	h.reconcileSimpleQuizTimer(next)
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
	h.BroadcastToSession(moduleID, ServerMessage{
		Type:    "simple:state",
		Payload: simpleStatePayload(s),
	})
}

func (h *QuizHub) reconcileSimpleQuizTimer(s *models.SimpleQuizSession) {
	moduleID := s.ModuleID
	h.mu.Lock()
	defer h.mu.Unlock()
	if t, ok := h.simpleQuizTimers[moduleID]; ok && t != nil {
		t.Stop()
	}
	delete(h.simpleQuizTimers, moduleID)
	h.simpleQuizTimerGeneration[moduleID]++
	generation := h.simpleQuizTimerGeneration[moduleID]
	if h.closed || s.Status != models.SimpleQuizStateQuestionOpen || s.DeadlineUnix <= 0 {
		return
	}
	created, opened, index, deadline := s.CreatedAt, s.QuestionOpenedAt, s.CurrentIndex, s.DeadlineUnix
	delay := time.Until(time.Unix(deadline, 0))
	if delay < 0 {
		delay = 0
	}
	h.simpleQuizTimers[moduleID] = time.AfterFunc(delay, func() {
		h.autoLockSimpleQuiz(moduleID, generation, created, opened, index, deadline)
	})
}

func (h *QuizHub) cancelSimpleQuizTimer(moduleID string) {
	h.mu.Lock()
	h.simpleQuizTimerGeneration[moduleID]++
	if t, ok := h.simpleQuizTimers[moduleID]; ok && t != nil {
		t.Stop()
		delete(h.simpleQuizTimers, moduleID)
	}
	h.mu.Unlock()
}

func (h *QuizHub) autoLockSimpleQuiz(moduleID string, generation uint64, created, opened time.Time, index int, deadline int64) {
	lock := h.simpleLock(moduleID)
	lock.Lock()
	h.mu.RLock()
	closed := h.closed || h.simpleQuizTimerGeneration[moduleID] != generation
	h.mu.RUnlock()
	if closed {
		lock.Unlock()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := h.tryUpdateSimpleLocked(ctx, moduleID, func(s *models.SimpleQuizSession) error {
		if s.Status != models.SimpleQuizStateQuestionOpen || !s.CreatedAt.Equal(created) || !s.QuestionOpenedAt.Equal(opened) || s.CurrentIndex != index || s.DeadlineUnix != deadline {
			return errNoSimpleChange
		}
		s.Status = models.SimpleQuizStateAnswersLocked
		s.DeadlineUnix = 0
		return nil
	})
	lock.Unlock()
	if err == nil {
		h.BroadcastSimpleQuiz(moduleID)
		return
	}
	if !errors.Is(err, postgres.ErrNotFound) && !errors.Is(err, errNoSimpleChange) {
		slog.Error("simple quiz auto-lock failed", "module", moduleID, "error", err)
		time.AfterFunc(time.Second, func() { h.autoLockSimpleQuiz(moduleID, generation, created, opened, index, deadline) })
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
		return subtle.ConstantTimeCompare([]byte(qd.JudgeToken), []byte(token)) == 1
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

// simpleJoinModule verifies module type before assigning a WebSocket role.
func (h *QuizHub) simpleJoinModule(moduleID string) bool {
	if h.db == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	event, err := h.db.GetEventByModule(ctx, moduleID)
	if err != nil {
		return false
	}
	for _, module := range event.Modules {
		if module.ID == moduleID {
			return module.Type == models.ModuleTypeSimpleQuiz
		}
	}
	return false
}

func (h *QuizHub) processSimpleHostJoin(client *Client, msg ClientMessage) {
	data, _ := msg.Payload.(map[string]interface{})
	moduleID, _ := data["moduleId"].(string)
	if moduleID == "" || !h.simpleJoinModule(moduleID) {
		return
	}
	h.mu.Lock()
	client.ModuleID = moduleID
	client.IsTeamsAdmin = false
	client.IsHost = true
	client.IsProjector = false
	client.PlayerID = ""
	h.mu.Unlock()
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
	secret, _ := data["secret"].(string)
	if moduleID == "" || !h.simpleJoinModule(moduleID) {
		return
	}
	s := h.GetSimpleQuiz(moduleID)
	if s == nil || secret == "" || s.Players[playerID] == nil || subtle.ConstantTimeCompare([]byte(s.Players[playerID].Secret), []byte(secret)) != 1 {
		return
	}
	h.mu.Lock()
	client.ModuleID = moduleID
	client.IsTeamsAdmin = false
	client.IsHost = false
	client.IsProjector = false
	client.PlayerID = playerID
	h.mu.Unlock()
	h.sendToClient(client, mustMarshal(ServerMessage{Type: "simple:state", Payload: simpleStatePayload(s)}))
}

func (h *QuizHub) processSimpleProjectorJoin(client *Client, msg ClientMessage) {
	data, _ := msg.Payload.(map[string]interface{})
	moduleID, _ := data["moduleId"].(string)
	if moduleID == "" || !h.simpleJoinModule(moduleID) {
		return
	}
	h.mu.Lock()
	client.ModuleID = moduleID
	client.IsTeamsAdmin = false
	client.IsProjector = true
	client.IsHost = false
	client.PlayerID = ""
	h.mu.Unlock()
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
	if moduleID == "" || token == "" || !h.simpleJoinModule(moduleID) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !h.ValidateSimpleJudgeToken(ctx, moduleID, token) {
		return
	}
	h.mu.Lock()
	client.ModuleID = moduleID
	client.IsTeamsAdmin = false
	client.IsHost = false
	client.IsProjector = false
	client.PlayerID = ""
	h.mu.Unlock()
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
		"updatedAt":        s.UpdatedAt,
	}
}
