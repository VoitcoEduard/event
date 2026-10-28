// Package postgres owns PostgreSQL connection and schema management.
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const schemaVersion int64 = 1
const migrationLockID int64 = 0x70676d6967726174

// Store owns the connection pool. Construct one per application and close it at shutdown.
type Store struct {
	Pool *pgxpool.Pool
	url  string
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("invalid DATABASE_URL")
	}
	config.MaxConns = 10
	if value := os.Getenv("DATABASE_MAX_CONNECTIONS"); value != "" {
		n, parseErr := strconv.ParseInt(value, 10, 32)
		if parseErr != nil || n < 1 || n > 100 {
			return nil, errors.New("DATABASE_MAX_CONNECTIONS must be between 1 and 100")
		}
		config.MaxConns = int32(n)
	}
	config.MinConns = 0
	config.MaxConnIdleTime = 5 * time.Minute
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		pool, openErr := pgxpool.NewWithConfig(attemptCtx, config)
		if openErr == nil {
			openErr = pool.Ping(attemptCtx)
			if openErr == nil {
				cancel()
				return &Store{Pool: pool, url: databaseURL}, nil
			}
			pool.Close()
		}
		cancel()
		lastErr = openErr
		if attempt < 4 {
			timer := time.NewTimer(2 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil, fmt.Errorf("connect to PostgreSQL failed after retries: %T", lastErr)
}

func (s *Store) Close() { s.Pool.Close() }

func (s *Store) provider() (*goose.Provider, error) {
	migrationFS, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", s.url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationFS, goose.WithSessionLocker(locker))
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return provider, nil
}

// Migrate serializes version-table initialization as well as Goose's own locked
// migration execution. The outer lock uses a separate key from Goose's lock.
func (s *Store) Migrate(ctx context.Context) error {
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationLockID)
	}()
	provider, err := s.provider()
	if err != nil {
		return err
	}
	defer provider.Close()
	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return fmt.Errorf("inspect migrations: %w", err)
	}
	if target != schemaVersion {
		return fmt.Errorf("migration sources target %d, expected %d", target, schemaVersion)
	}
	if current > target {
		return fmt.Errorf("database schema version %d is newer than supported version %d", current, target)
	}
	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	current, target, err = provider.GetVersions(ctx)
	if err != nil {
		return fmt.Errorf("verify migrations: %w", err)
	}
	if current != target {
		return fmt.Errorf("database schema version %d, expected %d", current, target)
	}
	return nil
}

// Ready checks the live connection and the exact schema version, without modifying it.
func (s *Store) Ready(ctx context.Context) error {
	if err := s.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}
	var version int64
	if err := s.Pool.QueryRow(ctx, "SELECT COALESCE((SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 1), 0)").Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version != schemaVersion {
		return fmt.Errorf("database schema version %d, expected %d", version, schemaVersion)
	}
	return nil
}

func (s *Store) Versions(ctx context.Context) (int64, int64, error) {
	provider, err := s.provider()
	if err != nil {
		return 0, 0, err
	}
	defer provider.Close()
	return provider.GetVersions(ctx)
}
