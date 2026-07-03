package spotify

import "errors"

var (
	ErrUnauthorized   = errors.New("spotify: unauthorized - check credentials")
	ErrRateLimited    = errors.New("spotify: rate limited")
	ErrNotFound       = errors.New("spotify: resource not found")
	ErrEmptyToken     = errors.New("spotify: empty access token")
	ErrEmptyName      = errors.New("name is empty")
)
