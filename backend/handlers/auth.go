package handlers

import (
	"encoding/json"
	"net/http"
	"os"
)

// Login handles POST /login
// Reads ADMIN_PASSWORD from env (defaults to "admin123" if not set).
func Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	if input.Password != adminPassword {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": "valid-admin-token"})
}
