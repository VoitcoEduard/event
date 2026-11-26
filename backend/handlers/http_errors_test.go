package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/user/go-expo-sample/backend/postgres"
)

func TestMissingRecordReturnsNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeRequestError(w, fmt.Errorf("update secret-table: %w", postgres.ErrNotFound), http.StatusInternalServerError)
	if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "secret-table") {
		t.Fatalf("response = %d %q", w.Code, w.Body.String())
	}
}

func TestInternalErrorDoesNotExposeDetails(t *testing.T) {
	w := httptest.NewRecorder()
	writeRequestError(w, errors.New("postgres://secret-host:5432"), http.StatusInternalServerError)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "secret-host") {
		t.Fatalf("response = %d %q", w.Code, w.Body.String())
	}
}

func TestCreateEventRejectsOversizedBody(t *testing.T) {
	h := New(Dependencies{})
	for _, body := range []string{
		`{"name":"` + strings.Repeat("x", 4<<20) + `"}`,
		`{"name":"test"}` + strings.Repeat(" ", 4<<20),
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
		h.CreateEvent(w, r)
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", w.Code)
		}
	}
}

func TestCreateEventRejectsTrailingJSON(t *testing.T) {
	h := New(Dependencies{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"name":"test"}{"name":"other"}`))
	h.CreateEvent(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWriteRequestErrorLogLevels(t *testing.T) {
	var mu sync.Mutex
	var sawError bool
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				mu.Lock()
				if a.Value.String() == "ERROR" {
					sawError = true
				}
				mu.Unlock()
			}
			return a
		},
	})))
	t.Cleanup(func() { slog.SetDefault(old) })

	w := httptest.NewRecorder()
	writeRequestError(w, errors.New("bad json"), http.StatusBadRequest)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	w = httptest.NewRecorder()
	writeRequestError(w, errors.New("too big"), http.StatusRequestEntityTooLarge)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
	mu.Lock()
	if sawError {
		mu.Unlock()
		t.Fatal("4xx request errors were logged at error level")
	}
	mu.Unlock()

	w = httptest.NewRecorder()
	writeRequestError(w, errors.New("boom"), http.StatusInternalServerError)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	mu.Lock()
	defer mu.Unlock()
	if !sawError {
		t.Fatal("500 was not logged at error level")
	}
}
