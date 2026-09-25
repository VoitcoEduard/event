package handlers

import "net/http"

// VerifyAuth is a middleware that checks for a valid Bearer token.
func (h *Handler) VerifyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.deps.Auth == nil {
			http.Error(w, "Authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		if _, _, ok := h.deps.Auth.Authorize(w, r, false); !ok {
			return
		}
		next(w, r)
	}
}
