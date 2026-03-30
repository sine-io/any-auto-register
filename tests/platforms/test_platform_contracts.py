import importlib

from core.base_platform import Account, RegisterConfig
from platforms.chatgpt.plugin import ChatGPTPlatform
from platforms.cursor.plugin import CursorPlatform
from platforms.grok.plugin import GrokPlatform
from platforms.kiro.plugin import KiroPlatform
from platforms.trae.plugin import TraePlatform


PLATFORM_CLASSES = [
    ChatGPTPlatform,
    CursorPlatform,
    GrokPlatform,
    KiroPlatform,
    TraePlatform,
]


def test_primary_platform_plugins_expose_required_metadata_and_action_shapes():
    for platform_cls in PLATFORM_CLASSES:
        assert platform_cls.name
        assert platform_cls.display_name
        assert platform_cls.version
        assert isinstance(platform_cls.supported_executors, list)
        assert platform_cls.supported_executors

        instance = platform_cls(RegisterConfig())
        actions = instance.get_platform_actions()
        assert isinstance(actions, list)

        for action in actions:
            assert action["id"]
            assert action["label"]
            assert "params" in action
            assert isinstance(action["params"], list)


def test_cursor_execute_action_missing_token_returns_standard_error():
    instance = CursorPlatform(RegisterConfig())

    result = instance.execute_action(
        "switch_account",
        Account(platform="cursor", email="user@example.com", password="secret"),
        {},
    )

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"]


def test_trae_execute_action_missing_token_returns_standard_error():
    instance = TraePlatform(RegisterConfig())

    result = instance.execute_action(
        "switch_account",
        Account(platform="trae", email="user@example.com", password="secret"),
        {},
    )

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"]


def test_grok_execute_action_failure_returns_standard_error(monkeypatch):
    module = importlib.import_module("platforms.grok.grok2api_upload")
    monkeypatch.setattr(module, "upload_to_grok2api", lambda account: (False, "upload failed"))
    instance = GrokPlatform(RegisterConfig())

    result = instance.execute_action(
        "upload_grok2api",
        Account(
            platform="grok",
            email="user@example.com",
            password="secret",
            extra={"sso": "token"},
        ),
        {},
    )

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"] == "upload failed"


def test_chatgpt_execute_action_failure_returns_standard_error(monkeypatch):
    module = importlib.import_module("platforms.chatgpt.cpa_upload")
    monkeypatch.setattr(module, "generate_token_json", lambda account: {"token": "data"})
    monkeypatch.setattr(module, "upload_to_cpa", lambda token_data, api_url=None, api_key=None: (False, "upload failed"))
    instance = ChatGPTPlatform(RegisterConfig())

    result = instance.execute_action(
        "upload_cpa",
        Account(
            platform="chatgpt",
            email="user@example.com",
            password="secret",
            token="access-token",
            extra={"access_token": "access-token"},
        ),
        {"api_url": "https://example.com", "api_key": "secret"},
    )

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"] == "upload failed"


def test_kiro_execute_action_failure_returns_standard_error(monkeypatch):
    module = importlib.import_module("platforms.kiro.account_manager_upload")
    monkeypatch.setattr(module, "upload_to_kiro_manager", lambda account: (False, "manager upload failed"))
    instance = KiroPlatform(RegisterConfig())

    result = instance.execute_action(
        "upload_kiro_manager",
        Account(
            platform="kiro",
            email="user@example.com",
            password="secret",
            extra={"accessToken": "token"},
        ),
        {},
    )

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"] == "manager upload failed"
