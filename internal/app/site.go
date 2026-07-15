package app

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"time"

	"github.com/pjpangilinan/muse-journ/internal/analytics"
	"github.com/pjpangilinan/muse-journ/internal/config"
	"github.com/pjpangilinan/muse-journ/internal/database"
)

func BuildSite(cfg *config.Config, tmpl *template.Template) error {
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return err
	}

	analyticsDB := analytics.New(db.DB)

	plays, err := db.GetRecentPlaysRange(200, 0, "", "")
	if err != nil {
		return err
	}

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
		log.Printf("daily summary: %v", err)
	}
	topArtists, err := analyticsDB.TopArtists(5)
	if err != nil {
		topArtists = []analytics.ArtistRanking{}
		log.Printf("top artists: %v", err)
	}

	var totalMinFiltered int
	for _, p := range plays {
		totalMinFiltered += p.Track.DurationMS
	}
	totalMinFiltered /= 60000

	var lastCollected string
	if err := db.QueryRow("SELECT played_at FROM play_events ORDER BY played_at DESC LIMIT 1").Scan(&lastCollected); err != nil {
		log.Printf("no plays collected yet")
	}

	staticJSON, err := json.Marshal(map[string]any{
		"plays":       plays,
		"totalPlays":  totalPlays,
		"totalMin":    totalMinutes,
		"streak":      streak,
		"daily":       daily,
		"filterPlays": len(plays),
		"filterMin":   totalMinFiltered,
		"topArtists":  topArtists,
	})
	if err != nil {
		return err
	}

	data := map[string]any{
		"StaticData":    template.JS(string(staticJSON)),
		"Daily":         daily,
		"TotalPlays":    totalPlays,
		"TotalMin":      totalMinutes,
		"Streak":        streak,
		"FilterPlays":   len(plays),
		"FilterMin":     totalMinFiltered,
		"TopArtists":    topArtists,
		"LastCollected": lastCollected,
	}

	if err := os.MkdirAll("_site", 0755); err != nil {
		return err
	}

	if err := os.WriteFile("_site/.nojekyll", nil, 0644); err != nil {
		return err
	}

	f, err := os.Create("_site/index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	if err := tmpl.ExecuteTemplate(f, "index.html", data); err != nil {
		return err
	}

	log.Printf("static site generated: _site/index.html (%d plays)", len(plays))
	return nil
}
