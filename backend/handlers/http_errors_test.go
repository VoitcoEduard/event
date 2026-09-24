package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInternalErrorDoesNotExposeDetails(t *testing.T) {
	w := httptest.NewRecorder()
	writeRequestError(w, errors.New("mongodb://secret-host:27017"), http.StatusInternalServerError)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "secret-host") {
		t.Fatalf("response = %d %q", w.Code, w.Body.String())
	}
}

func TestCreateEventRejectsOversizedBody(t *testing.T) {
	for _, body := range []string{
		`{"name":"` + strings.Repeat("x", 4<<20) + `"}`,
		`{"name":"test"}` + strings.Repeat(" ", 4<<20),
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
		CreateEvent(w, r)
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", w.Code)
		}
	}
}

func TestCreateEventRejectsTrailingJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"name":"test"}{"name":"other"}`))
	CreateEvent(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
