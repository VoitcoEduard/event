package handlers

import "net/http"

// isAdmin allows public read endpoints to return the full editor data only to
// an active administrator. Anonymous and invalid sessions receive public data.
func (h *Handler) isAdmin(r *http.Request) bool {
	if h.deps.Auth == nil || h.deps.Auth.Token(r) == "" {
		return false
	}
	_, user, err := h.deps.Auth.Current(r)
	return err == nil && user.Status == "active" && (user.Role == "admin" || user.Role == "owner")
}

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
