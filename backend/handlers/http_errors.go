package handlers

import (
	"errors"
	"log/slog"
	"net/http"
)

func writeRequestError(w http.ResponseWriter, err error, status int) {
	slog.Error("request failed", "status", status, "error", err)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	if status == http.StatusBadRequest {
		http.Error(w, "Invalid request", status)
		return
	}
	http.Error(w, "Internal server error", status)
}
