package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
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
