package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/oauth2"
)

const sessionCookie = "__Host-admin_session"
const loginCookie = "__Host-oauth_login"

type Service struct {
	Config   Config
	Store    Store
	OAuth    *oauth2.Config
	Verifier *oidc.IDTokenVerifier
}

func NewService(ctx context.Context, config Config, store Store) (*Service, error) {
	service := &Service{Config: config, Store: store}
	if config.Mode == "google" {
		provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			return nil, fmt.Errorf("discover Google OIDC: %w", err)
		}
		service.Verifier = provider.Verifier(&oidc.Config{ClientID: config.ClientID})
		service.OAuth = &oauth2.Config{
			ClientID: config.ClientID, ClientSecret: config.Secret,
			Endpoint: provider.Endpoint(), RedirectURL: config.BaseURL + "/auth/google/callback",
			Scopes: []string{oidc.ScopeOpenID, "email", "profile"},
		}
	}
	return service, nil
}

func (s *Service) secure() bool { return strings.HasPrefix(s.Config.BaseURL, "https://") }

func (s *Service) cookieName(name string) string {
	if s.secure() {
		return name
	}
	return strings.TrimPrefix(name, "__Host-") + "_dev"
}

func (s *Service) setCookie(w http.ResponseWriter, name, value string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{Name: s.cookieName(name), Value: value, Path: "/", HttpOnly: true, Secure: s.secure(), SameSite: http.SameSiteLaxMode, Expires: expiry})
}

func (s *Service) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: s.cookieName(name), Path: "/", HttpOnly: true, Secure: s.secure(), SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func (s *Service) sessionToken(r *http.Request) string {
	c, err := r.Cookie(s.cookieName(sessionCookie))
	if err != nil {
		return ""
	}
	return c.Value
}

func (s *Service) Token(r *http.Request) string { return s.sessionToken(r) }

func (s *Service) Current(r *http.Request) (Session, User, error) {
	return s.Store.Resolve(r.Context(), s.sessionToken(r))
}

func (s *Service) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return origin == s.Config.BaseURL
}

func unsafeMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func (s *Service) Authorize(w http.ResponseWriter, r *http.Request, owner bool) (Session, User, bool) {
	session, user, err := s.Current(r)
	if errors.Is(err, ErrNoSession) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return Session{}, User{}, false
	}
	if err != nil {
		http.Error(w, "Session unavailable", http.StatusServiceUnavailable)
		return Session{}, User{}, false
	}
	if user.Status != "active" || (user.Role != "owner" && user.Role != "admin") || (owner && user.Role != "owner") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return Session{}, User{}, false
	}
	if unsafeMethod(r.Method) && (!s.sameOrigin(r) || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(session.CSRF)) != 1) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return Session{}, User{}, false
	}
	return session, user, true
}

func (s *Service) issueSession(w http.ResponseWriter, r *http.Request, userID string) error {
	old := s.sessionToken(r)
	if old != "" {
		if err := s.Store.Revoke(r.Context(), old); err != nil {
			return err
		}
	}
	token, session, err := s.Store.NewSession(r.Context(), userID)
	if err != nil {
		return err
	}
	s.setCookie(w, sessionCookie, token, session.ExpiresAt)
	return nil
}

func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/config", s.ConfigHandler)
	mux.HandleFunc("GET /auth/google/start", s.GoogleStart)
	mux.HandleFunc("GET /auth/google/callback", s.GoogleCallback)
	mux.HandleFunc("POST /auth/dev/login", s.DevLogin)
	mux.HandleFunc("GET /auth/me", s.Me)
	mux.HandleFunc("POST /auth/logout", s.Logout)
	mux.HandleFunc("GET /auth/users", s.ListUsers)
	mux.HandleFunc("PATCH /auth/users/{id}", s.SetUserStatus)
}

func noStore(w http.ResponseWriter) { w.Header().Set("Cache-Control", "no-store") }
func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Service) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	writeJSON(w, map[string]string{"mode": s.Config.Mode})
}

