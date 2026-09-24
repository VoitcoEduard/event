package auth

import "testing"

func TestConfigFailsClosed(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_MODE", "dev")
	t.Setenv("PUBLIC_BASE_URL", "https://events.example.test")
	t.Setenv("DEV_ADMIN_PASSWORD", "test-password")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("development auth accepted in production")
	}
	t.Setenv("AUTH_MODE", "google")
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")
	t.Setenv("PUBLIC_BASE_URL", "http://events.example.test")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("HTTP accepted in production")
	}
	t.Setenv("PUBLIC_BASE_URL", "https://events.example.test")
	if _, err := ConfigFromEnv(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("missing Google secret accepted")
	}
}

func TestDevelopmentRequiresPassword(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_MODE", "dev")
	t.Setenv("PUBLIC_BASE_URL", "http://localhost:8081")
	t.Setenv("DEV_ADMIN_PASSWORD", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("missing password accepted")
	}
	t.Setenv("DEV_ADMIN_PASSWORD", "test-password")
	if _, err := ConfigFromEnv(); err != nil {
		t.Fatal(err)
	}
}
