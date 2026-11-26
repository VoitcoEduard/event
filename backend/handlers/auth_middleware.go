package handlers

import "net/http"

func (h *Handler) isAdmin(r *http.Request) bool {
	if h.deps.Auth == nil || h.deps.Auth.Token(r) == "" {
		return false
	}
	_, user, err := h.deps.Auth.Current(r)
	return err == nil && user.Status == "active" && (user.Role == "admin" || user.Role == "owner")
}

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
