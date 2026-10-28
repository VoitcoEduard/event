//go:build integration

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/go-expo-sample/backend/testutils"
)

func TestSessionAndApprovalIntegration(t *testing.T) {
	ctx := context.Background()
	store := Store{DB: testutils.SetupPostgres(t)}
	other := Store{DB: testutils.SetupPostgres(t)}
	owner, err := store.UpsertGoogleUser(ctx, "issuer", "owner-sub", "owner@example.test", "Owner")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.DB.Pool.Exec(ctx, `UPDATE auth_users SET role='owner',status='active' WHERE id=$1`, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	member, err := store.UpsertGoogleUser(ctx, "issuer", "member-sub", "first@example.test", "First")
	if err != nil {
		t.Fatal(err)
	}
	memberAgain, err := store.UpsertGoogleUser(ctx, "issuer", "member-sub", "second@example.test", "Second")
	if err != nil || memberAgain.ID != member.ID || memberAgain.Status != "pending" {
		t.Fatalf("identity changed: %#v %v", memberAgain, err)
	}
	ownerToken, _, err := store.NewSession(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := other.Resolve(ctx, ownerToken); !errors.Is(err, ErrNoSession) {
		t.Fatalf("environment leaked session: %v", err)
	}
	memberToken, _, err := store.NewSession(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{Config: Config{Mode: "dev", Environment: "development", BaseURL: "http://localhost:8081", DevPassword: "test-pass"}, Store: store}
	mux := http.NewServeMux()
	service.Register(mux)
	request := func(method, path, token, csrf, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Origin", service.Config.BaseURL)
		if token != "" {
			r.AddCookie(&http.Cookie{Name: service.cookieName(sessionCookie), Value: token})
		}
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if code := request("GET", "/auth/users", memberToken, "", "").Code; code != 403 {
		t.Fatalf("pending user listed admins: %d", code)
	}
	me := request("GET", "/auth/me", ownerToken, "", "")
	var payload struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(me.Body.Bytes(), &payload); err != nil || payload.CSRFToken == "" {
		t.Fatalf("me response: %s %v", me.Body.String(), err)
	}
	if code := request("PATCH", "/auth/users/"+member.ID, ownerToken, "", `{"status":"active"}`).Code; code != 403 {
		t.Fatalf("missing CSRF accepted: %d", code)
	}
	wrongOrigin := httptest.NewRequest("PATCH", "/auth/users/"+member.ID, strings.NewReader(`{"status":"active"}`))
	wrongOrigin.Header.Set("Origin", "https://other.example.test")
	wrongOrigin.Header.Set("X-CSRF-Token", payload.CSRFToken)
	wrongOrigin.AddCookie(&http.Cookie{Name: service.cookieName(sessionCookie), Value: ownerToken})
	wrongOriginResult := httptest.NewRecorder()
	mux.ServeHTTP(wrongOriginResult, wrongOrigin)
	if wrongOriginResult.Code != 403 {
		t.Fatalf("disallowed origin accepted: %d", wrongOriginResult.Code)
	}
	legacy := httptest.NewRequest("GET", "/auth/users", nil)
	legacy.Header.Set("Authorization", "Bearer hardcoded-admin-token")
	legacyResult := httptest.NewRecorder()
	mux.ServeHTTP(legacyResult, legacy)
	if legacyResult.Code != 401 {
		t.Fatalf("legacy bearer token accepted: %d", legacyResult.Code)
	}
	if code := request("PATCH", "/auth/users/"+member.ID, ownerToken, payload.CSRFToken, `{"status":"active"}`).Code; code != 204 {
		t.Fatalf("approval failed: %d", code)
	}
	approvedAgain, err := store.UpsertGoogleUser(ctx, "issuer", "member-sub", "renamed@example.test", "Renamed")
	if err != nil || approvedAgain.ID != member.ID || approvedAgain.Status != "active" {
		t.Fatalf("profile update changed approval: %#v %v", approvedAgain, err)
	}
	_, err = store.DB.Pool.Exec(ctx, `INSERT INTO auth_users(id,issuer,subject,role,status,created_at) VALUES ('duplicate','issuer','member-sub','admin','pending',now())`)
	if err == nil {
		t.Fatal("duplicate Google identity inserted")
	}
	if code := request("GET", "/auth/users", memberToken, "", "").Code; code != 403 {
		t.Fatalf("admin listed users: %d", code)
	}
	if code := request("GET", "/auth/users", ownerToken, "", "").Code; code != 200 {
		t.Fatalf("owner list: %d", code)
	}
	if code := request("PATCH", "/auth/users/"+owner.ID, ownerToken, payload.CSRFToken, `{"status":"disabled"}`).Code; code != 403 {
		t.Fatalf("owner disabled: %d", code)
	}
	if code := request("PATCH", "/auth/users/"+member.ID, ownerToken, payload.CSRFToken, `{"status":"disabled"}`).Code; code != 204 {
		t.Fatalf("disable failed: %d", code)
	}
	if _, _, err := store.Resolve(ctx, memberToken); !errors.Is(err, ErrNoSession) {
		t.Fatalf("disabled session remained: %v", err)
	}
	_, err = store.DB.Pool.Exec(ctx, `UPDATE auth_sessions SET expires_at=$2 WHERE hash=$1`, digest(ownerToken), time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Resolve(ctx, ownerToken); !errors.Is(err, ErrNoSession) {
		t.Fatalf("expired session worked before TTL cleanup: %v", err)
	}
}
