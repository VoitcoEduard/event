package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/user/go-expo-sample/backend/models"
)

func TestSimpleQuizViewAuthorization(t *testing.T) {
	h := New(Dependencies{})
	s := &models.SimpleQuizSession{Players: map[string]*models.SimpleQuizPlayer{"p": {ID: "p", Secret: "private"}}}
	for _, tc := range []struct {
		view, id, secret string
		allowed          bool
	}{
		{"host", "", "", false}, {"projector", "", "", true},
		{"player", "p", "", false}, {"player", "p", "wrong", false},
		{"player", "p", "private", true}, {"player", "other", "private", false},
	} {
		t.Run(tc.view+tc.id+tc.secret, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.Header.Set("X-Player-Secret", tc.secret)
			w := httptest.NewRecorder()
			if got := h.authorizeSimpleQuizView(w, r, s, tc.view, tc.id); got != tc.allowed {
				t.Fatalf("allowed = %v", got)
			}
		})
	}
}
