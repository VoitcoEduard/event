package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/user/go-expo-sample/backend/db"
	"github.com/user/go-expo-sample/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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

// persistSimpleQuiz writes the session through to Mongo (best-effort).
func (h *QuizHub) persistSimpleQuiz(ctx context.Context, s *models.SimpleQuizSession) {
	if db.Client == nil {
		return
	}
	s.UpdatedAt = time.Now()
	coll := db.GetCollection("simple_quiz_sessions")
	filter := bson.M{"moduleId": s.ModuleID}
	_, err := coll.ReplaceOne(ctx, filter, s, options.Replace().SetUpsert(true))
	if err != nil {
		log.Printf("simple-quiz persist failed for module %s: %v", s.ModuleID, err)
	}
}

// loadSimpleQuizFromDB restores a session after restart (marks paused if active).
func (h *QuizHub) loadSimpleQuizFromDB(ctx context.Context, moduleID string) *models.SimpleQuizSession {
	if db.Client == nil {
		return nil
	}
	coll := db.GetCollection("simple_quiz_sessions")
	var s models.SimpleQuizSession
	if err := coll.FindOne(ctx, bson.M{"moduleId": moduleID}).Decode(&s); err != nil {
		return nil
	}
	if s.Status == models.SimpleQuizStateQuestionOpen {
		s.PausedFrom = s.Status
		s.Status = models.SimpleQuizStatePaused
		s.DeadlineUnix = 0
	}
	if s.Players == nil {
		s.Players = make(map[string]*models.SimpleQuizPlayer)
	}
	if s.Grades == nil {
		s.Grades = make(map[string]*models.SimpleQuizGrade)
	}
	if s.Revealed == nil {
		s.Revealed = make(map[int]bool)
	}
	return &s
}

// GetSimpleQuiz returns the in-memory session, falling back to DB.
func (h *QuizHub) GetSimpleQuiz(moduleID string) *models.SimpleQuizSession {
	h.mu.RLock()
	s, ok := h.simpleQuizSessions[moduleID]
	h.mu.RUnlock()
	if ok {
		return s
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	restored := h.loadSimpleQuizFromDB(ctx, moduleID)
	if restored == nil {
		return nil
	}
	h.mu.Lock()
	h.simpleQuizSessions[moduleID] = restored
	h.mu.Unlock()
	return restored
}

// SetSimpleQuiz stores and persists the session, then broadcasts.
func (h *QuizHub) SetSimpleQuiz(s *models.SimpleQuizSession) {
	h.mu.Lock()
	h.simpleQuizSessions[s.ModuleID] = s
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.persistSimpleQuiz(ctx, s)
	h.BroadcastSimpleQuiz(s.ModuleID)
}

// UpdateSimpleQuiz mutates the session atomically, persists and broadcasts.
func (h *QuizHub) UpdateSimpleQuiz(moduleID string, fn func(*models.SimpleQuizSession)) *models.SimpleQuizSession {
	h.mu.Lock()
	s, ok := h.simpleQuizSessions[moduleID]
	if !ok {
		h.mu.Unlock()
		return nil
	}
	fn(s)
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.persistSimpleQuiz(ctx, s)
	h.BroadcastSimpleQuiz(moduleID)
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
	h.mu.Lock()
	cur, ok := h.simpleQuizSessions[moduleID]
	if !ok || cur.Status != models.SimpleQuizStateQuestionOpen {
		h.mu.Unlock()
		return
	}
	cur.Status = models.SimpleQuizStateAnswersLocked
	cur.DeadlineUnix = 0
	h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	h.persistSimpleQuiz(ctx, cur)
	h.BroadcastSimpleQuiz(moduleID)
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
	if h.eventByModule == nil || db.Client == nil {
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
		coll := db.GetCollection("events")
		filter := bson.M{"_id": eventID, "modules.id": moduleID}
		update := bson.M{"$set": bson.M{"modules.$.data.judgeToken": token}}
		if _, err := coll.UpdateOne(ctx, filter, update); err != nil {
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
