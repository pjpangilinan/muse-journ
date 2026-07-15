package database

import (
	"database/sql"
	"fmt"
	"strconv"
)

func (db *DB) UpsertArtist(a *Artist) (int64, error) {
	return upsert(db.DB, `
		INSERT INTO artists (spotify_id, name, genres, followers, popularity)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(spotify_id) DO UPDATE SET
			name=excluded.name, genres=excluded.genres,
			followers=excluded.followers, popularity=excluded.popularity,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id`, a.SpotifyID, a.Name, a.Genres, a.Followers, a.Popularity)
}

func (db *DB) UpsertAlbum(a *Album) (int64, error) {
	return upsert(db.DB, `
		INSERT INTO albums (spotify_id, name, release_date, total_tracks, cover_url)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(spotify_id) DO UPDATE SET
			name=excluded.name, release_date=excluded.release_date,
			total_tracks=excluded.total_tracks, cover_url=excluded.cover_url
		RETURNING id`, a.SpotifyID, a.Name, a.ReleaseDate, a.TotalTracks, a.CoverURL)
}

func (db *DB) UpsertTrack(t *Track) (int64, error) {
	return upsert(db.DB, `
		INSERT INTO tracks (spotify_id, name, duration_ms, explicit, disc_number, track_number, popularity, preview_url, album_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(spotify_id) DO UPDATE SET
			name=excluded.name, duration_ms=excluded.duration_ms,
			explicit=excluded.explicit, popularity=excluded.popularity,
			preview_url=excluded.preview_url, album_id=excluded.album_id
		RETURNING id`,
		t.SpotifyID, t.Name, t.DurationMS, t.Explicit, t.DiscNumber,
		t.TrackNumber, t.Popularity, t.PreviewURL, t.AlbumID)
}

func (db *DB) InsertPlayEvent(pe *PlayEvent) (int64, error) {
	var id int64
	err := db.QueryRow(`
		INSERT INTO play_events (track_id, played_at, device, shuffle, repeat, context, source)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(track_id, played_at) DO NOTHING
		RETURNING id`,
		pe.TrackID, pe.PlayedAt, pe.Device, pe.Shuffle, pe.Repeat, pe.Context, pe.Source).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (db *DB) InsertTrackArtist(trackID, artistID int64) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO track_artists (track_id, artist_id) VALUES (?, ?)",
		trackID, artistID)
	return err
}

func (db *DB) InsertAlbumArtist(albumID, artistID int64) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO album_artists (album_id, artist_id) VALUES (?, ?)",
		albumID, artistID)
	return err
}

func (db *DB) GetRecentPlaysRange(limit, offset int, from, to string) ([]PlayEventWithDetails, error) {
	query := `
		SELECT pe.id, pe.played_at, pe.device, pe.context,
		       t.id, t.spotify_id, t.name, t.duration_ms, t.explicit, t.preview_url,
		       a.id, a.spotify_id, a.name, a.cover_url,
		       GROUP_CONCAT(DISTINCT ar.name) as artists
		FROM play_events pe
		JOIN tracks t ON t.id = pe.track_id
		LEFT JOIN albums a ON a.id = t.album_id
		LEFT JOIN track_artists ta ON ta.track_id = t.id
		LEFT JOIN artists ar ON ar.id = ta.artist_id`

	var args []any
	if from != "" && to != "" {
		query += " WHERE pe.played_at >= ? AND pe.played_at < ?"
		args = append(args, from+"T00:00:00Z", to+"T24:00:00Z")
	} else if from != "" {
		query += " WHERE pe.played_at >= ?"
		args = append(args, from+"T00:00:00Z")
	} else if to != "" {
		query += " WHERE pe.played_at < ?"
		args = append(args, to+"T24:00:00Z")
	}

	query += ` GROUP BY pe.id ORDER BY pe.played_at DESC LIMIT ?`
	args = append(args, limit)
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query recent plays: %w", err)
	}
	defer rows.Close()

	var results []PlayEventWithDetails
	for rows.Next() {
		var p PlayEventWithDetails
		var albumID, albumSpotifyID, albumName, albumCover sql.NullString
		var previewURL sql.NullString
		err := rows.Scan(
			&p.ID, &p.PlayedAt, &p.Device, &p.Context,
			&p.Track.ID, &p.Track.SpotifyID, &p.Track.Name,
			&p.Track.DurationMS, &p.Track.Explicit, &previewURL,
			&albumID, &albumSpotifyID, &albumName, &albumCover,
			&p.Artists)
		if err != nil {
			return nil, fmt.Errorf("scan play event: %w", err)
		}
		p.Track.PreviewURL = previewURL.String
		if albumID.Valid {
			p.Album = &Album{
				ID:        parseID(albumID.String),
				SpotifyID: albumSpotifyID.String,
				Name:      albumName.String,
				CoverURL:  albumCover.String,
			}
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

type PlayEventWithDetails struct {
	ID       int64  `json:"id"`
	PlayedAt string `json:"played_at"`
	Device   string `json:"device"`
	Context  string `json:"context"`
	Track    Track  `json:"track"`
	Album    *Album `json:"album,omitempty"`
	Artists  string `json:"artists"`
}

func upsert(db *sql.DB, query string, args ...any) (int64, error) {
	var id int64
	err := db.QueryRow(query, args...).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func parseID(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
