package analytics

import (
	"database/sql"
	"fmt"
	"time"
)

type DB struct {
	*sql.DB
}

func New(db *sql.DB) *DB {
	return &DB{db}
}

type DailySummary struct {
	Date           string `json:"date"`
	TotalPlays     int    `json:"total_plays"`
	ListeningMin   int    `json:"listening_minutes"`
	UniqueArtists  int    `json:"unique_artists"`
	UniqueAlbums   int    `json:"unique_albums"`
	UniqueTracks   int    `json:"unique_tracks"`
	TopArtist      string `json:"top_artist"`
	TopTrack       string `json:"top_track"`
	TopAlbum       string `json:"top_album"`
	NewDiscoveries int    `json:"new_discoveries"`
}

type MonthlySummary struct {
	YearMonth     string `json:"year_month"`
	TotalPlays    int    `json:"total_plays"`
	ListeningMin  int    `json:"listening_minutes"`
	UniqueArtists int    `json:"unique_artists"`
	UniqueAlbums  int    `json:"unique_albums"`
	UniqueTracks  int    `json:"unique_tracks"`
	TopArtist     string `json:"top_artist"`
	TopTrack      string `json:"top_track"`
}

type ArtistRanking struct {
	ArtistName string `json:"artist_name"`
	PlayCount  int    `json:"play_count"`
	SpotifyID  string `json:"spotify_id"`
}

type HourlyHeatmap struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

func (a *DB) DailySummary(date string) (*DailySummary, error) {
	s := &DailySummary{Date: date}

	err := a.QueryRow(`
		SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(t.duration_ms), 0) / 60000
		FROM play_events pe
		LEFT JOIN tracks t ON t.id = pe.track_id
		WHERE pe.played_at >= ? AND pe.played_at < ?`,
		date+"T00:00:00Z", date+"T24:00:00Z").Scan(&s.TotalPlays, &s.ListeningMin)
	if err != nil {
		return s, nil
	}

	a.QueryRow(`
		SELECT COUNT(DISTINCT ar.id) FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		LEFT JOIN track_artists ta ON ta.track_id = t.id
		LEFT JOIN artists ar ON ar.id = ta.artist_id
		WHERE pe.played_at >= ? AND pe.played_at < ?`,
		date+"T00:00:00Z", date+"T24:00:00Z").Scan(&s.UniqueArtists)

	a.QueryRow(`
		SELECT COUNT(DISTINCT t.album_id) FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		WHERE pe.played_at >= ? AND pe.played_at < ?`,
		date+"T00:00:00Z", date+"T24:00:00Z").Scan(&s.UniqueAlbums)

	a.QueryRow(`
		SELECT COUNT(DISTINCT pe.track_id) FROM play_events pe
		WHERE pe.played_at >= ? AND pe.played_at < ?`,
		date+"T00:00:00Z", date+"T24:00:00Z").Scan(&s.UniqueTracks)

	a.QueryRow(`
		SELECT COALESCE(ar.name, '') FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		LEFT JOIN track_artists ta ON ta.track_id = t.id
		LEFT JOIN artists ar ON ar.id = ta.artist_id
		WHERE pe.played_at >= ? AND pe.played_at < ?
		GROUP BY ar.id ORDER BY COUNT(*) DESC LIMIT 1`,
		date+"T00:00:00Z", date+"T24:00:00Z").Scan(&s.TopArtist)

	a.QueryRow(`
		SELECT COALESCE(t.name, '') FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		WHERE pe.played_at >= ? AND pe.played_at < ?
		GROUP BY t.id ORDER BY COUNT(*) DESC LIMIT 1`,
		date+"T00:00:00Z", date+"T24:00:00Z").Scan(&s.TopTrack)

	a.QueryRow(`
		SELECT COALESCE(al.name, '') FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		LEFT JOIN albums al ON al.id = t.album_id
		WHERE pe.played_at >= ? AND pe.played_at < ?
		GROUP BY al.id ORDER BY COUNT(*) DESC LIMIT 1`,
		date+"T00:00:00Z", date+"T24:00:00Z").Scan(&s.TopAlbum)

	return s, nil
}

