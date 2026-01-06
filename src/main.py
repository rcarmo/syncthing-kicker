#!/usr/bin/env python3
"""Syncthing folder scan trigger service."""

from __future__ import annotations

import asyncio
import argparse
import logging
import os
import re
import ssl
import sys
import time
import urllib.parse
from dataclasses import dataclass
from typing import Any, Dict, Iterable, List, Optional, Set, cast

import aiohttp
from croniter import croniter  # type: ignore
from dotenv import load_dotenv


@dataclass
class Settings:
    api_url: str
    api_key: str
    scan_on_startup: bool
    verify_tls: bool
    request_timeout: Optional[float]
    run_once: bool
    dry_run: bool
    cron_expr: str
    folder_cron: Dict[str, str]
    cron_timezone: Optional[str]


def _status_delay_seconds() -> float:
    """Delay (seconds) before checking status."""
    raw = os.environ.get("ST_STATUS_DELAY", "5").strip()
    try:
        value = float(raw)
        return value if value >= 0 else 5.0
    except ValueError:
        return 5.0


def _make_client_timeout(settings: Settings, default_total: float) -> aiohttp.ClientTimeout:
    # Keep this conservative; Syncthing can keep /db/scan open for a long time.
    total = settings.request_timeout if settings.request_timeout is not None else default_total
    return aiohttp.ClientTimeout(total=total)


def _build_req_kwargs(
    settings: Settings, *, headers: Dict[str, str], timeout: aiohttp.ClientTimeout
) -> Dict[str, Any]:
    ssl_ctx = _build_ssl_context(settings)
    req_kwargs: Dict[str, Any] = {"headers": headers, "timeout": timeout}
    if ssl_ctx is not None:
        req_kwargs["ssl"] = ssl_ctx
    return req_kwargs


class SyncthingApi:
    """Tiny wrapper for Syncthing REST calls."""

    def __init__(self, settings: Settings, session: aiohttp.ClientSession):
        self._settings = settings
        self._session = session

    def _url(self, path: str) -> str:
        return urllib.parse.urljoin(self._settings.api_url, path.lstrip("/"))

    async def request_json(
        self,
        method: str,
        path: str,
        *,
        query: Optional[Dict[str, str]] = None,
        timeout_total: float,
        accept_json: bool = True,
    ) -> tuple[int, Any]:
        url = self._url(path)
        if query:
            url = f"{url}?{urllib.parse.urlencode(query)}"

        headers = {"X-API-Key": self._settings.api_key}
        if accept_json:
            headers["Accept"] = "application/json"

        timeout = _make_client_timeout(self._settings, default_total=timeout_total)
        req_kwargs = _build_req_kwargs(self._settings, headers=headers, timeout=timeout)

        async with self._session.request(method.upper(), url, **req_kwargs) as resp:
            if resp.status >= 400:
                # try to include body for diagnostics
                try:
                    body = await resp.text()
                except Exception:  # pylint: disable=broad-except
                    body = "<unreadable body>"
                return resp.status, {"error": body}

            # Some endpoints return JSON with wrong/missing content-type.
            data = await resp.json(content_type=None)
            return resp.status, data

    async def request_text(
        self,
        method: str,
        path: str,
        *,
        query: Optional[Dict[str, str]] = None,
        timeout_total: float,
    ) -> tuple[int, str]:
        url = self._url(path)
        if query:
            url = f"{url}?{urllib.parse.urlencode(query)}"

        headers = {"X-API-Key": self._settings.api_key, "Accept": "application/json"}
        timeout = _make_client_timeout(self._settings, default_total=timeout_total)
        req_kwargs = _build_req_kwargs(self._settings, headers=headers, timeout=timeout)

        async with self._session.request(method.upper(), url, **req_kwargs) as resp:
            body = await resp.text()
            return resp.status, body


def _get_env_bool(name: str, default: bool = False) -> bool:
    value = os.environ.get(name)
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "on"}


_CRON_LINE_RE = re.compile(r"^(?P<folder>[^:]+):(?P<expr>.+)$")


