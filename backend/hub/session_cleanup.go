package hub

func (h *QuizHub) ForgetModuleSessions(moduleID string) {
	lock := h.simpleLock(moduleID)
	lock.Lock()
	defer lock.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cancelQuizTransitionLocked(moduleID)
	if timer := h.simpleQuizTimers[moduleID]; timer != nil {
		timer.Stop()
	}
	delete(h.simpleQuizTimers, moduleID)
	h.simpleQuizTimerGeneration[moduleID]++
	delete(h.sessions, moduleID)
	delete(h.simpleQuizSessions, moduleID)
	delete(h.simpleQuizUndo, moduleID)
	h.broadcastToSessionLocked(moduleID, ServerMessage{Type: "cmd:state", Payload: map[string]any{"state": "inactive"}})
	h.broadcastToSessionLocked(moduleID, ServerMessage{Type: "simple:state", Payload: map[string]any{"status": "inactive"}})
}

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
