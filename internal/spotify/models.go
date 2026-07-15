package spotify

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type ErrorResponse struct {
	Error struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

type RecentlyPlayedResponse struct {
	Items   []PlayHistoryItem `json:"items"`
	Next    string            `json:"next"`
	Cursors struct {
		After  string `json:"after"`
		Before string `json:"before"`
	} `json:"cursors"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type PlayHistoryItem struct {
	Track    Track    `json:"track"`
	PlayedAt string   `json:"played_at"`
	Context  *Context `json:"context"`
}

type Track struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DurationMS  int      `json:"duration_ms"`
	Explicit    bool     `json:"explicit"`
	DiscNumber  int      `json:"disc_number"`
	TrackNumber int      `json:"track_number"`
	Popularity  int      `json:"popularity"`
	PreviewURL  string   `json:"preview_url"`
	Album       Album    `json:"album"`
	Artists     []Artist `json:"artists"`
}

type Album struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ReleaseDate string   `json:"release_date"`
	TotalTracks int      `json:"total_tracks"`
	Images      []Image  `json:"images"`
	Artists     []Artist `json:"artists"`
}

type Artist struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Genres    []string `json:"genres"`
	Followers struct {
		Total int `json:"total"`
	} `json:"followers"`
	Popularity int `json:"popularity"`
}

type Image struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

type Context struct {
	Type string `json:"type"`
	HRef string `json:"href"`
	URI  string `json:"uri"`
}

type NormalizedPlay struct {
	TrackName          string
	TrackSpotifyID     string
	DurationMS         int
	Explicit           bool
	DiscNumber         int
	TrackNumber        int
	Popularity         int
	PreviewURL         string
	AlbumSpotifyID     string
	AlbumName          string
	ReleaseDate        string
	TotalTracks        int
	CoverURL           string
	ArtistIDs          []string
	ArtistNames        []string
	ArtistGenres       []string
	ArtistFollowers    []int
	ArtistPopularities []int
	PlayedAt           string
	Context            string
	ContextType        string
}
