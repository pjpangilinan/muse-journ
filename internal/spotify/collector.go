package spotify

import (
	"fmt"
	"sync"
	"time"
)

type Collector struct {
	clientID       string
	clientSecret   string
	refreshToken   string
	client         *Client
	mu             sync.Mutex
	cachedToken    string
	tokenExpiresAt time.Time
}

func NewCollector(clientID, clientSecret, refreshToken string) *Collector {
	return &Collector{
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		client:       NewClient(""),
	}
}

func (c *Collector) ensureToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Now().Before(c.tokenExpiresAt) {
		c.client.SetToken(c.cachedToken)
		return nil
	}

	token, err := RefreshAccessToken(c.clientID, c.clientSecret, c.refreshToken)
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	c.cachedToken = token.AccessToken
	c.tokenExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	c.client.SetToken(token.AccessToken)
	return nil
}

func (c *Collector) Collect(after string) ([]NormalizedPlay, error) {
	if err := c.ensureToken(); err != nil {
		return nil, err
	}

	resp, err := c.client.GetRecentlyPlayed(after)
	if err != nil {
		return nil, fmt.Errorf("get recently played: %w", err)
	}

	var plays []NormalizedPlay
	for _, item := range resp.Items {
		play := c.normalize(item)
		plays = append(plays, play)
	}

	return plays, nil
}

func (c *Collector) normalize(item PlayHistoryItem) NormalizedPlay {
	track := item.Track
	album := track.Album

	var artistNames []string
	var artistIDs []string
	for _, a := range track.Artists {
		artistNames = append(artistNames, a.Name)
		artistIDs = append(artistIDs, a.ID)
	}

	var genres []string
	var followers []int
	var popularities []int
	for _, a := range track.Artists {
		genres = append(genres, a.Genres...)
		followers = append(followers, a.Followers.Total)
		popularities = append(popularities, a.Popularity)
	}

	coverURL := ""
	if len(album.Images) > 0 {
		coverURL = album.Images[0].URL
	}

	var contextURI, contextType string
	if item.Context != nil {
		contextURI = item.Context.URI
		contextType = item.Context.Type
	}

	return NormalizedPlay{
		TrackName:          track.Name,
		TrackSpotifyID:     track.ID,
		DurationMS:         track.DurationMS,
		Explicit:           track.Explicit,
		DiscNumber:         track.DiscNumber,
		TrackNumber:        track.TrackNumber,
		Popularity:         track.Popularity,
		PreviewURL:         track.PreviewURL,
		AlbumSpotifyID:     album.ID,
		AlbumName:          album.Name,
		ReleaseDate:        album.ReleaseDate,
		TotalTracks:        album.TotalTracks,
		CoverURL:           coverURL,
		ArtistIDs:          artistIDs,
		ArtistNames:        artistNames,
		ArtistGenres:       genres,
		ArtistFollowers:    followers,
		ArtistPopularities: popularities,
		PlayedAt:           item.PlayedAt,
		Context:            contextURI,
		ContextType:        contextType,
	}
}