def _parse_folder_cron(raw: Optional[str]) -> Dict[str, str]:
    """Parse the ST_FOLDER_CRON mapping."""

    if not raw:
        return {}

    folder_cron: Dict[str, str] = {}
    for line in raw.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue

        m = _CRON_LINE_RE.match(line)
        if not m:
            raise ValueError(
                "Invalid ST_FOLDER_CRON line. Expected 'folderId: <cron expr>'"
            )

        folder = m.group("folder").strip()
        expr = m.group("expr").strip()
        if not folder or not expr:
            raise ValueError(
                "Invalid ST_FOLDER_CRON line. Expected 'folderId: <cron expr>'"
            )
        folder_cron[folder] = expr

    return folder_cron


def load_settings() -> Settings:
    api_url = os.environ.get("ST_API_URL", "http://127.0.0.1:8384")
    if not api_url:
        raise ValueError("ST_API_URL must not be empty")
    api_url = api_url.rstrip("/") + "/"

    api_key = os.environ.get("ST_API_KEY")
    if not api_key:
        raise ValueError("ST_API_KEY environment variable is required")

    cron_expr = os.environ.get("ST_CRON", "").strip()
    folder_cron = _parse_folder_cron(os.environ.get("ST_FOLDER_CRON"))

    if not cron_expr and not folder_cron:
        raise ValueError(
            "Set ST_CRON (global cron schedule) and/or ST_FOLDER_CRON (per-folder schedules)."
        )

    # Backwards compatibility: if user still sets ST_FOLDERS, use it as a hint for global schedule.
    # For per-folder schedules, ST_FOLDER_CRON is authoritative.
    folders_raw = os.environ.get("ST_FOLDERS")
    folders = [part.strip() for part in (folders_raw or "").split(",") if part.strip()]
    if cron_expr and folders:
        logging.info("ST_FOLDERS is set; global schedule will scan: %s", ",".join(folders))
    if cron_expr and not folders:
        # Default for global schedule: scan all folders.
        folders = ["*"]

    return Settings(
        api_url=api_url,
        api_key=api_key,
        scan_on_startup=_get_env_bool("SCAN_ON_STARTUP", False),
        verify_tls=_get_env_bool("ST_TLS_VERIFY", True),
        request_timeout=float(os.environ["ST_REQUEST_TIMEOUT"])
        if os.environ.get("ST_REQUEST_TIMEOUT")
        else None,
        run_once=_get_env_bool("RUN_ONCE", False),
        dry_run=_get_env_bool("DRY_RUN", False),
        cron_expr=cron_expr,
        folder_cron=folder_cron,
        cron_timezone=os.environ.get("CRON_TZ") or os.environ.get("TZ"),
    )


def _build_ssl_context(settings: Settings) -> Optional[ssl.SSLContext]:
    if settings.verify_tls or not settings.api_url.startswith("https"):
        return None

    context = ssl.create_default_context()
    context.check_hostname = False
    context.verify_mode = ssl.CERT_NONE
    return context


async def _log_folder_status(api: SyncthingApi, folder_id: str) -> None:
    status, data = await api.request_json(
        "GET",
        "/rest/db/status",
        query={"folder": folder_id},
        timeout_total=10.0,
    )
    if status >= 400:
        err: Optional[str] = None
        if isinstance(data, dict):
            d = cast(Dict[str, Any], data)
            maybe_err = d.get("error")
            if isinstance(maybe_err, str):
                err = maybe_err
        logging.warning(
            "Folder %s status check failed: HTTP %s %s",
            folder_id,
            status,
            err,
        )
        return

    state = data.get("state")
    need_bytes = data.get("needBytes")
    in_sync = data.get("inSyncBytes")
    logging.info(
        "Folder %s status: state=%s needBytes=%s inSyncBytes=%s",
        folder_id,
        state,
        need_bytes,
        in_sync,
    )


