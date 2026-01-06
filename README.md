# Syncthing Kicker

Syncthing Kicker is my hacky fix for the fact that Syncthing still hasn't implemented a way to _schedule_ folder re-scanning and sticks to intervals.Since using intervals is fiddly and Syncthing completely tanks my Synology NAS CPU (because Synology refuses to implement CPU limits for Docker containers), I wrote this small utility to let me schedule folder rescans at specific times of the day.

## Features

- Trigger Syncthing `rest/db/scan` for specific folders or all folders.
- Schedule rescans using cron expressions
- Optional one-shot execution, startup scans, dry-run mode, and TLS skip-verify.
- Configured entirely via environment variables.

## Installation

### Local

```bash
python3 -m pip install -r requirements.txt
```

## Environment variables

| Variable             | Default                 | Description                                                                                                                         |
| -------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `ST_API_URL`         | `http://127.0.0.1:8384` | Base URL for the Syncthing API (trailing slash optional).                                                                           |
| `ST_API_KEY`         | _required_              | Syncthing API key.                                                                                                                  |
| `ST_FOLDERS`         | `*`                     | Comma-separated Syncthing folder IDs to scan when using `ST_CRON` (global schedule). For per-folder schedules use `ST_FOLDER_CRON`. |
| `ST_CRON`            | _unset_                 | Global cron expression (5-field: `min hour dom mon dow`) that triggers scans for `ST_FOLDERS` (or `*` if unset).                    |
| `ST_FOLDER_CRON`     | _unset_                 | Per-folder schedules, one per line: `folderId: <cron expr>`.                                                                        |
| `SCAN_ON_STARTUP`    | `false`                 | Trigger scans immediately after startup. |
| `RUN_ONCE`           | `false`                 | Exit after the first scan (post-startup or scheduled).                                                                              |
| `DRY_RUN`            | `false`                 | Log the scans without calling the Syncthing API.                                                                                    |
| `ST_TLS_VERIFY`      | `true`                  | Verify TLS certificates when using HTTPS.                                                                                           |
| `ST_REQUEST_TIMEOUT` | _unset_                 | Optional HTTP request timeout in seconds (float).                                                                                   |
| `ST_STATUS_DELAY`    | `5`                     | Seconds to wait after triggering a scan before checking `/rest/db/status` for the folder.                                          |
| `LOG_LEVEL`          | `INFO`                  | Python logging level (`DEBUG`, `INFO`, `WARNING`, `ERROR`).                                                                         |
| `TZ` / `CRON_TZ`     | _unset_                 | Timezone for cron evaluation (e.g. `Europe/Lisbon`).                                                                                |

## Notes

- Cron expressions are evaluated using the `croniter` library (inside the container).
- Timezone is taken from `CRON_TZ` (preferred) or `TZ`.
- Syncthing may keep `POST /rest/db/scan` open until work completes. This service uses a *short* request timeout and treats timeouts as non-fatal.
- A follow-up status check is performed via `GET /rest/db/status?folder=<id>` a few seconds after triggering.

## Running locally

```bash
python3 -m pip install -r requirements.txt
python3 src/main.py
```

Export the environment variables above before running.

## Container usage

```bash
docker build -t syncthing-kicker .
docker run --rm \
  -e ST_API_URL="http://syncthing:8384" \
  -e ST_API_KEY="<api-key>" \
  -e ST_CRON="*/15 * * * *" \
  -e ST_FOLDERS="default" \
  syncthing-kicker
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

The container runs as an unprivileged user and executes `python /app/main.py` by
default.

If you change the project layout or Dockerfile entrypoint, ensure the path above matches what the image runs.

## GitHub Actions

A reusable GitHub Actions workflow is included to lint (basic syntax check) and
build the container image on pushes and pull requests. Update the workflow to
push to your own registry if desired.
