package config

import (
	"os"
)

type Config struct {
	SpotifyClientID     string
	SpotifyClientSecret string
	SpotifyRefreshToken string
	DBPath              string
	BindAddr            string
}

func Load() *Config {
	return &Config{
		SpotifyClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		SpotifyClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		SpotifyRefreshToken: os.Getenv("SPOTIFY_REFRESH_TOKEN"),
		DBPath:              envOrDefault("DB_PATH", "music.db"),
		BindAddr:            envOrDefault("BIND_ADDR", ":8080"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
