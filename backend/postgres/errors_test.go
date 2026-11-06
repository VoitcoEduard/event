package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestCheckConstraintTranslationOnlyMapsKnownInput(t *testing.T) {
	known := &pgconn.PgError{Code: "23514", ConstraintName: "score_entries_points_check"}
	if !errors.Is(translate(known), ErrInvalidValue) {
		t.Fatal("finite-points check should be a validation error")
	}
	unknown := &pgconn.PgError{Code: "23514", ConstraintName: "score_entries_check"}
	if errors.Is(translate(unknown), ErrInvalidValue) {
		t.Fatal("unexpected invariant should remain a server error")
	}
}
