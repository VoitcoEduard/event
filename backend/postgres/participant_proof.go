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
	var profileProvidedRaw []byte
	if profile != nil {
		profileProvidedRaw = profileRaw
	} else {
		profileRaw = []byte("{}")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	now := time.Now().UTC()

	targetID := id
	foundExisting := false
	if joinKey != "" {
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, eventID+"/"+joinKey); err != nil {
			return nil, translate(err)
		}
		var existingID string
		var storedHash []byte
		findErr := tx.QueryRow(ctx, `SELECT id, secret_hash FROM participants WHERE event_id=$1 AND join_key=$2 FOR UPDATE`, eventID, joinKey).Scan(&existingID, &storedHash)
		if findErr == nil {
			providedHash := sha256.Sum256([]byte(providedSecret))
			if len(storedHash) != len(providedHash) || providedSecret == "" || subtle.ConstantTimeCompare(storedHash, providedHash[:]) != 1 {
				return nil, ErrConflict
			}
			targetID = existingID
			foundExisting = true
		} else if !errors.Is(findErr, pgx.ErrNoRows) {
			return nil, translate(findErr)
		}
	}

	if !foundExisting {
		row := tx.QueryRow(ctx, `INSERT INTO participants(event_id,id,join_key,display_name,normalized_name,profile,status,created_at,updated_at,secret_hash)
        VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,'joined',$7,$7,$8)
        ON CONFLICT(event_id,id) DO NOTHING RETURNING `+participantColumns,
			eventID, targetID, joinKey, name, strings.ToLower(strings.TrimSpace(name)), profileRaw, now, hash[:])
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
		if err = tx.QueryRow(ctx, `SELECT secret_hash FROM participants WHERE event_id=$1 AND id=$2 FOR UPDATE`, eventID, targetID).Scan(&storedHash); err != nil {
			return nil, translate(err)
		}
		providedHash := sha256.Sum256([]byte(providedSecret))
		if len(storedHash) != len(providedHash) || providedSecret == "" || subtle.ConstantTimeCompare(storedHash, providedHash[:]) != 1 {
			return nil, ErrConflict
		}
	}

	updateName := ""
	if strings.TrimSpace(name) != "" {
		updateName = name
	}
	normalizedName := strings.ToLower(strings.TrimSpace(updateName))
	p, err := scanParticipant(tx.QueryRow(ctx, `UPDATE participants SET
		display_name=CASE WHEN $3<>'' THEN $3 ELSE display_name END,
		normalized_name=CASE WHEN $3<>'' THEN $4 ELSE normalized_name END,
		profile=COALESCE($5::jsonb,profile),
		join_key=COALESCE(participants.join_key,NULLIF($6,'')),
		updated_at=$7
		WHERE event_id=$1 AND id=$2 RETURNING `+participantColumns,
		eventID, targetID, updateName, normalizedName, profileProvidedRaw, joinKey, now))
	if err != nil {
		return nil, err
	}
	p.JoinSecret = providedSecret
	if err = tx.Commit(ctx); err != nil {
		return nil, translate(err)
	}
	return p, nil
}

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
