package database

type Artist struct {
	ID         int64  `json:"id"`
	SpotifyID  string `json:"spotify_id"`
	Name       string `json:"name"`
	Genres     string `json:"genres"`
	Followers  int    `json:"followers"`
	Popularity int    `json:"popularity"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type Album struct {
	ID          int64  `json:"id"`
	SpotifyID   string `json:"spotify_id"`
	Name        string `json:"name"`
	ReleaseDate string `json:"release_date"`
	TotalTracks int    `json:"total_tracks"`
	CoverURL    string `json:"cover_url"`
	CreatedAt   string `json:"created_at"`
}

type Track struct {
	ID          int64  `json:"id"`
	SpotifyID   string `json:"spotify_id"`
	Name        string `json:"name"`
	DurationMS  int    `json:"duration_ms"`
	Explicit    bool   `json:"explicit"`
	DiscNumber  int    `json:"disc_number"`
	TrackNumber int    `json:"track_number"`
	Popularity  int    `json:"popularity"`
	PreviewURL  string `json:"preview_url"`
	AlbumID     int64  `json:"album_id"`
	CreatedAt   string `json:"created_at"`
}

type PlayEvent struct {
	ID       int64  `json:"id"`
	TrackID  int64  `json:"track_id"`
	PlayedAt string `json:"played_at"`
	Device   string `json:"device"`
	Shuffle  bool   `json:"shuffle"`
	Repeat   string `json:"repeat"`
	Context  string `json:"context"`
	Source   string `json:"source"`
}

type TrackArtist struct {
	TrackID  int64 `json:"track_id"`
	ArtistID int64 `json:"artist_id"`
}

type AlbumArtist struct {
	AlbumID  int64 `json:"album_id"`
	ArtistID int64 `json:"artist_id"`
}