async def _check_sync_status(settings: Settings, folders: List[str], delay_s: float = 5.0) -> None:
    """Report sync status after a scan."""

    await asyncio.sleep(delay_s)
    want_all = any(f.strip() == "*" for f in folders)

    try:
        async with aiohttp.ClientSession() as session:
            api = SyncthingApi(settings, session)

            folder_ids: List[str]
            if want_all:
                status, config = await api.request_json(
                    "GET", "/rest/system/config", timeout_total=15.0
                )
                if status >= 400:
                    err: Optional[str] = None
                    if isinstance(config, dict):
                        c = cast(Dict[str, Any], config)
                        maybe_err = c.get("error")
                        if isinstance(maybe_err, str):
                            err = maybe_err
                    logging.warning(
                        "Failed to fetch folder list for wildcard status check: HTTP %s %s",
                        status,
                        err,
                    )
                    return

                folders_cfg: List[Any] = []
                if isinstance(config, dict):
                    c = cast(Dict[str, Any], config)
                    raw_folders: Any = c.get("folders")
                    if isinstance(raw_folders, list):
                        folders_cfg = cast(List[Any], raw_folders)

                folder_ids = []
                for f in folders_cfg:
                    if not isinstance(f, dict):
                        continue
                    fd = cast(Dict[str, Any], f)
                    maybe_id: Any = fd.get("id")
                    if isinstance(maybe_id, str) and maybe_id:
                        folder_ids.append(maybe_id)
                if not folder_ids:
                    logging.warning("No folders returned by Syncthing config; nothing to report")
                    return
            else:
                folder_ids = [f.strip() for f in folders if f.strip() and f.strip() != "*"]

            for folder_id in folder_ids:
                await _log_folder_status(api, folder_id)

    except asyncio.TimeoutError:
        logging.warning("Status check timed out")
    except Exception as exc:  # pylint: disable=broad-except
        logging.error("Error checking sync status: %s", exc)


async def _post_scan(
    settings: Settings,
    session: aiohttp.ClientSession,
    folder: Optional[str],
) -> None:
    path = "rest/db/scan"

    url = urllib.parse.urljoin(settings.api_url, path)
    if folder and folder != "*":
        query = urllib.parse.urlencode({"folder": folder})
        url = f"{url}?{query}"

    headers = {"X-API-Key": settings.api_key, "Accept": "application/json"}

    if settings.dry_run:
        logging.info("[dry-run] Would trigger scan: url=%s", url)
        return

    # Syncthing may hold this POST open until sync completes. Keep timeout low and
    # treat timeouts as success (request likely accepted).
    timeout = _make_client_timeout(settings, default_total=5.0)

    try:
        req_kwargs = _build_req_kwargs(settings, headers=headers, timeout=timeout)
        async with session.post(url, **req_kwargs) as resp:
            # Don't wait for anything beyond headers/short response.
            logging.info(
                "Triggered scan for folder '%s': HTTP %s",
                folder or "*",
                resp.status,
            )
            if resp.status >= 400:
                body = await resp.text()
                logging.error(
                    "Scan trigger failed for folder '%s': HTTP %s %s",
                    folder or "*",
                    resp.status,
                    body,
                )
    except asyncio.TimeoutError:
        logging.warning(
            "Scan trigger for folder '%s' timed out after %.1fs; Syncthing may still be processing",
            folder or "*",
            timeout.total if timeout.total is not None else 0.0,
        )
    except aiohttp.ClientError as exc:
        logging.error("Failed to reach Syncthing API for folder '%s': %s", folder or "*", exc)


async def trigger_scans_for(
    settings: Settings,
    session: aiohttp.ClientSession,
    folders: Iterable[str],
    pending_tasks: Set[asyncio.Task[Any]],
    status_delay_s: float,
) -> None:
    for folder in folders:
        folder_id = None if folder == "*" else folder
        await _post_scan(settings, session, folder_id)
        # fire-and-forget: check status shortly after triggering (tracked)
        # Pass the original folder selection to status checker, which will decide
        # whether to report per-folder or global statuses.
        task = asyncio.create_task(
            _check_sync_status(settings, [folder_id or "*"], delay_s=status_delay_s)
        )
        pending_tasks.add(task)
        task.add_done_callback(pending_tasks.discard)


async def run_service(settings: Settings) -> None:
    logging.info("Syncthing kicker starting with settings: %s", settings)

    pending_tasks: Set[asyncio.Task[Any]] = set()
    status_delay_s = _status_delay_seconds()

    if settings.scan_on_startup:
        logging.info("Triggering scan on startup")
        # Startup scan triggers everything that is configured.
        async with aiohttp.ClientSession() as session:
            if settings.cron_expr:
                await trigger_scans_for(
                    settings,
                    session,
                    os.environ.get("ST_FOLDERS", "*").split(","),
                    pending_tasks,
                    status_delay_s,
                )
            for folder in settings.folder_cron.keys():
                await trigger_scans_for(
                    settings, session, [folder], pending_tasks, status_delay_s
                )
        if settings.run_once:
            logging.info("RUN_ONCE set; exiting after startup scan")
            return

    try:
        await _run_scheduler(settings, pending_tasks, status_delay_s)
    finally:
        await _drain_pending_tasks(pending_tasks, timeout_s=30.0)


