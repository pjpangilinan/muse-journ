package analytics

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", t.TempDir()+"/test.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS artists (id INTEGER PRIMARY KEY AUTOINCREMENT, spotify_id TEXT UNIQUE, name TEXT, genres TEXT, popularity INTEGER, created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), updated_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')))`,
		`CREATE TABLE IF NOT EXISTS albums (id INTEGER PRIMARY KEY AUTOINCREMENT, spotify_id TEXT UNIQUE, name TEXT, total_tracks INTEGER, created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')))`,
		`CREATE TABLE IF NOT EXISTS tracks (id INTEGER PRIMARY KEY AUTOINCREMENT, spotify_id TEXT UNIQUE, name TEXT, duration_ms INTEGER, album_id INTEGER, created_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')))`,
		`CREATE TABLE IF NOT EXISTS play_events (id INTEGER PRIMARY KEY AUTOINCREMENT, track_id INTEGER, played_at TEXT, source TEXT, FOREIGN KEY(track_id) REFERENCES tracks(id))`,
		`CREATE TABLE IF NOT EXISTS track_artists (track_id INTEGER, artist_id INTEGER, PRIMARY KEY(track_id, artist_id))`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}

	db.Exec(`INSERT INTO artists (id, spotify_id, name) VALUES (1, 'a1', 'Artist 1'), (2, 'a2', 'Artist 2')`)
	db.Exec(`INSERT INTO albums (id, spotify_id, name, total_tracks) VALUES (1, 'al1', 'Album 1', 10)`)
	db.Exec(`INSERT INTO tracks (id, spotify_id, name, duration_ms, album_id) VALUES (1, 't1', 'Track 1', 240000, 1), (2, 't2', 'Track 2', 180000, 1)`)
	db.Exec(`INSERT INTO play_events (track_id, played_at, source) VALUES (1, '2024-03-15T10:00:00Z', 'test'), (1, '2024-03-15T11:00:00Z', 'test'), (2, '2024-03-15T12:00:00Z', 'test')`)
	db.Exec(`INSERT INTO track_artists (track_id, artist_id) VALUES (1, 1), (2, 2)`)

	return db
}

func TestDailySummary(t *testing.T) {
	a := New(setupTestDB(t))
	s, err := a.DailySummary("2024-03-15")
	if err != nil {
		t.Fatalf("daily summary: %v", err)
	}
	if s.TotalPlays != 3 {
		t.Fatalf("expected 3 plays, got %d", s.TotalPlays)
	}
	if s.ListeningMin != 11 {
		t.Fatalf("expected 11 minutes, got %d", s.ListeningMin)
	}
	if s.UniqueTracks != 2 {
		t.Fatalf("expected 2 unique tracks, got %d", s.UniqueTracks)
	}
}

func TestTopArtists(t *testing.T) {
	a := New(setupTestDB(t))
	artists, err := a.TopArtists(5)
	if err != nil {
		t.Fatalf("top artists: %v", err)
	}
	if len(artists) != 2 {
		t.Fatalf("expected 2 artists, got %d", len(artists))
	}
	if artists[0].ArtistName != "Artist 1" || artists[0].PlayCount != 2 {
		t.Fatalf("expected Artist 1 with 2 plays, got %s with %d", artists[0].ArtistName, artists[0].PlayCount)
	}
}

func TestNoData(t *testing.T) {
	db, _ := sql.Open("sqlite", t.TempDir()+"/empty.db")
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS artists (id INTEGER PRIMARY KEY AUTOINCREMENT, spotify_id TEXT UNIQUE, name TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS tracks (id INTEGER PRIMARY KEY AUTOINCREMENT, spotify_id TEXT UNIQUE, name TEXT, duration_ms INTEGER)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS play_events (id INTEGER PRIMARY KEY AUTOINCREMENT, track_id INTEGER, played_at TEXT, source TEXT)`)

	a := New(db)
	s, err := a.DailySummary("2024-03-15")
	if err != nil {
		t.Fatalf("daily summary empty: %v", err)
	}
	if s.TotalPlays != 0 {
		t.Fatalf("expected 0 plays, got %d", s.TotalPlays)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
