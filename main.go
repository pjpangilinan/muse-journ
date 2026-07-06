package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pjpangilinan/muse-journ/internal/analytics"
	"github.com/pjpangilinan/muse-journ/internal/config"
	"github.com/pjpangilinan/muse-journ/internal/database"
	"github.com/pjpangilinan/muse-journ/internal/reports"
	"github.com/pjpangilinan/muse-journ/internal/spotify"
)

//go:embed cmd/dashboard/templates/*
var templateFS embed.FS

func main() {
	cfg := config.Load()

	if len(os.Args) < 2 {
		fmt.Println("Usage: muse-journ <command>")
		fmt.Println("Commands:")
		fmt.Println("  collector   Fetch recent plays from Spotify")
		fmt.Println("  dashboard   Start the web dashboard server")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "collector":
		runCollector(cfg)
	case "dashboard":
		runDashboard(cfg)
	case "build-site":
		runBuildSite(cfg)
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func runCollector(cfg *config.Config) {
	if cfg.SpotifyClientID == "" || cfg.SpotifyClientSecret == "" || cfg.SpotifyRefreshToken == "" {
		log.Fatal("SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET, and SPOTIFY_REFRESH_TOKEN required")
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	collector := spotify.NewCollector(cfg.SpotifyClientID, cfg.SpotifyClientSecret, cfg.SpotifyRefreshToken)

	var after string
	var lastPlayedAt string
	row := db.QueryRow("SELECT played_at FROM play_events ORDER BY played_at DESC LIMIT 1")
	if err := row.Scan(&lastPlayedAt); err == nil {
		t, err := time.Parse(time.RFC3339, lastPlayedAt)
		if err == nil {
			after = fmt.Sprintf("%d", t.UnixMilli())
		}
	}

	plays, err := collector.Collect(after)
	if err != nil {
		log.Fatalf("collect: %v", err)
	}

	inserted := 0
	for _, play := range plays {
		if len(play.ArtistIDs) == 0 || len(play.ArtistNames) == 0 {
			log.Printf("skip play with no artists: %s at %s", play.TrackName, play.PlayedAt)
			continue
		}
		artistID, err := db.UpsertArtist(&database.Artist{
			SpotifyID:  play.ArtistIDs[0],
			Name:       play.ArtistNames[0],
			Genres:     joinGenres(play.ArtistGenres),
			Popularity: play.ArtistPopularities[0],
		})
		if err != nil {
			log.Printf("upsert artist: %v", err)
			continue
		}

		albumID, err := db.UpsertAlbum(&database.Album{
			SpotifyID:   play.AlbumSpotifyID,
			Name:        play.AlbumName,
			ReleaseDate: play.ReleaseDate,
			TotalTracks: play.TotalTracks,
			CoverURL:    play.CoverURL,
		})
		if err != nil {
			log.Printf("upsert album: %v", err)
			continue
		}

		trackID, err := db.UpsertTrack(&database.Track{
			SpotifyID:   play.TrackSpotifyID,
			Name:        play.TrackName,
			DurationMS:  play.DurationMS,
			Explicit:    play.Explicit,
			DiscNumber:  play.DiscNumber,
			TrackNumber: play.TrackNumber,
			Popularity:  play.Popularity,
			PreviewURL:  play.PreviewURL,
			AlbumID:     albumID,
		})
		if err != nil {
			log.Printf("upsert track: %v", err)
			continue
		}

		if err := db.InsertTrackArtist(trackID, artistID); err != nil {
			log.Printf("link track-artist: %v", err)
		}
		if err := db.InsertAlbumArtist(albumID, artistID); err != nil {
			log.Printf("link album-artist: %v", err)
		}

		_, err = db.InsertPlayEvent(&database.PlayEvent{
			TrackID:  trackID,
			PlayedAt: play.PlayedAt,
			Source:   "collector",
		})
		if err != nil {
			log.Printf("insert play event: %v", err)
			continue
		}
		inserted++
	}

	log.Printf("collector: %d new plays inserted", inserted)

	analyticsDB := analytics.New(db.DB)
	reportGen := reports.New(analyticsDB)

	today := time.Now().UTC().Format("2006-01-02")
	if report, err := reportGen.DailyReport(today); err == nil {
		fmt.Println(report)
	}

	currentMonth := time.Now().UTC().Format("2006-01")
	if report, err := reportGen.MonthlyReport(currentMonth); err == nil {
		fmt.Println(report)
	}
}

func runDashboard(cfg *config.Config) {
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
			return t.Local().Format("Jan 2 15:04")
		},
	}).ParseFS(templateFS, "cmd/dashboard/templates/*.html"))

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

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		rangeVal := r.URL.Query().Get("range")
		from, to := parseDateRange(rangeVal, r.URL.Query().Get("from"), r.URL.Query().Get("to"))

		plays, err := db.GetRecentPlaysRange(8, 0, from, to)
		if err != nil {
			log.Printf("dashboard error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		totalPlays, totalMinutes, _ := analyticsDB.TotalStats()
		streak, _ := analyticsDB.ListeningStreak()

		var totalPlaysFiltered int
		var totalMinFiltered int
		allPlays, _ := db.GetRecentPlaysRange(500, 0, from, to)
		for _, p := range allPlays {
			totalPlaysFiltered++
			totalMinFiltered += p.Track.DurationMS
		}
		totalMinFiltered /= 60000

		today := time.Now().UTC().Format("2006-01-02")
		daily, _ := analyticsDB.DailySummary(today)
		topArtists, _ := analyticsDB.TopArtists(5)
		var lastCollected string
		db.QueryRow("SELECT played_at FROM play_events ORDER BY played_at DESC LIMIT 1").Scan(&lastCollected)

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

		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("render template: %v", err)
		}
	})

	bind := os.Getenv("BIND_ADDR")
	if bind == "" {
		bind = fmt.Sprintf(":%d", cfg.Port)
	}
	log.Printf("dashboard listening on http://%s", bind)
	if err := http.ListenAndServe(bind, mux); err != nil {
		log.Fatal(err)
	}
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
		start := now.AddDate(-1, 0, 0)
		return start.Format("2006-01-02"), now.Format("2006-01-02")
	default:
		return "", ""
	}
}

