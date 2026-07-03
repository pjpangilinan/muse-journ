package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pjpangilinan/muse-journ/internal/analytics"
	"github.com/pjpangilinan/muse-journ/internal/config"
	"github.com/pjpangilinan/muse-journ/internal/database"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	analyticsDB := analytics.New(db.DB)

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"formatTime": func(s string) string {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return s
			}
			return t.Format("15:04:05")
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"percent": func(a, b int) float64 {
			if b == 0 {
				return 0
			}
			return float64(a) / float64(b) * 100
		},
		"now": func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
	}).ParseFS(templateFS, "templates/*.html"))

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/plays", func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		limit := 50
		if limitStr != "" {
			if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		plays, err := db.GetRecentPlays(limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, plays)
	})

	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		totalPlays, totalMinutes, err := analyticsDB.TotalStats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		streak, _ := analyticsDB.ListeningStreak()

		topArtists, _ := analyticsDB.TopArtists(10)
		hourlyDist, _ := analyticsDB.HourlyDistribution()

		today := time.Now().UTC().Format("2006-01-02")
		daily, _ := analyticsDB.DailySummary(today)
		monthly, _ := analyticsDB.MonthlySummary(time.Now().UTC().Format("2006-01"))

		writeJSON(w, map[string]any{
			"total_plays":     totalPlays,
			"total_minutes":   totalMinutes,
			"listening_streak": streak,
			"top_artists":     topArtists,
			"hourly_dist":     hourlyDist,
			"daily":           daily,
			"monthly":         monthly,
		})
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		plays, err := db.GetRecentPlays(50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		totalPlays, totalMinutes, _ := analyticsDB.TotalStats()
		streak, _ := analyticsDB.ListeningStreak()
		today := time.Now().UTC().Format("2006-01-02")
		daily, _ := analyticsDB.DailySummary(today)

		data := map[string]any{
			"Plays":      plays,
			"TotalPlays": totalPlays,
			"TotalMin":   totalMinutes,
			"Streak":     streak,
			"Daily":      daily,
		}

		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("render template: %v", err)
		}
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("dashboard listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w io.Writer, v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	w.Write(data)
}
