package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var teamsSubscribers = struct {
	sync.Mutex
	groups map[string]map[chan int64]bool
}{groups: map[string]map[chan int64]bool{}}

func teamsStreamKey(e, m string) string { b, _ := json.Marshal([]string{e, m}); return string(b) }
func notifyTeamsSubscribers(e, m string, revision int64) {
	teamsSubscribers.Lock()
	defer teamsSubscribers.Unlock()
	for ch := range teamsSubscribers.groups[teamsStreamKey(e, m)] {
		select {
		case ch <- revision:
		default:
		}
	}
}
func (h *Handler) TeamsActivityStream(w http.ResponseWriter, r *http.Request) {
	if _, err := h.teamsModule(r); err != nil {
		teamsFailure(w, err)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unavailable", 500)
		return
	}
	e, m := teamsIDs(r)
	key := teamsStreamKey(e, m)
	ch := make(chan int64, 1)
	teamsSubscribers.Lock()
	if teamsSubscribers.groups[key] == nil {
		teamsSubscribers.groups[key] = map[chan int64]bool{}
	}
	teamsSubscribers.groups[key][ch] = true
	teamsSubscribers.Unlock()
	defer func() {
		teamsSubscribers.Lock()
		delete(teamsSubscribers.groups[key], ch)
		if len(teamsSubscribers.groups[key]) == 0 {
			delete(teamsSubscribers.groups, key)
		}
		teamsSubscribers.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, "event: ready\ndata: {}\n\n")
	flusher.Flush()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case revision := <-ch:
			fmt.Fprintf(w, "data: {\"eventId\":%q,\"moduleId\":%q,\"revision\":%d}\n\n", e, m, revision)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
