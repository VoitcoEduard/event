package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
