package hub

import "sync"

// ForgetModuleSessions evicts runtime data after the module has been deleted
// from storage. Wait for in-flight simple-quiz operations before eviction so
// they cannot publish a stale snapshot after cleanup.
func (h *QuizHub) ForgetModuleSessions(moduleID string) {
	// A module without a lock has no simple-quiz operation in flight. Avoid
	// allocating permanent lock entries for unrelated module types on deletion.
	if value, ok := h.simpleQuizLocks.Load(moduleID); ok {
		lock := value.(*sync.Mutex)
		lock.Lock()
		defer lock.Unlock()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cancelQuizTransitionLocked(moduleID)
	if timer := h.simpleQuizTimers[moduleID]; timer != nil {
		timer.Stop()
	}
	delete(h.simpleQuizTimers, moduleID)
	delete(h.sessions, moduleID)
	delete(h.simpleQuizSessions, moduleID)
	delete(h.simpleQuizUndo, moduleID)
	h.broadcastToSessionLocked(moduleID, ServerMessage{Type: "cmd:state", Payload: map[string]any{"state": "inactive"}})
	h.broadcastToSessionLocked(moduleID, ServerMessage{Type: "simple:state", Payload: map[string]any{"status": "inactive"}})
}

// Include the event's stored module IDs as well as cached sessions: an operation
// may be loading a session for the first time when the event is deleted.
func (h *QuizHub) ForgetEventSessions(eventID string, moduleIDs []string) {
	modules := make(map[string]struct{}, len(moduleIDs))
	for _, moduleID := range moduleIDs {
		modules[moduleID] = struct{}{}
	}
	h.mu.RLock()
	for moduleID, session := range h.sessions {
		if session.EventID == eventID {
			modules[moduleID] = struct{}{}
		}
	}
	for moduleID, session := range h.simpleQuizSessions {
		if session.EventID == eventID {
			modules[moduleID] = struct{}{}
		}
	}
	h.mu.RUnlock()
	for moduleID := range modules {
		h.ForgetModuleSessions(moduleID)
	}
}
