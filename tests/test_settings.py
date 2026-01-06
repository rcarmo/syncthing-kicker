import os
from unittest import mock

import pytest

from src.main import Settings, _parse_folder_cron, load_settings


@pytest.fixture(autouse=True)
def clear_env():
    with mock.patch.dict(os.environ, {}, clear=True):
        yield


def test_dotenv_importable():
    import dotenv  # noqa: F401


def test_parse_folder_cron_basic():
    raw = """
    folderA: */5 * * * *
    folderB: 0 0 * * 1
    """.strip()
    result = _parse_folder_cron(raw)
    assert result == {"folderA": "*/5 * * * *", "folderB": "0 0 * * 1"}


def test_parse_folder_cron_ignores_comments_and_blank_lines():
    raw = """
    # comment

    folderA: */5 * * * *
    folderB: 0 0 * * 1
    """.strip()
    result = _parse_folder_cron(raw)
    assert result == {"folderA": "*/5 * * * *", "folderB": "0 0 * * 1"}


def test_parse_folder_cron_invalid_line():
    raw = "invalid line"
    with pytest.raises(ValueError):
        _parse_folder_cron(raw)


def test_load_settings_requires_api_key():
    with pytest.raises(ValueError, match="ST_API_KEY"):
        load_settings()


def test_load_settings_populates_defaults():
    env = {
        "ST_API_KEY": "abc123",
        "ST_CRON": "*/5 * * * *",
        "ST_FOLDERS": "folder1, folder2",
    }
    with mock.patch.dict(os.environ, env, clear=True):
        settings = load_settings()

    assert isinstance(settings, Settings)
    assert settings.api_url == "http://127.0.0.1:8384/"
    assert settings.api_key == "abc123"
    assert settings.cron_expr == "*/5 * * * *"
    assert settings.folder_cron == {}
    assert settings.scan_on_startup is False
    assert settings.verify_tls is True
    assert settings.request_timeout is None
    assert settings.run_once is False
    assert settings.dry_run is False
    assert settings.cron_timezone is None


def test_load_settings_requires_schedule():
    env = {
        "ST_API_KEY": "abc123",
    }
    with mock.patch.dict(os.environ, env, clear=True):
        with pytest.raises(ValueError, match="ST_CRON"):
            load_settings()


def test_load_settings_folder_cron_only():
    env = {
        "ST_API_KEY": "abc123",
        "ST_FOLDER_CRON": "folderA: */5 * * * *\n",
    }
    with mock.patch.dict(os.environ, env, clear=True):
        settings = load_settings()

    assert settings.cron_expr == ""
    assert settings.folder_cron == {"folderA": "*/5 * * * *"}


def test_parse_folder_cron_empty_returns_empty_dict():
    assert _parse_folder_cron(None) == {}
    assert _parse_folder_cron("") == {}
