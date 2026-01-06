# Syncthing Kicker

![Logo](docs/icon-256.png)

This is my hacky fix for the fact that Syncthing still hasn't implemented a way to _schedule_ folder re-scanning and sticks to intervals.

Since using intervals is fiddly and Syncthing completely tanks my Synology NAS CPU (because Synology refuses to implement CPU limits for Docker containers, which is a separate but very real issue), I wrote this small utility to let me schedule folder scans at specific times of the day/week.

## Features

- Trigger Syncthing `rest/db/scan` for specific folders or all folders.
- Schedule folder scans using `cron` expressions
- Optional one-shot execution, startup scans, dry-run mode, and TLS skip-verify.
- Configured entirely via environment variables.

## Configuration

Recommended default global schedule: `0 5 * * 1,3,5` (5AM Mon/Wed/Fri).

| Variable             | Default                 | Description                                                                                                                         |
| -------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `ST_API_URL`         | `http://127.0.0.1:8384` | Base URL for the Syncthing API (trailing slash optional).                                                                           |
| `ST_API_KEY`         | _required_              | Syncthing API key.                                                                                                                  |
| `ST_FOLDERS`         | `*`                     | Comma-separated Syncthing folder IDs to scan when using `ST_CRON` (global schedule). For per-folder schedules use `ST_FOLDER_CRON`. |
| `ST_CRON`            | _unset_                 | Global cron expression (5-field: `min hour dom mon dow`) that triggers scans for `ST_FOLDERS` (or `*` if unset).                    |
| `ST_FOLDER_CRON`     | _unset_                 | Per-folder schedules, one per line: `folderId: <cron expr>`.                                                                        |
| `SCAN_ON_STARTUP`    | `false`                 | Trigger scans immediately after startup.                                                                                            |
| `RUN_ONCE`           | `false`                 | Exit after the first scan (post-startup or scheduled).                                                                              |
| `DRY_RUN`            | `false`                 | Log the scans without calling the Syncthing API.                                                                                    |
| `ST_TLS_VERIFY`      | `true`                  | Verify TLS certificates when using HTTPS.                                                                                           |
| `ST_REQUEST_TIMEOUT` | _unset_                 | Optional HTTP request timeout in seconds (float).                                                                                   |
| `ST_STATUS_DELAY`    | `5`                     | Seconds to wait after triggering a scan before checking `/rest/db/status` for the folder.                                           |
| `TZ` / `CRON_TZ`     | _unset_                 | Timezone for cron evaluation (e.g. `Europe/Lisbon`).                                                                                |

## Notes

- Timezone is taken from `CRON_TZ` (preferred) or `TZ`.
- A follow-up status check is performed via `GET /rest/db/status?folder=<id>` a few seconds after triggering.

## Docker

```bash
docker build -t syncthing-kicker .
docker run --rm \
  -e ST_API_URL="http://syncthing:8384" \
  -e ST_API_KEY="<api-key>" \
  -e ST_CRON="0 5 * * 1,3,5" \
  -e ST_FOLDERS="default" \
  syncthing-kicker
```

### `docker-compose`

```yaml
services:
  syncthing-kicker:
    build: .
    environment:
      ST_API_URL: http://syncthing:8384
      ST_API_KEY: ${ST_API_KEY}
      TZ: Europe/Lisbon
      ST_CRON: "0 5 * * 1,3,5"
      ST_FOLDERS: default
    restart: unless-stopped
```

## Development

```bash
make deps
make test
make check
```

### Per-folder schedules

```bash
docker run --rm \
  -e ST_API_URL="http://syncthing:8384" \
  -e ST_API_KEY="<api-key>" \
  -e TZ="Europe/Lisbon" \
  -e ST_FOLDER_CRON=$'default: 0 * * * *\npictures: 30 2 * * *' \
  syncthing-kicker
```
