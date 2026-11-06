package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/user/go-expo-sample/backend/models"
)

func newParticipantSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// JoinEventParticipant grants proof only when a new participant is inserted.
// Existing and legacy rows cannot be claimed by knowing a public player ID.
func (s *Store) JoinEventParticipant(ctx context.Context, eventID, id, joinKey, name string, profile map[string]any, providedSecret string) (*models.Participant, error) {
	if id == "" {
		id = uuid.NewString()
	}
	secret, err := newParticipantSecret()
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(secret))
	profileRaw, err := marshal(profile)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		profileRaw = []byte("{}")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()
	row := tx.QueryRow(ctx, `INSERT INTO participants(event_id,id,join_key,display_name,normalized_name,profile,status,created_at,updated_at,secret_hash)
        VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,'joined',$7,$7,$8)
        ON CONFLICT(event_id,id) DO NOTHING RETURNING `+participantColumns,
		eventID, id, joinKey, name, strings.ToLower(strings.TrimSpace(name)), profileRaw, now, hash[:])
	p, err := scanParticipant(row)
	if err == nil {
		p.JoinSecret = secret
		if err = tx.Commit(ctx); err != nil {
			return nil, translate(err)
		}
		return p, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, translate(err)
	}
	var storedHash []byte
	if err = tx.QueryRow(ctx, `SELECT secret_hash FROM participants WHERE event_id=$1 AND id=$2 FOR UPDATE`, eventID, id).Scan(&storedHash); err != nil {
		return nil, translate(err)
	}
	providedHash := sha256.Sum256([]byte(providedSecret))
	if len(storedHash) != len(providedHash) || providedSecret == "" || subtle.ConstantTimeCompare(storedHash, providedHash[:]) != 1 {
		return nil, ErrConflict
	}
	p, err = scanParticipant(tx.QueryRow(ctx, `SELECT `+participantColumns+` FROM participants WHERE event_id=$1 AND id=$2`, eventID, id))
	if err != nil {
		return nil, err
	}
	p.JoinSecret = providedSecret
	if err = tx.Commit(ctx); err != nil {
		return nil, translate(err)
	}
	return p, nil
}

// CheckParticipantProof reports whether a participant has a credential and
// whether the supplied secret matches it. Legacy rows have hasProof=false.
func (s *Store) CheckParticipantProof(ctx context.Context, eventID, id, secret string) (hasProof, valid bool, err error) {
	var storedHash []byte
	err = s.Pool.QueryRow(ctx, `SELECT secret_hash FROM participants WHERE event_id=$1 AND id=$2`, eventID, id).Scan(&storedHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, ErrNotFound
	}
	if err != nil {
		return false, false, translate(err)
	}
	if len(storedHash) == 0 {
		return false, false, nil
	}
	hash := sha256.Sum256([]byte(secret))
	return true, secret != "" && len(storedHash) == len(hash) && subtle.ConstantTimeCompare(storedHash, hash[:]) == 1, nil
}
