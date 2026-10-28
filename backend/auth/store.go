package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/user/go-expo-sample/backend/postgres"
)

const sessionLifetime = 12 * time.Hour
const loginLifetime = 10 * time.Minute

var ErrNoSession = errors.New("no valid session")

type User = postgres.AuthUser
type Session = postgres.AuthSession
type LoginTransaction = postgres.AuthLogin
type Store struct{ DB *postgres.Store }

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func (s Store) UpsertGoogleUser(ctx context.Context, issuer, subject, email, name string) (User, error) {
	return s.DB.UpsertGoogleUser(ctx, issuer, subject, email, name)
}
func (s Store) EnsureDevUser(ctx context.Context) (User, error) { return s.DB.EnsureDevUser(ctx) }
func (s Store) NewSession(ctx context.Context, userID string) (string, Session, error) {
	token, err := randomToken()
	if err != nil {
		return "", Session{}, err
	}
	csrf, err := randomToken()
	if err != nil {
		return "", Session{}, err
	}
	session := Session{Hash: digest(token), UserID: userID, CSRF: csrf, ExpiresAt: time.Now().Add(sessionLifetime).UTC()}
	if err = s.DB.CreateAuthSession(ctx, session); err != nil {
		return "", Session{}, err
	}
	return token, session, nil
}
func (s Store) Resolve(ctx context.Context, token string) (Session, User, error) {
	if token == "" {
		return Session{}, User{}, ErrNoSession
	}
	session, user, err := s.DB.ResolveAuthSession(ctx, digest(token), time.Now().UTC())
	if errors.Is(err, postgres.ErrNotFound) {
		return Session{}, User{}, ErrNoSession
	}
	return session, user, err
}
func (s Store) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.DB.DeleteAuthSession(ctx, digest(token))
}
func (s Store) AuditAction(ctx context.Context, actor, target, action string) error {
	return s.DB.AuditAuthAction(ctx, actor, target, action)
}
