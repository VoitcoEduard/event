package handlers

import (
	"github.com/user/go-expo-sample/backend/auth"
	"net/http"
)

var authService *auth.Service

func SetAuthService(service *auth.Service) { authService = service }

// VerifyAuth is a middleware that checks for a valid Bearer token.
func VerifyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authService == nil {
			http.Error(w, "Authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		if _, _, ok := authService.Authorize(w, r, false); !ok {
			return
		}
		next(w, r)
	}
}
