# GitHub Actions Configuration

## Repository Secrets (Settings > Secrets > Actions)

| Secret | Required | Description |
|---|---|---|
| `SPOTIFY_CLIENT_ID` | Yes | Spotify App Client ID |
| `SPOTIFY_CLIENT_SECRET` | Yes | Spotify App Client Secret |
| `SPOTIFY_REFRESH_TOKEN` | Yes | OAuth refresh token from `go run ./cmd/auth-server/` |

## Collector Schedule

Default: 12:00 and 20:00 GMT+8 (04:00 and 12:00 UTC).

To change collection times, edit `.github/workflows/collector.yml` directly:

```yaml
on:
  schedule:
    - cron: '0 4 * * *'   # 12:00 GMT+8
    - cron: '0 12 * * *'  # 20:00 GMT+8
```

**Note:** GitHub Actions requires static cron strings — repository variables cannot be used in schedule triggers.

## Timezone Reference

| Local Time (GMT+8) | Cron (UTC) |
|---|---|
| 12:00 | `0 4 * * *` |
| 20:00 | `0 12 * * *` |
| 08:00 | `0 0 * * *` |
| 00:00 | `0 16 * * *` (previous day) |