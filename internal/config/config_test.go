package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	os.Setenv("SPOTIFY_CLIENT_ID", "test-id")
	os.Setenv("SPOTIFY_CLIENT_SECRET", "test-secret")
	os.Setenv("SPOTIFY_REFRESH_TOKEN", "test-token")
	os.Setenv("DB_PATH", "/tmp/test.db")
	os.Setenv("BIND_ADDR", ":9090")
	defer func() {
		os.Unsetenv("SPOTIFY_CLIENT_ID")
		os.Unsetenv("SPOTIFY_CLIENT_SECRET")
		os.Unsetenv("SPOTIFY_REFRESH_TOKEN")
		os.Unsetenv("DB_PATH")
		os.Unsetenv("BIND_ADDR")
	}()

	c := Load()
	if c.SpotifyClientID != "test-id" {
		t.Fatalf("expected test-id, got %s", c.SpotifyClientID)
	}
	if c.SpotifyClientSecret != "test-secret" {
		t.Fatalf("expected test-secret, got %s", c.SpotifyClientSecret)
	}
	if c.SpotifyRefreshToken != "test-token" {
		t.Fatalf("expected test-token, got %s", c.SpotifyRefreshToken)
	}
	if c.DBPath != "/tmp/test.db" {
		t.Fatalf("expected /tmp/test.db, got %s", c.DBPath)
	}
	if c.BindAddr != ":9090" {
		t.Fatalf("expected :9090, got %s", c.BindAddr)
	}
}

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("DB_PATH")
	os.Unsetenv("BIND_ADDR")
	os.Unsetenv("SPOTIFY_CLIENT_ID")
	os.Unsetenv("SPOTIFY_CLIENT_SECRET")
	os.Unsetenv("SPOTIFY_REFRESH_TOKEN")

	c := Load()
	if c.DBPath != "music.db" {
		t.Fatalf("expected default music.db, got %s", c.DBPath)
	}
	if c.BindAddr != ":8080" {
		t.Fatalf("expected default :8080, got %s", c.BindAddr)
	}
	if c.SpotifyClientID != "" {
		t.Fatalf("expected empty client id, got %s", c.SpotifyClientID)
	}
}

func TestEnvOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		def      string
		expected string
	}{
		{"env set", "MY_TEST_VAR", "hello", "fallback", "hello"},
		{"env empty", "MY_TEST_VAR_EMPTY", "", "fallback", "fallback"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != "" {
				os.Setenv(tc.key, tc.value)
				defer os.Unsetenv(tc.key)
			} else {
				os.Unsetenv(tc.key)
			}
			got := envOrDefault(tc.key, tc.def)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
