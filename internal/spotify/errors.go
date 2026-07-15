package spotify

import "errors"

var (
	ErrUnauthorized = errors.New("spotify: unauthorized - check credentials")
	ErrRateLimited  = errors.New("spotify: rate limited")
	ErrEmptyToken   = errors.New("spotify: empty access token")
)
