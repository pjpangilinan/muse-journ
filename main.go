package main

import (
	"errors"
	"fmt"
)

type Track struct {
	Name       string
	Artist     string
	DurationMs int
	Popularity int
}

func (t *Track) FormatTrack() (string, error) {
	if t.DurationMs < 0 {
		return "", errors.New("Duration cannot be negative.")
	}

	seconds := t.DurationMs / 1000
	minutes := seconds / 60
	rem := seconds % 60

	return fmt.Sprintf("%d:%02d", minutes, rem), nil
}

func (t *Track) String() string {
	s, err := t.FormatTrack()
	if err != nil {
		return fmt.Sprintf("Error encountered: %s", err)
	}
	return fmt.Sprintf("%-10s | %-13s | %s", t.Name, t.Artist, s)
}

func main() {
	tracks := []*Track{
		{Name: "Mananatili", Artist: "Cup of Jose", DurationMs: 257000},
		{Name: "Prinsesa", Artist: "OPM", DurationMs: 300001},
		{Name: "Notos", Artist: "The Oh Hellos", DurationMs: 351006},
	}

	for _, track := range tracks {
		fmt.Println(track.String())
	}
}
