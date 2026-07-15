package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &DB{db}, nil
}

func (db *DB) Migrate() error {
	migrations := []struct {
		version int
		name    string
		sql     string
	}{
		{1, "init", Migration1},
		{2, "dedup_index", Migration2},
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback()

	var currentVersion int
	err = tx.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		_, err = tx.Exec("CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')))")
		if err != nil {
			return fmt.Errorf("create schema_version: %w", err)
		}
	}

	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}
		_, err := tx.Exec(m.sql)
		if err != nil {
			return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_version (version) VALUES (?)", m.version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}

	return tx.Commit()
}

const Migration1 = `CREATE TABLE IF NOT EXISTS artists (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id  TEXT    NOT NULL UNIQUE,
    name        TEXT    NOT NULL,
    genres      TEXT,
    followers   INTEGER,
    popularity  INTEGER,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE TABLE IF NOT EXISTS albums (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id   TEXT    NOT NULL UNIQUE,
    name         TEXT    NOT NULL,
    release_date TEXT,
    total_tracks INTEGER,
    cover_url    TEXT,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE TABLE IF NOT EXISTS tracks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    spotify_id   TEXT    NOT NULL UNIQUE,
    name         TEXT    NOT NULL,
    duration_ms  INTEGER NOT NULL,
    explicit     BOOLEAN NOT NULL DEFAULT 0,
    disc_number  INTEGER,
    track_number INTEGER,
    popularity   INTEGER,
    preview_url  TEXT,
    album_id     INTEGER REFERENCES albums(id) ON DELETE SET NULL,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE TABLE IF NOT EXISTS play_events (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id  INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    played_at TEXT    NOT NULL,
    device    TEXT,
    shuffle   BOOLEAN,
    repeat    TEXT,
    context   TEXT,
    source    TEXT
);
CREATE TABLE IF NOT EXISTS track_artists (
    track_id  INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    PRIMARY KEY (track_id, artist_id)
);
CREATE TABLE IF NOT EXISTS album_artists (
    album_id  INTEGER NOT NULL REFERENCES albums(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    PRIMARY KEY (album_id, artist_id)
);
CREATE INDEX IF NOT EXISTS idx_play_events_played_at ON play_events(played_at);
CREATE INDEX IF NOT EXISTS idx_play_events_track_id  ON play_events(track_id);
CREATE INDEX IF NOT EXISTS idx_tracks_spotify_id     ON tracks(spotify_id);
CREATE INDEX IF NOT EXISTS idx_artists_spotify_id    ON artists(spotify_id);
CREATE INDEX IF NOT EXISTS idx_albums_spotify_id     ON albums(spotify_id);`

const Migration2 = `CREATE UNIQUE INDEX IF NOT EXISTS idx_play_events_dedup
    ON play_events(track_id, played_at);`
