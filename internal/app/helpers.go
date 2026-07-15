package app

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"
)

var TemplateFuncs = template.FuncMap{
	"formatTime": func(s string) string {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return s
		}
		return t.Local().Format("Jan 2 15:04")
	},
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func joinGenres(genres []string) string {
	if len(genres) == 0 {
		return ""
	}
	seen := make(map[string]bool)
	var unique []string
	for _, g := range genres {
		if !seen[g] {
			seen[g] = true
			unique = append(unique, g)
		}
	}
	data, err := json.Marshal(unique)
	if err != nil {
		return ""
	}
	return string(data)
}

func atoi(s string, def, max int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > max {
		return def
	}
	return n
}

func parseDateRange(rangeVal, fromStr, toStr string) (from, to string) {
	if fromStr != "" && toStr != "" {
		return fromStr, toStr
	}
	now := time.Now().UTC()
	switch rangeVal {
	case "today":
		return now.Format("2006-01-02"), now.Format("2006-01-02")
	case "week":
		start := now.AddDate(0, 0, -7)
		return start.Format("2006-01-02"), now.Format("2006-01-02")
	case "month":
		start := now.AddDate(0, -1, 0)
		return start.Format("2006-01-02"), now.Format("2006-01-02")
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return start.Format("2006-01-02"), now.Format("2006-01-02")
	default:
		return "", ""
	}
}
