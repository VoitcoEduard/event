package postgres

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound         = errors.New("record not found")
	ErrConflict         = errors.New("record conflicts with an existing record")
	ErrInvalidReference = errors.New("invalid record reference")
	ErrHasAwards        = errors.New("activity has awards and cannot be deleted")
)

// translate keeps PostgreSQL-specific errors inside persistence code.
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		switch pgerr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", ErrConflict, pgerr.ConstraintName)
		case "23503":
			return fmt.Errorf("%w: %s", ErrInvalidReference, pgerr.ConstraintName)
		}
	}
	return err
}

func marshal(value any) ([]byte, error) { return json.Marshal(value) }
func unmarshal(raw []byte, out any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