func (s *Service) GoogleStart(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if s.Config.Mode != "google" {
		http.NotFound(w, r)
		return
	}
	state, err := randomToken()
	if err != nil {
		http.Error(w, "Login unavailable", 500)
		return
	}
	browser, err := randomToken()
	if err != nil {
		http.Error(w, "Login unavailable", 500)
		return
	}
	nonce, err := randomToken()
	if err != nil {
		http.Error(w, "Login unavailable", 500)
		return
	}
	verifier := oauth2.GenerateVerifier()
	expiry := time.Now().Add(loginLifetime)
	_, err = s.Store.Logins().InsertOne(r.Context(), LoginTransaction{Hash: digest(state), Browser: digest(browser), Nonce: nonce, Verifier: verifier, ExpiresAt: expiry})
	if err != nil {
		http.Error(w, "Login unavailable", 503)
		return
	}
	s.setCookie(w, loginCookie, browser, expiry)
	http.Redirect(w, r, s.OAuth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce)), http.StatusFound)
}

func (s *Service) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	w.Header().Set("Referrer-Policy", "no-referrer")
	if s.Config.Mode != "google" {
		http.NotFound(w, r)
		return
	}
	s.clearCookie(w, loginCookie)
	if r.URL.Query().Get("error") != "" {
		http.Redirect(w, r, "/login?error=cancelled", http.StatusFound)
		return
	}
	state, code := r.URL.Query().Get("state"), r.URL.Query().Get("code")
	browserCookie, err := r.Cookie(s.cookieName(loginCookie))
	if state == "" || code == "" || err != nil || browserCookie.Value == "" {
		http.Error(w, "Invalid login", 400)
		return
	}
	var transaction LoginTransaction
	err = s.Store.Logins().FindOneAndDelete(r.Context(), bson.M{"_id": digest(state), "browser": digest(browserCookie.Value), "expiresAt": bson.M{"$gt": time.Now().UTC()}}).Decode(&transaction)
	if err != nil {
		http.Error(w, "Invalid or expired login", 400)
		return
	}
	token, err := s.OAuth.Exchange(r.Context(), code, oauth2.VerifierOption(transaction.Verifier))
	if err != nil {
		http.Error(w, "Google login failed", 401)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "Missing identity", 401)
		return
	}
	idToken, err := s.Verifier.Verify(r.Context(), rawIDToken)
	if err != nil || idToken.Nonce != transaction.Nonce {
		http.Error(w, "Invalid identity", 401)
		return
	}
	var claims struct {
		Issuer        string          `json:"iss"`
		Subject       string          `json:"sub"`
		Email         string          `json:"email"`
		EmailVerified json.RawMessage `json:"email_verified"`
		Name          string          `json:"name"`
	}
	if err = idToken.Claims(&claims); err != nil || claims.Subject == "" || claims.Issuer == "" || claims.Email == "" || !verifiedEmail(claims.EmailVerified) {
		http.Error(w, "Unverified identity", 401)
		return
	}
	user, err := s.Store.UpsertGoogleUser(r.Context(), claims.Issuer, claims.Subject, claims.Email, claims.Name)
	if err != nil {
		http.Error(w, "Login unavailable", 503)
		return
	}
	if user.Status == "disabled" {
		http.Redirect(w, r, "/login?error=disabled", http.StatusFound)
		return
	}
	if err = s.issueSession(w, r, user.ID); err != nil {
		http.Error(w, "Login unavailable", 503)
		return
	}
	destination := "/pending"
	if user.Status == "active" {
		destination = "/admin"
	}
	http.Redirect(w, r, destination, http.StatusFound)
}

func verifiedEmail(raw json.RawMessage) bool {
	var verified bool
	if json.Unmarshal(raw, &verified) == nil {
		return verified
	}
	var text string
	return json.Unmarshal(raw, &text) == nil && text == "true"
}

