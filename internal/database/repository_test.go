package database

import (
	"os"
	"testing"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	path := t.TempDir() + "/test.db"
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUpsertArtist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	id, err := db.UpsertArtist(&Artist{
		SpotifyID: "spotify:artist:test123",
		Name:      "Test Artist",
		Genres:    `["electronic"]`,
		Popularity: 75,
	})
	if err != nil {
		t.Fatalf("upsert artist: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	artist, err := db.GetArtistBySpotifyID("spotify:artist:test123")
	if err != nil {
		t.Fatalf("get artist: %v", err)
	}
	if artist.Name != "Test Artist" {
		t.Fatalf("expected Test Artist, got %s", artist.Name)
	}

	dupeID, err := db.UpsertArtist(&Artist{
		SpotifyID: "spotify:artist:test123",
		Name:      "Test Artist Updated",
		Genres:    `["electronic","synthwave"]`,
		Popularity: 80,
	})
	if err != nil {
		t.Fatalf("upsert duplicate artist: %v", err)
	}
	if dupeID != id {
		t.Fatalf("expected same id %d, got %d", id, dupeID)
	}
}

func TestUpsertAlbum(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	id, err := db.UpsertAlbum(&Album{
		SpotifyID:   "spotify:album:test456",
		Name:        "Test Album",
		ReleaseDate: "2024-01-15",
		TotalTracks: 10,
		CoverURL:    "https://example.com/cover.jpg",
	})
	if err != nil {
		t.Fatalf("upsert album: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	album, err := db.GetAlbumBySpotifyID("spotify:album:test456")
	if err != nil {
		t.Fatalf("get album: %v", err)
	}
	if album.Name != "Test Album" {
		t.Fatalf("expected Test Album, got %s", album.Name)
	}
}

func TestInsertPlayEvent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	artistID, _ := db.UpsertArtist(&Artist{
		SpotifyID: "spotify:artist:ae1",
		Name:      "Artist 1",
	})
	albumID, _ := db.UpsertAlbum(&Album{
		SpotifyID: "spotify:album:al1",
		Name:      "Album 1",
	})
	trackID, _ := db.UpsertTrack(&Track{
		SpotifyID:  "spotify:track:tr1",
		Name:       "Track 1",
		DurationMS: 240000,
		AlbumID:    albumID,
	})

	db.InsertTrackArtist(trackID, artistID)
	db.InsertAlbumArtist(albumID, artistID)

	peID, err := db.InsertPlayEvent(&PlayEvent{
		TrackID:  trackID,
		PlayedAt: "2024-03-15T14:30:00Z",
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("insert play event: %v", err)
	}
	if peID == 0 {
		t.Fatal("expected non-zero play event id")
	}

	dupeID, err := db.InsertPlayEvent(&PlayEvent{
		TrackID:  trackID,
		PlayedAt: "2024-03-15T14:30:00Z",
		Source:   "test",
	})
	if err != nil {
		t.Fatalf("insert duplicate play event: %v", err)
	}
	if dupeID != 0 {
		t.Fatal("expected 0 for duplicate (ON CONFLICT DO NOTHING)")
	}

	plays, err := db.GetRecentPlays(10)
	if err != nil {
		t.Fatalf("get recent plays: %v", err)
	}
	if len(plays) != 1 {
		t.Fatalf("expected 1 play, got %d", len(plays))
	}
	if plays[0].Track.Name != "Track 1" {
		t.Fatalf("expected Track 1, got %s", plays[0].Track.Name)
	}
}

func TestMigration(t *testing.T) {
	path := t.TempDir() + "/migrate.db"
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	if err := db.Migrate(); err != nil {
		t.Fatalf("second migrate (idempotent): %v", err)
	}

	var version int
	err = db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if version != 3 {
		t.Fatalf("expected version 3, got %d", version)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