func runBuildSite(cfg *config.Config) {
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
			return t.Local().Format("Jan 2 15:04")
		},
	}).ParseFS(templateFS, "cmd/dashboard/templates/*.html"))

	plays, err := db.GetRecentPlaysRange(200, 0, "", "")
	if err != nil {
		log.Fatalf("query plays: %v", err)
	}

	totalPlays, totalMinutes, _ := analyticsDB.TotalStats()
	streak, _ := analyticsDB.ListeningStreak()
	today := time.Now().UTC().Format("2006-01-02")
	daily, _ := analyticsDB.DailySummary(today)
	topArtists, _ := analyticsDB.TopArtists(5)

	var totalMinFiltered int
	for _, p := range plays {
		totalMinFiltered += p.Track.DurationMS
	}
	totalMinFiltered /= 60000

	var lastCollected string
	db.QueryRow("SELECT played_at FROM play_events ORDER BY played_at DESC LIMIT 1").Scan(&lastCollected)

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
		log.Fatalf("marshal static data: %v", err)
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
		log.Fatalf("mkdir _site: %v", err)
	}

	f, err := os.Create("_site/index.html")
	if err != nil {
		log.Fatalf("create _site/index.html: %v", err)
	}
	defer f.Close()

	if err := tmpl.ExecuteTemplate(f, "index.html", data); err != nil {
		log.Fatalf("render: %v", err)
	}

	log.Printf("static site generated: _site/index.html (%d plays)", len(plays))
}

func writeJSON(w io.Writer, v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
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

	result := "["
	for i, g := range unique {
		if i > 0 {
			result += ","
		}
		result += `"` + g + `"`
	}
	result += "]"
	return result
}