func (s *Service) DevLogin(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if s.Config.Mode != "dev" {
		http.NotFound(w, r)
		return
	}
	if !s.sameOrigin(r) {
		http.Error(w, "Forbidden", 403)
		return
	}
	var input struct {
		Password string `json:"password"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil {
		http.Error(w, "Invalid request", 400)
		return
	}
	if subtle.ConstantTimeCompare([]byte(input.Password), []byte(s.Config.DevPassword)) != 1 {
		http.Error(w, "Unauthorized", 401)
		return
	}
	user, err := s.Store.EnsureDevUser(r.Context())
	if err != nil {
		http.Error(w, "Login unavailable", 503)
		return
	}
	if err := s.issueSession(w, r, user.ID); err != nil {
		http.Error(w, "Login unavailable", 503)
		return
	}
	writeJSON(w, map[string]any{"user": user})
}

func (s *Service) Me(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	session, user, err := s.Current(r)
	if errors.Is(err, ErrNoSession) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	if err != nil {
		http.Error(w, "Session unavailable", 503)
		return
	}
	writeJSON(w, map[string]any{"user": user, "expiresAt": session.ExpiresAt, "csrfToken": session.CSRF})
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	session, _, err := s.Current(r)
	if errors.Is(err, ErrNoSession) {
		http.Error(w, "Unauthorized", 401)
		return
	}
	if err != nil {
		http.Error(w, "Session unavailable", 503)
		return
	}
	if !s.sameOrigin(r) || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(session.CSRF)) != 1 {
		http.Error(w, "Forbidden", 403)
		return
	}
	if err := s.Store.Revoke(r.Context(), s.sessionToken(r)); err != nil {
		http.Error(w, "Logout unavailable", 503)
		return
	}
	s.clearCookie(w, sessionCookie)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) ListUsers(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if _, _, ok := s.Authorize(w, r, true); !ok {
		return
	}
	page := 0
	if value := r.URL.Query().Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			http.Error(w, "Invalid page", 400)
			return
		}
		page = parsed
	}
	if page > 10000 {
		http.Error(w, "Invalid page", 400)
		return
	}
	cursor, err := s.Store.Users().Find(r.Context(), bson.M{"issuer": bson.M{"$ne": "development"}}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}, {Key: "_id", Value: 1}}).SetSkip(int64(page)*50).SetLimit(51))
	if err != nil {
		http.Error(w, "Users unavailable", 503)
		return
	}
	defer cursor.Close(r.Context())
	users := make([]User, 0)
	for cursor.Next(r.Context()) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			http.Error(w, "Users unavailable", 503)
			return
		}
		users = append(users, user)
	}
	if cursor.Err() != nil {
		http.Error(w, "Users unavailable", 503)
		return
	}
	// The admin population is small; pagination still limits the response size.
	end := 50
	if end > len(users) {
		end = len(users)
	}
	writeJSON(w, map[string]any{"users": users[:end], "page": page, "hasMore": end < len(users)})
}

func (s *Service) SetUserStatus(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	_, actor, ok := s.Authorize(w, r, true)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil || (input.Status != "active" && input.Status != "disabled") {
		http.Error(w, "Invalid status", 400)
		return
	}
	id := r.PathValue("id")
	var user User
	err := s.Store.Users().FindOne(r.Context(), bson.M{"_id": id}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Users unavailable", 503)
		return
	}
	if user.Role == "owner" || user.Issuer == "development" {
		http.Error(w, "Owner status requires server administration", 403)
		return
	}
	update, err := s.Store.Users().UpdateOne(r.Context(), bson.M{"_id": id, "role": "admin"}, bson.M{"$set": bson.M{"status": input.Status}})
	if err != nil {
		http.Error(w, "Users unavailable", 503)
		return
	}
	if update.MatchedCount != 1 {
		http.Error(w, "Owner status requires server administration", 403)
		return
	}
	if input.Status == "disabled" {
		if _, err := s.Store.Sessions().DeleteMany(r.Context(), bson.M{"userId": id}); err != nil {
			http.Error(w, "Users unavailable", 503)
			return
		}
	}
	if err := s.Store.AuditAction(r.Context(), actor.ID, id, "set-status:"+input.Status); err != nil {
		http.Error(w, "Audit unavailable", 503)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
