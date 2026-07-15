# Muse Journal

[![CI](https://github.com/pjpangilinan/muse-journ/actions/workflows/ci.yml/badge.svg)](https://github.com/pjpangilinan/muse-journ/actions/workflows/ci.yml)
[![Collector](https://github.com/pjpangilinan/muse-journ/actions/workflows/collector.yml/badge.svg)](https://github.com/pjpangilinan/muse-journ/actions/workflows/collector.yml)
[![GitHub Pages](https://github.com/pjpangilinan/muse-journ/actions/workflows/pages.yml/badge.svg)](https://github.com/pjpangilinan/muse-journ/actions/workflows/pages.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

A personal Spotify listening history archive that runs itself. Twice a day (12:00 and 20:00 UTC) it grabs your recently played tracks from Spotify, saves them to a local SQLite database, and publishes a dashboard to GitHub Pages — completely automated, no server needed.

You can browse your listening history by day, week, month, or year. Filter by calendar date. See stats like total plays, listening time, streaks, and top artists. All of it lives in a static HTML page that doesn't cost anything to host.

## How it works

On a schedule (or whenever you push to main), three GitHub Actions workflows take care of everything:

1. **Collector** runs twice a day, calls the Spotify API, saves any new plays into `music.db`, and commits the database back to the repo.
2. **CI** runs on every push — checks formatting, runs vet, builds, and runs tests.
3. **Deploy** triggers after the Collector finishes (or after any push to main). It builds the static site from the database and publishes it to GitHub Pages.

The dashboard is a single HTML file with vanilla JavaScript. It works both as a static site (hosted on Pages) and as a local web server you can run from your machine. No build step, no bundler, no backend API needed at runtime — the static site embeds all your play data directly into the page.

## What you'll need

- A **Spotify account** (free or premium — both work for recently played)
- **Go 1.26** (to run locally and for GitHub Actions)
- A **GitHub account** (for hosting the dashboard on Pages)

## Setting it up

### 1. Create a Spotify app

Head over to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) and create a new app. Any name works — call it whatever you want.

Once it's created, click "Edit Settings" and add this as a redirect URI:

```
http://127.0.0.1:9090/callback
```

Save it. You'll see a **Client ID** and **Client Secret** — keep those handy.

### 2. Get a refresh token

The refresh token is what lets the collector authenticate with Spotify automatically, without you having to log in every time. You only need to do this once.

Run this in your terminal:

```bash
SPOTIFY_CLIENT_ID=your_client_id SPOTIFY_CLIENT_SECRET=your_client_secret go run ./cmd/auth-server/
```

It'll print a URL. Open it in your browser, log in to Spotify, and authorize the app. A JSON response with your `refresh_token` will show up — save it. That's the key to the whole thing.

### 3. Try it locally

Before deploying, run the collector once to make sure everything works:

```bash
SPOTIFY_CLIENT_ID=xxx SPOTIFY_CLIENT_SECRET=yyy SPOTIFY_REFRESH_TOKEN=zzz go run . collector
```

You should see some output about how many plays were collected. If it worked, start the dashboard:

```bash
go run . dashboard
```

Open `http://127.0.0.1:8080` in your browser. You should see your listening history.

If you want to generate the static site locally (same thing that gets deployed to Pages):

```bash
go run . build-site
```

This creates an `_site/` folder with the dashboard as a static HTML file.

### 4. Push to GitHub

Create a repository on GitHub (private or public, your call), then push:

```bash
git remote add origin git@github.com:you/your-repo-name.git
git push -u origin main
```

### 5. Set up GitHub Secrets

Go to your repo's Settings → Secrets and variables → Actions, and add these three secrets:

- `SPOTIFY_CLIENT_ID` — from your Spotify app
- `SPOTIFY_CLIENT_SECRET` — from your Spotify app
- `SPOTIFY_REFRESH_TOKEN` — from step 2

### 6. Enable GitHub Pages

Go to your repo's Settings → Pages. Under "Source", select **GitHub Actions**. That's it — the next time the Collector runs or you push to main, the dashboard will deploy automatically.

## Usage

Once everything's running, here's what you can do:

| Command | What it does |
|---------|-------------|
| `go run . collector` | Fetches your latest plays from Spotify and saves them |
| `go run . dashboard` | Starts a local web server so you can browse your history |
| `go run . build-site` | Generates the static site into the `_site/` folder |
| `go run ./cmd/auth-server/` | One-time OAuth flow to get your refresh token |

The collector runs at 12:00 and 20:00 UTC by default (that's 4 AM / 12 PM Pacific, 7 AM / 3 PM Eastern, 1 PM / 9 PM Central Europe, 5:30 PM / 1:30 AM India). If those don't line up with when you actually listen to music, edit the schedule in `.github/workflows/collector.yml`.

## Project layout

```
├── .github/workflows/
│   ├── collector.yml    # Fetches plays on a schedule, commits the DB
│   ├── ci.yml           # Format check, vet, build, and test
│   └── pages.yml        # Builds the static site and deploys to Pages
├── cmd/
│   ├── collector/       # The collector binary (thin wrapper)
│   ├── dashboard/       # Dashboard server binary
│   │   └── templates/
│   │       └── index.html
│   └── auth-server/     # OAuth flow to get your refresh token
├── internal/
│   ├── analytics/       # Computes daily/monthly stats, streaks, etc.
│   ├── app/             # Shared logic across all entry points
│   ├── config/          # Reads environment variables
│   ├── database/        # SQLite layer with auto-migrations
│   ├── reports/         # Generates markdown reports
│   └── spotify/         # Spotify API client, OAuth, token management
├── main.go              # CLI entry point (routes to the right command)
├── go.mod               # Only 2 direct dependencies
└── music.db             # SQLite database (gitignored, force-added in CI)
```

## How the data fits together

```
play_events ──> tracks ──> albums
    │               │
    │               └──> track_artists ──> artists
    │
    └──> each play is unique by (track_id + played_at)
```

The database is SQLite with WAL mode. It auto-migrates every time anything runs. For reference, the database grows about 7.5KB per year (it's just text and timestamps).

## Security

- Secrets are never committed to the repo. They live in GitHub Secrets.
- The refresh token never appears in logs (except during the one-time setup).
- The SQLite database is gitignored by default. The Collector workflow force-adds it.
- The dashboard binds to all interfaces by default. If you want to lock it down locally: `BIND_ADDR=127.0.0.1:8080`

## Notes

- The Spotify API returns up to 50 recently played tracks per call. That covers about 12 hours of listening for most people.
- If the Collector workflow fails (rate limit, auth issue, etc.), it won't retry — it'll just pick up from where it left off on the next scheduled run.
- The dashboard is fully client-side. No cookies, no tracking, no analytics. It's just your data, rendered in your browser.

## License

MIT
