package config

import (
	"os"
	"strconv"
)

type Config struct {
	SpotifyClientID     string
	SpotifyClientSecret string
	SpotifyRefreshToken string
	DBPath              string
	Port                int
}

func Load() *Config {
	return &Config{
		SpotifyClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		SpotifyClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		SpotifyRefreshToken: os.Getenv("SPOTIFY_REFRESH_TOKEN"),
		DBPath:              envOrDefault("DB_PATH", "music.db"),
		Port:                envIntOrDefault("PORT", 8080),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
