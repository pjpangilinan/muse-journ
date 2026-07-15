package app

import (
	"net/http/httptest"
	"testing"
)

func TestAtoi(t *testing.T) {
	tests := []struct {
		s   string
		def int
		max int
		exp int
	}{
		{"42", 0, 100, 42},
		{"", 10, 100, 10},
		{"999", 0, 50, 0},
		{"-1", 0, 100, 0},
		{"abc", 5, 100, 5},
		{"0", 0, 100, 0},
		{"100", 0, 100, 100},
	}
	for _, tc := range tests {
		got := atoi(tc.s, tc.def, tc.max)
		if got != tc.exp {
			t.Errorf("atoi(%q, %d, %d) = %d, want %d", tc.s, tc.def, tc.max, got, tc.exp)
		}
	}
}

func TestJoinGenres(t *testing.T) {
	if got := joinGenres(nil); got != "" {
		t.Errorf("joinGenres(nil) = %q, want empty", got)
	}
	if got := joinGenres([]string{}); got != "" {
		t.Errorf("joinGenres([]) = %q, want empty", got)
	}
	got := joinGenres([]string{"rock", "pop"})
	if got != `["rock","pop"]` {
		t.Errorf("joinGenres([rock pop]) = %q, want JSON array", got)
	}
	got = joinGenres([]string{"rock", "rock"})
	if got != `["rock"]` {
		t.Errorf("joinGenres([rock rock]) = %q, want deduped", got)
	}
}

func TestParseDateRange(t *testing.T) {
	from, to := parseDateRange("today", "", "")
	if from != to || from == "" {
		t.Errorf("today: from=%q to=%q, expected same non-empty", from, to)
	}
	from, to = parseDateRange("", "", "")
	if from != "" || to != "" {
		t.Errorf("empty range: from=%q to=%q, expected empty", from, to)
	}
	from, to = parseDateRange("year", "", "")
	if from == "" || to == "" {
		t.Errorf("year: got empty from/to")
	}
	if len(from) < 10 || from[5:] != "01-01" {
		t.Errorf("year: from=%q, expected Jan 1 of current year", from)
	}
	from, to = parseDateRange("week", "", "")
	if from == "" || to == "" {
		t.Errorf("week: got empty from/to")
	}
	from, to = parseDateRange("month", "", "")
	if from == "" || to == "" {
		t.Errorf("month: got empty from/to")
	}
	from, to = parseDateRange("", "2024-03-15", "2024-03-20")
	if from != "2024-03-15" || to != "2024-03-20" {
		t.Errorf("explicit range: from=%q to=%q, expected 2024-03-15/2024-03-20", from, to)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, map[string]string{"hello": "world"})
	if w.Code != 200 {
		t.Errorf("writeJSON status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if w.Body.Len() == 0 {
		t.Error("writeJSON body is empty")
	}
}
