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
	w := httptest.NewRecorder()
	body := `{"name":"` + strings.Repeat("x", 4<<20) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
	CreateEvent(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", w.Code)
	}
}
