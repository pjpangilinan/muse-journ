package speaker

import (
	"errors"
	"fmt"
)

// ErrEmptyName is returned when a Track has no name.
var ErrEmptyName = errors.New("name is empty")

type Speaker interface {
	Speak() (string, error)
}

type Track struct {
	Name   string
	Artist string
}

func (t *Track) Speak() (string, error) {
	if t.Name == "" {
		return "", fmt.Errorf("track by %q: %w", t.Artist, ErrEmptyName)
	}
	return fmt.Sprintf("♪ Now playing: %s by %s", t.Name, t.Artist), nil
}

type Alert struct {
	Level   string
	Message string
}

func (a *Alert) Speak() (string, error) {
	if a.Message == "" {
		return "", fmt.Errorf("%s alert: %w", a.Level, ErrEmptyName)
	}
	return fmt.Sprintf("[%s] %s", a.Level, a.Message), nil
}
