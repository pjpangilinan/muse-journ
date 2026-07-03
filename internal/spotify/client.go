package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.spotify.com/v1"

type Client struct {
	http        *http.Client
	accessToken string
}

func NewClient(accessToken string) *Client {
	return &Client{
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &http.Transport{IdleConnTimeout: 15 * time.Second},
		},
		accessToken: accessToken,
	}
}

func (c *Client) SetToken(token string) {
	c.accessToken = token
}

func (c *Client) GetRecentlyPlayed(after string) (*RecentlyPlayedResponse, error) {
	u := fmt.Sprintf("%s/me/player/recently-played?limit=50", baseURL)
	if after != "" {
		u += "&after=" + after
	}

	resp, err := c.get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retry := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("%w: retry after %s sec", ErrRateLimited, retry)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result RecentlyPlayedResponse
	if err := decodeJSON(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("decode recently played: %w", err)
	}
	if len(result.Items) == 0 {
		return &result, nil
	}
	return &result, nil
}

func (c *Client) GetTrack(id string) (*TrackDetails, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrNotFound
	}
	u := fmt.Sprintf("%s/tracks/%s", baseURL, id)
	resp, err := c.get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retry := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("%w: retry after %s sec", ErrRateLimited, retry)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var track TrackDetails
	if err := decodeJSON(resp.Body, &track); err != nil {
		return nil, fmt.Errorf("decode track: %w", err)
	}
	if track.ID == "" {
		return nil, fmt.Errorf("track response missing ID for %s", id)
	}
	return &track, nil
}

func (c *Client) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.accessToken == "" {
		return nil, ErrEmptyToken
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	return resp, nil
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp ErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		return fmt.Errorf("spotify API error (HTTP %d): %s", resp.StatusCode, errResp.Error.Message)
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = "no response body"
	}
	return fmt.Errorf("spotify API error (HTTP %d): %s", resp.StatusCode, msg)
}

func decodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