async def _drain_pending_tasks(pending_tasks: Set[asyncio.Task[Any]], timeout_s: float) -> None:
    """Finish pending background tasks."""
    if not pending_tasks:
        return

    try:
        await asyncio.wait(pending_tasks, timeout=timeout_s)
    finally:
        for task in list(pending_tasks):
            if not task.done():
                task.cancel()
        if pending_tasks:
            await asyncio.gather(*pending_tasks, return_exceptions=True)


def _next_epoch(expr: str, base_epoch: float) -> float:
    # croniter operates on naive datetimes; we interpret them in *local* time
    # by using datetime.fromtimestamp(base_epoch) without tzinfo.
    import datetime as dt

    base_dt = dt.datetime.fromtimestamp(base_epoch)
    it = croniter(expr, base_dt)
    nxt = it.get_next(dt.datetime)
    return nxt.timestamp()


async def _run_scheduler(
    settings: Settings,
    pending_tasks: Set[asyncio.Task[Any]],
    status_delay_s: float,
) -> None:
    folders_raw = os.environ.get("ST_FOLDERS", "*")
    global_folders = [f.strip() for f in folders_raw.split(",") if f.strip()] or ["*"]

    schedules: List[tuple[str, List[str]]] = []
    if settings.cron_expr:
        schedules.append((settings.cron_expr, global_folders))
    for folder, expr in settings.folder_cron.items():
        schedules.append((expr, [folder]))

    if not schedules:
        raise ValueError("No schedules configured (check ST_CRON / ST_FOLDER_CRON).")

    logging.info("Scheduler timezone: %s", settings.cron_timezone or "(system default)")

    while True:
        now_epoch = time.time()
        # pick next event across all schedules
        next_events: List[tuple[float, List[str], str]] = []
        for expr, folders in schedules:
            nxt = _next_epoch(expr, now_epoch)
            next_events.append((nxt, folders, expr))

        next_events.sort(key=lambda x: x[0])
        next_epoch, folders, expr = next_events[0]
        delay = max(0.0, next_epoch - time.time())
        logging.info(
            "Next scan at %s (in %.0fs) for %s via '%s'",
            time.strftime("%Y-%m-%d %H:%M:%S", time.localtime(next_epoch)),
            delay,
            ",".join(folders),
            expr,
        )
        await asyncio.sleep(delay)

        async with aiohttp.ClientSession() as session:
            await trigger_scans_for(
                settings, session, folders, pending_tasks, status_delay_s
            )
        if settings.run_once:
            logging.info("RUN_ONCE set; exiting after scheduled scan")
            break


def _configure_logging() -> None:
    log_level = os.environ.get("LOG_LEVEL", "INFO").upper()
    logging.basicConfig(
        level=log_level,
        format="%(asctime)s %(levelname)s %(message)s",
    )


def main() -> int:
    _configure_logging()

    # Load environment variables from .env, if present.
    load_dotenv(override=False)

    parser = argparse.ArgumentParser(add_help=True)
    parser.add_argument(
        "--check",
        action="store_true",
        help="Check Syncthing folder status and exit.",
    )
    args = parser.parse_args()
    try:
        settings = load_settings()
    except Exception as exc:  # pylint: disable=broad-except
        logging.error("Failed to load settings: %s", exc)
        return 1

    if args.check:
        folders_raw = os.environ.get("ST_FOLDERS", "*")
        folders = [f.strip() for f in folders_raw.split(",") if f.strip()] or ["*"]
        try:
            asyncio.run(_check_sync_status(settings, folders, delay_s=0.0))
        except KeyboardInterrupt:
            logging.info("Received interrupt, shutting down.")
        except Exception as exc:  # pylint: disable=broad-except
            logging.exception("Unexpected error: %s", exc)
            return 1
        return 0

    try:
        asyncio.run(run_service(settings))
    except KeyboardInterrupt:
        logging.info("Received interrupt, shutting down.")
    except Exception as exc:  # pylint: disable=broad-except
        logging.exception("Unexpected error: %s", exc)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
