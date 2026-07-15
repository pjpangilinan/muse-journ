package app

import (
	"fmt"
	"log"
	"time"

	"github.com/pjpangilinan/muse-journ/internal/analytics"
	"github.com/pjpangilinan/muse-journ/internal/config"
	"github.com/pjpangilinan/muse-journ/internal/database"
	"github.com/pjpangilinan/muse-journ/internal/reports"
	"github.com/pjpangilinan/muse-journ/internal/spotify"
)

func RunCollector(cfg *config.Config) error {
	if cfg.SpotifyClientID == "" || cfg.SpotifyClientSecret == "" || cfg.SpotifyRefreshToken == "" {
		return fmt.Errorf("SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET, and SPOTIFY_REFRESH_TOKEN required")
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	collector := spotify.NewCollector(cfg.SpotifyClientID, cfg.SpotifyClientSecret, cfg.SpotifyRefreshToken)

	var after string
	var lastPlayedAt string
	row := db.QueryRow("SELECT played_at FROM play_events ORDER BY played_at DESC LIMIT 1")
	if err := row.Scan(&lastPlayedAt); err == nil {
		t, err := time.Parse(time.RFC3339, lastPlayedAt)
		if err == nil {
			after = fmt.Sprintf("%d", t.UnixMilli())
		}
	}

	plays, err := collector.Collect(after)
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}

	inserted := 0
	for _, play := range plays {
		if len(play.ArtistIDs) == 0 || len(play.ArtistNames) == 0 || len(play.ArtistPopularities) == 0 {
			log.Printf("skip play with no artists: %s at %s", play.TrackName, play.PlayedAt)
			continue
		}
		artistID, err := db.UpsertArtist(&database.Artist{
			SpotifyID:  play.ArtistIDs[0],
			Name:       play.ArtistNames[0],
			Genres:     joinGenres(play.ArtistGenres),
			Popularity: play.ArtistPopularities[0],
		})
		if err != nil {
			log.Printf("upsert artist: %v", err)
			continue
		}

		albumID, err := db.UpsertAlbum(&database.Album{
			SpotifyID:   play.AlbumSpotifyID,
			Name:        play.AlbumName,
			ReleaseDate: play.ReleaseDate,
			TotalTracks: play.TotalTracks,
			CoverURL:    play.CoverURL,
		})
		if err != nil {
			log.Printf("upsert album: %v", err)
			continue
		}

		trackID, err := db.UpsertTrack(&database.Track{
			SpotifyID:   play.TrackSpotifyID,
			Name:        play.TrackName,
			DurationMS:  play.DurationMS,
			Explicit:    play.Explicit,
			DiscNumber:  play.DiscNumber,
			TrackNumber: play.TrackNumber,
			Popularity:  play.Popularity,
			PreviewURL:  play.PreviewURL,
			AlbumID:     albumID,
		})
		if err != nil {
			log.Printf("upsert track: %v", err)
			continue
		}

		if err := db.InsertTrackArtist(trackID, artistID); err != nil {
			log.Printf("link track-artist: %v", err)
		}
		if err := db.InsertAlbumArtist(albumID, artistID); err != nil {
			log.Printf("link album-artist: %v", err)
		}

		_, err = db.InsertPlayEvent(&database.PlayEvent{
			TrackID:  trackID,
			PlayedAt: play.PlayedAt,
			Source:   "collector",
		})
		if err != nil {
			log.Printf("insert play event: %v", err)
			continue
		}
		inserted++
	}

	log.Printf("collector: %d new plays inserted", inserted)

	analyticsDB := analytics.New(db.DB)
	reportGen := reports.New(analyticsDB)

	today := time.Now().UTC().Format("2006-01-02")
	if report, err := reportGen.DailyReport(today); err == nil {
		fmt.Println(report)
	}

	currentMonth := time.Now().UTC().Format("2006-01")
	if report, err := reportGen.MonthlyReport(currentMonth); err == nil {
		fmt.Println(report)
	}

	return nil
}
