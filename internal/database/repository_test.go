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

func seedTestData(t *testing.T, db *DB) (artistID, albumID, trackID int64) {
	t.Helper()
	artistID, err := db.UpsertArtist(&Artist{
		SpotifyID: "spotify:artist:test1", Name: "Test Artist",
		Genres: `["rock"]`, Popularity: 80,
	})
	if err != nil {
		t.Fatalf("seed artist: %v", err)
	}
	albumID, err = db.UpsertAlbum(&Album{
		SpotifyID: "spotify:album:test1", Name: "Test Album",
		ReleaseDate: "2024-01-01", TotalTracks: 10,
	})
	if err != nil {
		t.Fatalf("seed album: %v", err)
	}
	trackID, err = db.UpsertTrack(&Track{
		SpotifyID: "spotify:track:test1", Name: "Test Track",
		DurationMS: 240000, AlbumID: albumID,
	})
	if err != nil {
		t.Fatalf("seed track: %v", err)
	}
	db.InsertTrackArtist(trackID, artistID)
	db.InsertAlbumArtist(albumID, artistID)
	return
}

func TestOpenAndMigrate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var version int
	err := db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if version != 3 {
		t.Fatalf("expected version 3, got %d", version)
	}

	// Second migrate should be idempotent
	if err := db.Migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestUpsertArtist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	id, err := db.UpsertArtist(&Artist{
		SpotifyID: "spotify:artist:test123", Name: "Test Artist",
		Genres: `["electronic"]`, Popularity: 75,
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

	// Upsert same spotify_id should update
	id2, err := db.UpsertArtist(&Artist{
		SpotifyID: "spotify:artist:test123", Name: "Updated Artist",
		Genres: `["synthwave"]`, Popularity: 85,
	})
	if err != nil {
		t.Fatalf("upsert duplicate: %v", err)
	}
	if id2 != id {
		t.Fatalf("expected same id %d, got %d", id, id2)
	}
	artist, _ = db.GetArtistBySpotifyID("spotify:artist:test123")
	if artist.Name != "Updated Artist" {
		t.Fatalf("expected Updated Artist, got %s", artist.Name)
	}

	// Get nonexistent
	artist, err = db.GetArtistBySpotifyID("nope")
	if err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
	if artist != nil {
		t.Fatal("expected nil for nonexistent")
	}
}

func TestUpsertAlbum(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	id, err := db.UpsertAlbum(&Album{
		SpotifyID: "spotify:album:test456", Name: "Test Album",
		ReleaseDate: "2024-06-15", TotalTracks: 12,
		CoverURL: "https://example.com/cover.jpg",
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

func TestUpsertTrack(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Need album first
	albumID, _ := db.UpsertAlbum(&Album{
		SpotifyID: "spotify:album:trk1", Name: "Track Album",
	})

	id, err := db.UpsertTrack(&Track{
		SpotifyID: "spotify:track:trk1", Name: "Test Track",
		DurationMS: 200000, AlbumID: albumID,
	})
	if err != nil {
		t.Fatalf("upsert track: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero id")
	}

	track, err := db.GetTrackBySpotifyID("spotify:track:trk1")
	if err != nil {
		t.Fatalf("get track: %v", err)
	}
	if track.Name != "Test Track" {
		t.Fatalf("expected Test Track, got %s", track.Name)
	}
	if track.AlbumID != albumID {
		t.Fatalf("expected album_id %d, got %d", albumID, track.AlbumID)
	}
}

func TestPlayEventLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, _, trackID := seedTestData(t, db)

	peID, err := db.InsertPlayEvent(&PlayEvent{
		TrackID: trackID, PlayedAt: "2024-03-15T14:30:00Z",
		Source: "test", Device: "browser",
	})
	if err != nil {
		t.Fatalf("insert play: %v", err)
	}
	if peID == 0 {
		t.Fatal("expected non-zero id")
	}

	// Duplicate should return 0 (ON CONFLICT DO NOTHING)
	dupeID, err := db.InsertPlayEvent(&PlayEvent{
		TrackID: trackID, PlayedAt: "2024-03-15T14:30:00Z",
		Source: "test",
	})
	if err != nil {
		t.Fatalf("insert duplicate: %v", err)
	}
	if dupeID != 0 {
		t.Fatal("expected 0 for duplicate")
	}

	// Get recent plays
	plays, err := db.GetRecentPlays(10)
	if err != nil {
		t.Fatalf("get recent plays: %v", err)
	}
	if len(plays) != 1 {
		t.Fatalf("expected 1 play, got %d", len(plays))
	}
	if plays[0].Track.Name != "Test Track" {
		t.Fatalf("expected Test Track, got %s", plays[0].Track.Name)
	}
	if plays[0].Artists != "Test Artist" {
		t.Fatalf("expected Test Artist, got %s", plays[0].Artists)
	}
	if plays[0].Album == nil || plays[0].Album.Name != "Test Album" {
		t.Fatalf("expected Test Album album")
	}

	// Get recent plays with date range
	plays, err = db.GetRecentPlaysRange(10, 0, "2024-03-15", "2024-03-15")
	if err != nil {
		t.Fatalf("get plays range: %v", err)
	}
	if len(plays) != 1 {
		t.Fatalf("expected 1 play in range, got %d", len(plays))
	}

	// Wrong date range should return 0
	plays, err = db.GetRecentPlaysRange(10, 0, "2025-01-01", "2025-01-01")
	if err != nil {
		t.Fatalf("get plays empty range: %v", err)
	}
	if len(plays) != 0 {
		t.Fatalf("expected 0 plays, got %d", len(plays))
	}

	// Offset should skip the first play
	plays, err = db.GetRecentPlaysRange(10, 1, "", "")
	if err != nil {
		t.Fatalf("get plays offset: %v", err)
	}
	if len(plays) != 0 {
		t.Fatalf("expected 0 plays with offset 1 (only 1 play total), got %d", len(plays))
	}

	// Play count
	count, err := db.GetPlayCount(trackID)
	if err != nil {
		t.Fatalf("play count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 play count, got %d", count)
	}
}

func TestMultiplePlaysAndPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, _, trackID := seedTestData(t, db)

	// Insert 15 plays
	times := []string{
		"2024-03-15T10:00:00Z",
		"2024-03-15T11:00:00Z",
		"2024-03-15T12:00:00Z",
		"2024-03-15T13:00:00Z",
		"2024-03-15T14:00:00Z",
		"2024-03-15T15:00:00Z",
		"2024-03-15T16:00:00Z",
		"2024-03-15T17:00:00Z",
		"2024-03-16T10:00:00Z",
		"2024-03-16T11:00:00Z",
		"2024-03-16T12:00:00Z",
		"2024-03-16T13:00:00Z",
		"2024-03-17T10:00:00Z",
		"2024-03-17T11:00:00Z",
		"2024-03-17T12:00:00Z",
	}
	for _, pt := range times {
		_, err := db.InsertPlayEvent(&PlayEvent{
			TrackID: trackID, PlayedAt: pt, Source: "test",
		})
		if err != nil {
			t.Fatalf("insert play at %s: %v", pt, err)
		}
	}

	// Get all plays (should be 15)
	plays, err := db.GetRecentPlays(100)
	if err != nil {
		t.Fatalf("get all: %v", err)
	}
	if len(plays) != 15 {
		t.Fatalf("expected 15 plays, got %d", len(plays))
	}
}

func TestEmptyDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	plays, err := db.GetRecentPlays(10)
	if err != nil {
		t.Fatalf("get plays empty: %v", err)
	}
	if len(plays) != 0 {
		t.Fatalf("expected 0 plays, got %d", len(plays))
	}

	count, err := db.GetPlayCount(999)
	if err != nil {
		t.Fatalf("play count empty: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 count, got %d", count)
	}

	artist, err := db.GetArtistBySpotifyID("nonexistent")
	if err != nil {
		t.Fatalf("get artist nonexistent: %v", err)
	}
	if artist != nil {
		t.Fatal("expected nil artist")
	}
}

func TestArtistRelationships(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	artistID1, _ := db.UpsertArtist(&Artist{
		SpotifyID: "a1", Name: "Artist 1",
	})
	artistID2, _ := db.UpsertArtist(&Artist{
		SpotifyID: "a2", Name: "Artist 2",
	})
	albumID, _ := db.UpsertAlbum(&Album{
		SpotifyID: "al1", Name: "Album 1",
	})
	trackID, _ := db.UpsertTrack(&Track{
		SpotifyID: "t1", Name: "Track 1", DurationMS: 100000, AlbumID: albumID,
	})

	if err := db.InsertTrackArtist(trackID, artistID1); err != nil {
		t.Fatalf("link track-artist 1: %v", err)
	}
	if err := db.InsertTrackArtist(trackID, artistID2); err != nil {
		t.Fatalf("link track-artist 2: %v", err)
	}
	if err := db.InsertAlbumArtist(albumID, artistID1); err != nil {
		t.Fatalf("link album-artist: %v", err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
