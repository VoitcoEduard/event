package auth

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Environment string
	Mode        string
	BaseURL     string
	ClientID    string
	Secret      string
	DevPassword string
}

func ConfigFromEnv() (Config, error) {
	c := Config{
		Environment: valueOr("APP_ENV", "production"),
		Mode:        valueOr("AUTH_MODE", "google"),
		BaseURL:     strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/"),
		ClientID:    os.Getenv("GOOGLE_CLIENT_ID"),
		Secret:      os.Getenv("GOOGLE_CLIENT_SECRET"),
		DevPassword: os.Getenv("DEV_ADMIN_PASSWORD"),
	}
	if c.Environment != "production" && c.Environment != "development" {
		return c, errors.New("APP_ENV must be production or development")
	}
	if c.Mode != "google" && c.Mode != "dev" {
		return c, errors.New("AUTH_MODE must be google or dev")
	}
	if c.Environment == "production" && c.Mode != "google" {
		return c, errors.New("development authentication is forbidden in production")
	}
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return c, errors.New("PUBLIC_BASE_URL must be an origin without a path")
	}
	if c.Environment == "production" && u.Scheme != "https" {
		return c, errors.New("production PUBLIC_BASE_URL must use HTTPS")
	}
	if c.Mode == "google" && (c.ClientID == "" || c.Secret == "") {
		return c, errors.New("Google client ID and secret are required")
	}
	if c.Mode == "dev" && c.DevPassword == "" {
		return c, errors.New("DEV_ADMIN_PASSWORD is required")
	}
	return c, nil
}

func valueOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
