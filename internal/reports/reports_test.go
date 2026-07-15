package reports

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"github.com/pjpangilinan/muse-journ/internal/analytics"
	"github.com/pjpangilinan/muse-journ/internal/database"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seedData(t, db.DB)
	return db.DB
}

func seedData(t *testing.T, db *sql.DB) {
	t.Helper()
	db.Exec(`INSERT OR IGNORE INTO artists (id, spotify_id, name) VALUES (1, 'a1', 'Artist 1'), (2, 'a2', 'Artist 2')`)
	db.Exec(`INSERT OR IGNORE INTO albums (id, spotify_id, name, total_tracks) VALUES (1, 'al1', 'Album 1', 10)`)
	db.Exec(`INSERT OR IGNORE INTO tracks (id, spotify_id, name, duration_ms, album_id) VALUES (1, 't1', 'Track 1', 240000, 1), (2, 't2', 'Track 2', 180000, 1)`)
	db.Exec(`INSERT OR IGNORE INTO play_events (track_id, played_at, source) VALUES (1, '2024-03-15T10:00:00Z', 'test'), (1, '2024-03-15T11:00:00Z', 'test'), (2, '2024-03-15T12:00:00Z', 'test')`)
	db.Exec(`INSERT OR IGNORE INTO track_artists (track_id, artist_id) VALUES (1, 1), (2, 2)`)
}

func TestDailyReport(t *testing.T) {
	a := analytics.New(setupTestDB(t))
	g := New(a)

	report, err := g.DailyReport("2024-03-15")
	if err != nil {
		t.Fatalf("daily report: %v", err)
	}
	if !strings.Contains(report, "Total plays") {
		t.Fatal("expected Total plays in report")
	}
	if !strings.Contains(report, "Top track") {
		t.Fatal("expected Top track in report")
	}
	if !strings.Contains(report, "Artist 1") {
		t.Fatalf("expected Artist 1 in report, got:\n%s", report)
	}
}

func TestDailyReportNoData(t *testing.T) {
	a := analytics.New(setupTestDB(t))
	g := New(a)

	report, err := g.DailyReport("2099-01-01")
	if err != nil {
		t.Fatalf("daily report no data: %v", err)
	}
	if !strings.Contains(report, "No plays recorded") {
		t.Fatalf("expected no-plays message, got:\n%s", report)
	}
}

func TestDailyReportError(t *testing.T) {
	db := setupTestDB(t)
	db.Close()
	a := analytics.New(db)
	g := New(a)

	_, err := g.DailyReport("2024-03-15")
	if err == nil {
		t.Fatal("expected error from closed db")
	}
}

func TestMonthlyReport(t *testing.T) {
	a := analytics.New(setupTestDB(t))
	g := New(a)

	report, err := g.MonthlyReport("2024-03")
	if err != nil {
		t.Fatalf("monthly report: %v", err)
	}
	if !strings.Contains(report, "Total plays") {
		t.Fatal("expected Total plays in report")
	}
	if !strings.Contains(report, "Artist 1") {
		t.Fatalf("expected Artist 1 in report, got:\n%s", report)
	}
}

func TestMonthlyReportNoData(t *testing.T) {
	a := analytics.New(setupTestDB(t))
	g := New(a)

	report, err := g.MonthlyReport("2099-01")
	if err != nil {
		t.Fatalf("monthly report no data: %v", err)
	}
	if !strings.Contains(report, "No plays recorded") {
		t.Fatalf("expected no-plays message, got:\n%s", report)
	}
}

func TestMonthlyReportError(t *testing.T) {
	db := setupTestDB(t)
	db.Close()
	a := analytics.New(db)
	g := New(a)

	_, err := g.MonthlyReport("2024-03")
	if err == nil {
		t.Fatal("expected error from closed db")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
