package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/user/go-expo-sample/backend/postgres"
)

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil || r.Body == http.NoBody {
		return nil
	}
	if err := decodeJSON(w, r, dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func writeRequestError(w http.ResponseWriter, err error, status int) {
	if errors.Is(err, postgres.ErrInvalidValue) {
		http.Error(w, "Points must be finite", http.StatusBadRequest)
		return
	}
	if errors.Is(err, postgres.ErrInvalidReference) {
		http.Error(w, "Invalid reference", http.StatusBadRequest)
		return
	}
	if errors.Is(err, postgres.ErrConflict) {
		http.Error(w, "Conflict", http.StatusConflict)
		return
	}
	if errors.Is(err, postgres.ErrNotFound) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		// Client-side body size violations are expected noise on public
		// routes; never log them as server errors.
		slog.Debug("request body too large", "error", err)
		http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	if status == http.StatusBadRequest {
		// Malformed client payloads are not server errors; keep them out of
		// error-level logs while still responding 400.
		slog.Debug("bad request", "error", err)
		http.Error(w, "Invalid request", status)
		return
	}
	if status >= http.StatusInternalServerError {
		slog.Error("request failed", "status", status, "error", err)
	}
	http.Error(w, "Internal server error", status)
}
