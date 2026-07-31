# GitHub Actions Configuration

## Repository Variables (Settings > Variables > Actions)

| Variable | Default | Description |
|---|---|---|
| `COLLECTOR_CRON_1` | `0 4 * * *` | First daily collection (UTC). Default: 12:00 GMT+8 |
| `COLLECTOR_CRON_2` | `0 12 * * *` | Second daily collection (UTC). Default: 20:00 GMT+8 |

## Repository Secrets (Settings > Secrets > Actions)

| Secret | Required | Description |
|---|---|---|
| `SPOTIFY_CLIENT_ID` | Yes | Spotify App Client ID |
| `SPOTIFY_CLIENT_SECRET` | Yes | Spotify App Client Secret |
| `SPOTIFY_REFRESH_TOKEN` | Yes | OAuth refresh token from `go run ./cmd/auth-server/` |

## Timezone Reference

| Local Time (GMT+8) | Cron (UTC) |
|---|---|
| 12:00 | `0 4 * * *` |
| 20:00 | `0 12 * * *` |
| 08:00 | `0 0 * * *` |
| 00:00 | `0 16 * * *` (previous day) |

## Override Example

To change collection times, go to **Settings > Variables > Actions** and add:
- `COLLECTOR_CRON_1` = `0 2 * * *` (10:00 GMT+8)
- `COLLECTOR_CRON_2` = `0 10 * * *` (18:00 GMT+8)

The workflow uses these variables with fallback defaults.