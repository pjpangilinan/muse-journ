package reports

import (
	"fmt"
	"strings"
	"time"

	"github.com/pjpangilinan/muse-journ/internal/analytics"
)

type Generator struct {
	analytics *analytics.DB
}

func New(db *analytics.DB) *Generator {
	return &Generator{analytics: db}
}

func (g *Generator) DailyReport(date string) (string, error) {
	summary, err := g.analytics.DailySummary(date)
	if err != nil {
		return "", fmt.Errorf("daily report: %w", err)
	}

	parsed, _ := time.Parse("2006-01-02", date)
	weekday := parsed.Weekday().String()

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Daily Report — %s (%s)\n\n", date, weekday))

	if summary.TotalPlays == 0 {
		b.WriteString("No plays recorded on this day.\n")
		return b.String(), nil
	}

	b.WriteString("## At a Glance\n\n")
	b.WriteString(fmt.Sprintf("- **Total plays:** %d\n", summary.TotalPlays))
	b.WriteString(fmt.Sprintf("- **Listening time:** %d minutes\n", summary.ListeningMin))
	b.WriteString(fmt.Sprintf("- **Unique artists:** %d\n", summary.UniqueArtists))
	b.WriteString(fmt.Sprintf("- **Unique albums:** %d\n", summary.UniqueAlbums))
	b.WriteString(fmt.Sprintf("- **Unique tracks:** %d\n", summary.UniqueTracks))

	b.WriteString("\n## Top of the Day\n\n")
	if summary.TopArtist != "" {
		b.WriteString(fmt.Sprintf("- **Top artist:** %s\n", summary.TopArtist))
	}
	if summary.TopTrack != "" {
		b.WriteString(fmt.Sprintf("- **Top track:** %s\n", summary.TopTrack))
	}
	if summary.TopAlbum != "" {
		b.WriteString(fmt.Sprintf("- **Top album:** %s\n", summary.TopAlbum))
	}

	return b.String(), nil
}

func (g *Generator) MonthlyReport(yearMonth string) (string, error) {
	summary, err := g.analytics.MonthlySummary(yearMonth)
	if err != nil {
		return "", fmt.Errorf("monthly report: %w", err)
	}

	parsed, _ := time.Parse("2006-01", yearMonth)
	monthName := parsed.Format("January 2006")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Monthly Report — %s\n\n", monthName))

	if summary.TotalPlays == 0 {
		b.WriteString("No plays recorded this month.\n")
		return b.String(), nil
	}

	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- **Total plays:** %d\n", summary.TotalPlays))
	b.WriteString(fmt.Sprintf("- **Listening time:** %d minutes (%.1f hours)\n",
		summary.ListeningMin, float64(summary.ListeningMin)/60))
	b.WriteString(fmt.Sprintf("- **Unique artists:** %d\n", summary.UniqueArtists))
	b.WriteString(fmt.Sprintf("- **Unique albums:** %d\n", summary.UniqueAlbums))
	b.WriteString(fmt.Sprintf("- **Unique tracks:** %d\n", summary.UniqueTracks))

	b.WriteString("\n## Top of the Month\n\n")
	if summary.TopArtist != "" {
		b.WriteString(fmt.Sprintf("- **Top artist:** %s\n", summary.TopArtist))
	}
	if summary.TopTrack != "" {
		b.WriteString(fmt.Sprintf("- **Top track:** %s\n", summary.TopTrack))
	}

	topArtists, err := g.analytics.TopArtists(10)
	if err == nil && len(topArtists) > 0 {
		b.WriteString("\n## Top Artists\n\n")
		b.WriteString("| Rank | Artist | Plays |\n")
		b.WriteString("|------|--------|-------|\n")
		for i, a := range topArtists {
			b.WriteString(fmt.Sprintf("| %d | %s | %d |\n", i+1, a.ArtistName, a.PlayCount))
		}
	}

	return b.String(), nil
}
