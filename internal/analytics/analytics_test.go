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
		`CREATE TABLE IF NOT EXISTS play_events (id INTEGER PRIMARY KEY AUTOINCREMENT, track_id INTEGER, played_at TEXT, source TEXT)`,
		`CREATE TABLE IF NOT EXISTS track_artists (track_id INTEGER, artist_id INTEGER, PRIMARY KEY(track_id, artist_id))`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	seedData(t, db)
	return db
}

func seedData(t *testing.T, db *sql.DB) {
	t.Helper()
	db.Exec(`INSERT OR IGNORE INTO artists (id, spotify_id, name) VALUES (1, 'a1', 'Artist 1'), (2, 'a2', 'Artist 2')`)
	db.Exec(`INSERT OR IGNORE INTO albums (id, spotify_id, name, total_tracks) VALUES (1, 'al1', 'Album 1', 10)`)
	db.Exec(`INSERT OR IGNORE INTO tracks (id, spotify_id, name, duration_ms, album_id) VALUES (1, 't1', 'Track 1', 240000, 1), (2, 't2', 'Track 2', 180000, 1)`)
	db.Exec(`INSERT OR IGNORE INTO play_events (track_id, played_at, source) VALUES (1, '2024-03-15T10:00:00Z', 'test'), (1, '2024-03-15T11:00:00Z', 'test'), (2, '2024-03-15T12:00:00Z', 'test')`)
	db.Exec(`INSERT OR IGNORE INTO track_artists (track_id, artist_id) VALUES (1, 1), (2, 2)`)
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
	// 240000+240000+180000 = 660000ms = 11 min
	if s.ListeningMin != 11 {
		t.Fatalf("expected 11 minutes, got %d", s.ListeningMin)
	}
	if s.UniqueTracks != 2 {
		t.Fatalf("expected 2 unique tracks, got %d", s.UniqueTracks)
	}
	if s.TopArtist != "Artist 1" {
		t.Fatalf("expected Artist 1, got %s", s.TopArtist)
	}
	if s.TopTrack != "Track 1" {
		t.Fatalf("expected Track 1, got %s", s.TopTrack)
	}
}

func TestSummaryNoData(t *testing.T) {
	a := New(setupTestDB(t))
	s, err := a.DailySummary("2099-01-01")
	if err != nil {
		t.Fatalf("daily no data: %v", err)
	}
	if s.TotalPlays != 0 {
		t.Fatalf("expected 0 plays, got %d", s.TotalPlays)
	}
	if s.TopArtist != "" {
		t.Fatalf("expected empty top artist, got %s", s.TopArtist)
	}
}

func TestMonthlySummary(t *testing.T) {
	a := New(setupTestDB(t))
	s, err := a.MonthlySummary("2024-03")
	if err != nil {
		t.Fatalf("monthly: %v", err)
	}
	if s.TotalPlays != 3 {
		t.Fatalf("expected 3 plays, got %d", s.TotalPlays)
	}
	if s.UniqueArtists != 2 {
		t.Fatalf("expected 2 artists, got %d", s.UniqueArtists)
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
	if artists[0].SpotifyID == "" {
		t.Fatal("expected spotify_id in artist ranking")
	}
}

func TestTopArtistsLimit(t *testing.T) {
	a := New(setupTestDB(t))
	artists, err := a.TopArtists(1)
	if err != nil {
		t.Fatalf("top artists limited: %v", err)
	}
	if len(artists) != 1 {
		t.Fatalf("expected 1 artist, got %d", len(artists))
	}
}

func TestTotalStats(t *testing.T) {
	a := New(setupTestDB(t))
	totalPlays, totalMinutes, err := a.TotalStats()
	if err != nil {
		t.Fatalf("total stats: %v", err)
	}
	if totalPlays != 3 {
		t.Fatalf("expected 3 plays, got %d", totalPlays)
	}
	// 240000+240000+180000 = 660000ms = 11 min
	if totalMinutes != 11 {
		t.Fatalf("expected 11 min, got %d", totalMinutes)
	}
}

func TestListeningStreak(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Single day
	db.Exec("DELETE FROM play_events")
	db.Exec("INSERT INTO play_events (track_id, played_at) VALUES (1, '2024-03-15T10:00:00Z')")

	a := New(db)
	streak, err := a.ListeningStreak()
	if err != nil {
		t.Fatalf("streak: %v", err)
	}
	if streak != 1 {
		t.Fatalf("expected 1 day streak, got %d", streak)
	}

	// Two consecutive days
	db.Exec("INSERT INTO play_events (track_id, played_at) VALUES (1, '2024-03-14T10:00:00Z')")
	streak, _ = a.ListeningStreak()
	if streak != 2 {
		t.Fatalf("expected 2 day streak, got %d", streak)
	}

	// Gap should reset streak
	db.Exec("INSERT INTO play_events (track_id, played_at) VALUES (1, '2024-03-12T10:00:00Z')")
	streak, _ = a.ListeningStreak()
	if streak != 2 {
		t.Fatalf("expected 2 day streak (gap resets), got %d", streak)
	}
}

func TestHourlyDistribution(t *testing.T) {
	a := New(setupTestDB(t))
	hours, err := a.HourlyDistribution()
	if err != nil {
		t.Fatalf("hourly: %v", err)
	}
	if len(hours) == 0 {
		t.Fatal("expected at least 1 hour")
	}
	// Plays at 10, 11, 12 -> 3 different hours
	if len(hours) != 3 {
		t.Fatalf("expected 3 hours, got %d", len(hours))
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
