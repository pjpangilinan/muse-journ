package app

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/pjpangilinan/muse-journ/internal/analytics"
	"github.com/pjpangilinan/muse-journ/internal/config"
	"github.com/pjpangilinan/muse-journ/internal/database"
)

func RunDashboard(cfg *config.Config, tmpl *template.Template) error {
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return err
	}

	analyticsDB := analytics.New(db.DB)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/plays", func(w http.ResponseWriter, r *http.Request) {
		limit := atoi(r.URL.Query().Get("limit"), 100, 500)
		offset := atoi(r.URL.Query().Get("offset"), 0, 10000)
		from, to := parseDateRange(r.URL.Query().Get("range"), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
		plays, err := db.GetRecentPlaysRange(limit, offset, from, to)
		if err != nil {
			log.Printf("api error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, plays)
	})

	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		totalPlays, totalMinutes, err := analyticsDB.TotalStats()
		if err != nil {
			log.Printf("total stats: %v", err)
		}
		streak, err := analyticsDB.ListeningStreak()
		if err != nil {
			log.Printf("listening streak: %v", err)
		}
		today := time.Now().UTC().Format("2006-01-02")
		daily, err := analyticsDB.DailySummary(today)
		if err != nil {
			daily = &analytics.DailySummary{Date: today}
		}
		topArtists, err := analyticsDB.TopArtists(5)
		if err != nil {
			topArtists = []analytics.ArtistRanking{}
		}

		writeJSON(w, map[string]any{
			"total_plays":       totalPlays,
			"total_minutes":     totalMinutes,
			"listening_streak":  streak,
			"daily":             daily,
			"topArtists":        topArtists,
		})
	})

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		rangeVal := r.URL.Query().Get("range")
		from, to := parseDateRange(rangeVal, r.URL.Query().Get("from"), r.URL.Query().Get("to"))

		totalPlays, totalMinutes, err := analyticsDB.TotalStats()
		if err != nil {
			log.Printf("total stats: %v", err)
		}
		streak, err := analyticsDB.ListeningStreak()
		if err != nil {
			log.Printf("listening streak: %v", err)
		}

		plays, err := db.GetRecentPlaysRange(200, 0, from, to)
		if err != nil {
			log.Printf("get recent plays: %v", err)
		}
		var totalPlaysFiltered int
		var totalMinFiltered int
		for _, p := range plays {
			totalPlaysFiltered++
			totalMinFiltered += p.Track.DurationMS
		}
		totalMinFiltered /= 60000

		today := time.Now().UTC().Format("2006-01-02")
		daily, err := analyticsDB.DailySummary(today)
		if err != nil {
			daily = &analytics.DailySummary{Date: today}
			log.Printf("daily summary: %v", err)
		}
		topArtists, err := analyticsDB.TopArtists(5)
		if err != nil {
			topArtists = []analytics.ArtistRanking{}
			log.Printf("top artists: %v", err)
		}
		var lastCollected string
		if err := db.QueryRow("SELECT played_at FROM play_events ORDER BY played_at DESC LIMIT 1").Scan(&lastCollected); err != nil {
			log.Printf("no plays collected yet")
		}

		data := map[string]any{
			"Plays":         plays,
			"TotalPlays":    totalPlays,
			"TotalMin":      totalMinutes,
			"Streak":        streak,
			"Daily":         daily,
			"FilterPlays":   totalPlaysFiltered,
			"FilterMin":     totalMinFiltered,
			"TopArtists":    topArtists,
			"CurrentRange":  rangeVal,
			"LastCollected": lastCollected,
			"From":          from,
			"To":            to,
		}

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "index.html", data); err != nil {
			log.Printf("render template: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		buf.WriteTo(w)
	})

	log.Printf("dashboard listening on http://%s", cfg.BindAddr)
	return http.ListenAndServe(cfg.BindAddr, mux)
}