func (a *DB) MonthlySummary(yearMonth string) (*MonthlySummary, error) {
	s := &MonthlySummary{YearMonth: yearMonth}

	start := yearMonth + "-01T00:00:00Z"
	t, err := time.Parse("2006-01-02T15:04:05Z", start)
	if err != nil {
		return nil, fmt.Errorf("parse month: %w", err)
	}
	end := t.AddDate(0, 1, 0).Format(time.RFC3339)

	err = a.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(t.duration_ms), 0) / 60000
		FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		WHERE pe.played_at >= ? AND pe.played_at < ?`, start, end).Scan(&s.TotalPlays, &s.ListeningMin)
	if err != nil {
		return nil, fmt.Errorf("monthly summary base: %w", err)
	}

	a.QueryRow(`SELECT COUNT(DISTINCT ar.id) FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		JOIN track_artists ta ON ta.track_id = t.id
		JOIN artists ar ON ar.id = ta.artist_id
		WHERE pe.played_at >= ? AND pe.played_at < ?`, start, end).Scan(&s.UniqueArtists)

	a.QueryRow(`SELECT COUNT(DISTINCT t.album_id) FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		WHERE pe.played_at >= ? AND pe.played_at < ?`, start, end).Scan(&s.UniqueAlbums)

	a.QueryRow(`SELECT COUNT(DISTINCT pe.track_id) FROM play_events pe
		WHERE pe.played_at >= ? AND pe.played_at < ?`, start, end).Scan(&s.UniqueTracks)

	a.QueryRow(`SELECT ar.name FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		JOIN track_artists ta ON ta.track_id = t.id
		JOIN artists ar ON ar.id = ta.artist_id
		WHERE pe.played_at >= ? AND pe.played_at < ?
		GROUP BY ar.id ORDER BY COUNT(*) DESC LIMIT 1`, start, end).Scan(&s.TopArtist)

	a.QueryRow(`SELECT t.name FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		WHERE pe.played_at >= ? AND pe.played_at < ?
		GROUP BY t.id ORDER BY COUNT(*) DESC LIMIT 1`, start, end).Scan(&s.TopTrack)

	return s, nil
}

func (a *DB) TopArtists(limit int) ([]ArtistRanking, error) {
	rows, err := a.Query(`
		SELECT ar.name, ar.spotify_id, COUNT(*) as cnt
		FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		JOIN track_artists ta ON ta.track_id = t.id
		JOIN artists ar ON ar.id = ta.artist_id
		GROUP BY ar.id
		ORDER BY cnt DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rankings []ArtistRanking
	for rows.Next() {
		var r ArtistRanking
		if err := rows.Scan(&r.ArtistName, &r.SpotifyID, &r.PlayCount); err != nil {
			return nil, err
		}
		rankings = append(rankings, r)
	}
	return rankings, rows.Err()
}

func (a *DB) HourlyDistribution() ([]HourlyHeatmap, error) {
	rows, err := a.Query(`
		SELECT CAST(strftime('%H', pe.played_at) AS INTEGER) as hour, COUNT(*) as cnt
		FROM play_events pe
		GROUP BY hour ORDER BY hour`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hours []HourlyHeatmap
	for rows.Next() {
		var h HourlyHeatmap
		if err := rows.Scan(&h.Hour, &h.Count); err != nil {
			return nil, err
		}
		hours = append(hours, h)
	}
	return hours, rows.Err()
}

func (a *DB) TotalStats() (totalPlays int, totalMinutes int, err error) {
	err = a.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(t.duration_ms), 0) / 60000
		FROM play_events pe JOIN tracks t ON t.id = pe.track_id`).Scan(&totalPlays, &totalMinutes)
	return
}

func (a *DB) ListeningStreak() (int, error) {
	rows, err := a.Query(`
		SELECT DISTINCT DATE(played_at) as d
		FROM play_events ORDER BY d DESC`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var days []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return 0, err
		}
		days = append(days, d)
	}

	if len(days) == 0 {
		return 0, nil
	}

	streak := 1
	for i := 1; i < len(days); i++ {
		prev, _ := time.Parse("2006-01-02", days[i-1])
		if prev.AddDate(0, 0, -1).Format("2006-01-02") == days[i] {
			streak++
		} else {
			break
		}
	}
	return streak, nil
}
