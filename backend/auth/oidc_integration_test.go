//go:build integration

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	jose "github.com/go-jose/go-jose/v4"
	jwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/user/go-expo-sample/backend/testutils"
	"golang.org/x/oauth2"
)

func TestGoogleCallbackWithFakeOIDCProvider(t *testing.T) {
	store := Store{DB: testutils.SetupPostgres(t)}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var providerURL string
	var currentToken string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{"issuer": providerURL, "authorization_endpoint": providerURL + "/authorize", "token_endpoint": providerURL + "/token", "jwks_uri": providerURL + "/keys", "response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"}, "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test-key", Use: "sig", Algorithm: "RS256"}}})
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "temporary", "token_type": "Bearer", "expires_in": 300, "id_token": currentToken})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	providerURL = provider.URL
	oidcProvider, err := oidc.NewProvider(context.Background(), provider.URL)
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{
		Config:   Config{Mode: "google", Environment: "production", BaseURL: "https://events.example.test", ClientID: "test-client", Secret: "test-secret"},
		Store:    store,
		OAuth:    &oauth2.Config{ClientID: "test-client", ClientSecret: "test-secret", Endpoint: oidcProvider.Endpoint(), RedirectURL: "https://events.example.test/auth/google/callback", Scopes: []string{"openid", "email", "profile"}},
		Verifier: oidcProvider.Verifier(&oidc.Config{ClientID: "test-client"}),
	}
	mux := http.NewServeMux()
	service.Register(mux)
	sign := func(claims map[string]any, signingKey *rsa.PrivateKey) string {
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: signingKey}, (&jose.SignerOptions{}).WithHeader(jose.HeaderKey("kid"), "test-key"))
		if err != nil {
			t.Fatal(err)
		}
		value, err := jwt.Signed(signer).Claims(claims).Serialize()
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	callback := func(name string, mutate func(map[string]any)) int {
		t.Helper()
		state, browser, nonce := "state-"+name, "browser-"+name, "nonce-"+name
		err := store.DB.CreateAuthLogin(context.Background(), LoginTransaction{Hash: digest(state), Browser: digest(browser), Nonce: nonce, Verifier: oauth2.GenerateVerifier(), ExpiresAt: time.Now().Add(time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		claims := map[string]any{"iss": providerURL, "sub": "user-1", "aud": "test-client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "nonce": nonce, "email": "user@example.test", "email_verified": true, "name": "Test User"}
		if mutate != nil {
			mutate(claims)
		}
		signingKey := key
		if name == "signature" {
			signingKey = wrongKey
		}
		currentToken = sign(claims, signingKey)
		r := httptest.NewRequest("GET", "/auth/google/callback?code=authorization-code&state="+url.QueryEscape(state), nil)
		r.AddCookie(&http.Cookie{Name: service.cookieName(loginCookie), Value: browser})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if name == "valid" {
			if w.Code != 302 || w.Header().Get("Location") != "/pending" {
				t.Fatalf("valid callback: %d %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Header().Get("Set-Cookie"), "__Host-") {
				t.Fatal("missing secure cookie")
			}
			second := httptest.NewRecorder()
			mux.ServeHTTP(second, r)
			if second.Code != 400 {
				t.Fatalf("replayed callback accepted: %d", second.Code)
			}
		}
		return w.Code
	}
	if code := callback("valid", nil); code != 302 {
		t.Fatal(code)
	}
	if code := callback("verified-string", func(c map[string]any) { c["email_verified"] = "true" }); code != 302 {
		t.Fatalf("Google string email verification claim rejected: %d", code)
	}
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"nonce", func(c map[string]any) { c["nonce"] = "wrong" }},
		{"issuer", func(c map[string]any) { c["iss"] = "https://other.example.test" }},
		{"audience", func(c map[string]any) { c["aud"] = "other-client" }},
		{"expired", func(c map[string]any) { c["exp"] = time.Now().Add(-time.Minute).Unix() }},
		{"email", func(c map[string]any) { c["email_verified"] = false }},
	} {
		if code := callback(tc.name, tc.mutate); code != 401 {
			t.Errorf("%s: got %d", tc.name, code)
		}
	}
	if code := callback("signature", nil); code != 401 {
		t.Errorf("signature: got %d", code)
	}

	start := httptest.NewRecorder()
	mux.ServeHTTP(start, httptest.NewRequest("GET", "/auth/google/start", nil))
	if start.Code != 302 {
		t.Fatalf("start: %d", start.Code)
	}
	authURL, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := authURL.Query().Get("state")
	if state == "" || authURL.Query().Get("nonce") == "" || authURL.Query().Get("code_challenge") == "" || authURL.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("missing state, nonce, or PKCE challenge: %s", authURL.String())
	}
	cookies := start.Result().Cookies()
	var browser *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == service.cookieName(loginCookie) {
			browser = cookie
		}
	}
	if browser == nil || !browser.Secure {
		t.Fatal("missing secure browser binding")
	}
	badBrowser := httptest.NewRequest("GET", "/auth/google/callback?code=code&state="+url.QueryEscape(state), nil)
	badBrowser.AddCookie(&http.Cookie{Name: browser.Name, Value: "wrong-browser"})
	badResult := httptest.NewRecorder()
	mux.ServeHTTP(badResult, badBrowser)
	if badResult.Code != 400 {
		t.Fatalf("wrong browser accepted: %d", badResult.Code)
	}
	missingState := httptest.NewRequest("GET", "/auth/google/callback?code=code&state=unknown", nil)
	missingState.AddCookie(browser)
	missingResult := httptest.NewRecorder()
	mux.ServeHTTP(missingResult, missingState)
	if missingResult.Code != 400 {
		t.Fatalf("wrong state accepted: %d", missingResult.Code)
	}
	_, err = store.DB.Pool.Exec(context.Background(), `UPDATE auth_logins SET expires_at=$2 WHERE hash=$1`, digest(state), time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	expired := httptest.NewRequest("GET", "/auth/google/callback?code=code&state="+url.QueryEscape(state), nil)
	expired.AddCookie(browser)
	expiredResult := httptest.NewRecorder()
	mux.ServeHTTP(expiredResult, expired)
	if expiredResult.Code != 400 {
		t.Fatalf("expired transaction accepted: %d", expiredResult.Code)
	}
	cancelled := httptest.NewRecorder()
	mux.ServeHTTP(cancelled, httptest.NewRequest("GET", "/auth/google/callback?error=access_denied", nil))
	if cancelled.Code != 302 || cancelled.Header().Get("Location") != "/login?error=cancelled" {
		t.Fatalf("cancelled flow: %d", cancelled.Code)
	}
}
