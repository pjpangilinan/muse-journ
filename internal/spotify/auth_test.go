package spotify

import (
	"testing"
)

func TestNewOAuthConfig(t *testing.T) {
	cfg := NewOAuthConfig("cid", "csecret", "http://localhost:8080/callback")
	if cfg.ClientID != "cid" {
		t.Fatalf("expected cid, got %s", cfg.ClientID)
	}
	if len(cfg.Scopes) != 1 || cfg.Scopes[0] != "user-read-recently-played" {
		t.Fatalf("unexpected scopes: %v", cfg.Scopes)
	}
}

func TestCoverURL(t *testing.T) {
	track := &Track{
		ID:   "test",
		Name: "Test",
		Album: Album{
			Images: []Image{
				{URL: "https://example.com/cover.jpg", Width: 640, Height: 640},
			},
		},
	}
	if url := track.CoverURL(); url != "https://example.com/cover.jpg" {
		t.Fatalf("expected cover url, got %s", url)
	}

	noImg := &Track{Name: "No image"}
	if url := noImg.CoverURL(); url != "" {
		t.Fatalf("expected empty url, got %s", url)
	}
}

func TestNormalize(t *testing.T) {
	collector := NewCollector("cid", "csecret", "rtoken")

	item := PlayHistoryItem{
		Track: Track{
			ID:         "track123",
			Name:       "Test Song",
			DurationMS: 200000,
			Explicit:   false,
			Album: Album{
				ID:          "album123",
				Name:        "Test Album",
				ReleaseDate: "2024-01-01",
			},
			Artists: []Artist{
				{ID: "artist123", Name: "Test Artist", Popularity: 80},
			},
		},
		PlayedAt: "2024-03-15T14:30:00Z",
	}

	play := collector.normalize(item)
	if play.TrackName != "Test Song" {
		t.Fatalf("expected Test Song, got %s", play.TrackName)
	}
	if play.PlayedAt != "2024-03-15T14:30:00Z" {
		t.Fatalf("wrong played_at: %s", play.PlayedAt)
	}
	if len(play.ArtistIDs) != 1 || play.ArtistIDs[0] != "artist123" {
		t.Fatalf("wrong artist ids: %v", play.ArtistIDs)
	}
}
