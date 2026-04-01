import importlib
import sys
from types import ModuleType, SimpleNamespace

from sqlmodel import Session


def _fresh_import(module_name: str):
    sys.modules.pop(module_name, None)
    importlib.invalidate_caches()
    return importlib.import_module(module_name)


def _install_fake_module(monkeypatch, module_name: str, **attrs):
    module = ModuleType(module_name)
    for key, value in attrs.items():
        setattr(module, key, value)
    monkeypatch.setitem(sys.modules, module_name, module)
    return module


def _make_chatgpt_account(session, db_module, **overrides):
    account = db_module.AccountModel(
        platform="chatgpt",
        email=overrides.pop("email", "user@example.com"),
        password=overrides.pop("password", "secret"),
        token=overrides.pop("token", "db-token"),
        status=overrides.pop("status", "registered"),
        **overrides,
    )
    account.set_extra(
        {
            "access_token": "extra-access-token",
            "refresh_token": "extra-refresh-token",
            "id_token": "id-token",
            "session_token": "session-token",
            "cookies": "oai-did=device-id; session=abc",
        }
    )
    session.add(account)
    session.commit()
    session.refresh(account)
    return account


def _make_grok_account(session, db_module, **overrides):
    account = db_module.AccountModel(
        platform="grok",
        email=overrides.pop("email", "grok@example.com"),
        password=overrides.pop("password", "secret"),
        token=overrides.pop("token", ""),
        status=overrides.pop("status", "registered"),
        **overrides,
    )
    account.set_extra({"sso": "token", "sso_rw": "token-rw"})
    session.add(account)
    session.commit()
    session.refresh(account)
    return account


def test_chatgpt_api_refresh_token_uses_token_service_raw_method(isolated_modules, monkeypatch):
    api_module = _fresh_import("api.chatgpt")
    token_service_module = _fresh_import("platforms.chatgpt.services.token")

    raw_calls = []
    legacy_calls = []
    service_access_token = "service-access-token-1234567890-abcdefghijklmnopqrstuvwxyz"
    legacy_access_token = "legacy-access-token-1234567890-abcdefghijklmnopqrstuvwxyz"

    def fake_refresh_account_raw(self, account, proxy=None):
        raw_calls.append(
            {
                "email": account.email,
                "access_token": account.access_token,
                "refresh_token": account.refresh_token,
                "id_token": account.id_token,
                "session_token": account.session_token,
                "cookies": account.cookies,
                "proxy": proxy,
            }
        )
        return SimpleNamespace(
            success=True,
            access_token=service_access_token,
            refresh_token="service-refresh-token",
            error_message="",
        )

    class FakeTokenRefreshManager:
        def __init__(self, proxy_url=None):
            legacy_calls.append({"proxy_url": proxy_url})

        def refresh_account(self, account):
            legacy_calls[-1].update(
                {
                    "email": account.email,
                    "access_token": account.access_token,
                    "refresh_token": account.refresh_token,
                }
            )
            return SimpleNamespace(
                success=True,
                access_token=legacy_access_token,
                refresh_token="legacy-refresh-token",
                error_message="",
            )

    monkeypatch.setattr(
        token_service_module.ChatGPTTokenService,
        "refresh_account_raw",
        fake_refresh_account_raw,
        raising=False,
    )
    _install_fake_module(monkeypatch, "platforms.chatgpt.token_refresh", TokenRefreshManager=FakeTokenRefreshManager)

    with Session(isolated_modules.db.engine) as session:
        account = _make_chatgpt_account(session, isolated_modules.db)

        result = api_module.refresh_token(
            account.id,
            proxy="http://proxy.example.com",
            session=session,
        )

        persisted = session.get(isolated_modules.db.AccountModel, account.id)

    assert result == {
        "ok": True,
        "access_token": service_access_token[:40] + "...",
    }
    assert raw_calls == [
        {
            "email": "user@example.com",
            "access_token": "extra-access-token",
            "refresh_token": "extra-refresh-token",
            "id_token": "id-token",
            "session_token": "session-token",
            "cookies": "oai-did=device-id; session=abc",
            "proxy": "http://proxy.example.com",
        }
    ]
    assert legacy_calls == []
    assert persisted.token == service_access_token
    assert persisted.get_extra()["access_token"] == service_access_token
    assert persisted.get_extra()["refresh_token"] == "service-refresh-token"



def test_chatgpt_api_payment_link_uses_billing_service_raw_method(isolated_modules, monkeypatch):
    api_module = _fresh_import("api.chatgpt")
    billing_service_module = _fresh_import("platforms.chatgpt.services.billing")

    raw_calls = []
    legacy_calls = []

    def fake_generate_payment_link_raw(
        self,
        account,
        plan,
        country,
        proxy=None,
        workspace_name="MyTeam",
        seat_quantity=5,
        price_interval="month",
    ):
        raw_calls.append(
            {
                "email": account.email,
                "access_token": account.access_token,
                "cookies": account.cookies,
                "plan": plan,
                "country": country,
                "proxy": proxy,
                "workspace_name": workspace_name,
                "seat_quantity": seat_quantity,
                "price_interval": price_interval,
            }
        )
        return "https://service.example.com/team-link"

    def fake_generate_plus_link(account, proxy=None, country="SG"):
        legacy_calls.append(
            {
                "plan": "plus",
                "email": account.email,
                "proxy": proxy,
                "country": country,
            }
        )
        return "https://legacy.example.com/plus-link"

    def fake_generate_team_link(
        account,
        workspace_name="MyTeam",
        seat_quantity=5,
        price_interval="month",
        proxy=None,
        country="SG",
    ):
        legacy_calls.append(
            {
                "plan": "team",
                "email": account.email,
                "proxy": proxy,
                "country": country,
                "workspace_name": workspace_name,
                "seat_quantity": seat_quantity,
                "price_interval": price_interval,
            }
        )
        return "https://legacy.example.com/team-link"

    monkeypatch.setattr(
        billing_service_module.ChatGPTBillingService,
        "generate_payment_link_raw",
        fake_generate_payment_link_raw,
        raising=False,
    )
    _install_fake_module(
        monkeypatch,
        "platforms.chatgpt.payment",
        generate_plus_link=fake_generate_plus_link,
        generate_team_link=fake_generate_team_link,
    )

    with Session(isolated_modules.db.engine) as session:
        account = _make_chatgpt_account(session, isolated_modules.db)

        result = api_module.generate_payment_link(
            account.id,
            api_module.PaymentReq(
                plan="team",
                country="JP",
                proxy="http://proxy.example.com",
                workspace_name="My Squad",
                seat_quantity=8,
                price_interval="year",
            ),
            session=session,
        )

    assert result == {
        "url": "https://service.example.com/team-link",
        "plan": "team",
        "country": "JP",
    }
    assert raw_calls == [
        {
            "email": "user@example.com",
            "access_token": "extra-access-token",
            "cookies": "oai-did=device-id; session=abc",
            "plan": "team",
            "country": "JP",
            "proxy": "http://proxy.example.com",
            "workspace_name": "My Squad",
            "seat_quantity": 8,
            "price_interval": "year",
        }
    ]
    assert legacy_calls == []



def test_chatgpt_api_subscription_uses_token_service_raw_method(isolated_modules, monkeypatch):
    api_module = _fresh_import("api.chatgpt")
    token_service_module = _fresh_import("platforms.chatgpt.services.token")

    raw_calls = []
    legacy_calls = []

    def fake_get_subscription_status_raw(self, account, proxy=None):
        raw_calls.append(
            {
                "email": account.email,
                "access_token": account.access_token,
                "cookies": account.cookies,
                "proxy": proxy,
            }
        )
        return "team"

    def fake_check_subscription_status(account, proxy=None):
        legacy_calls.append(
            {
                "email": account.email,
                "access_token": account.access_token,
                "cookies": account.cookies,
                "proxy": proxy,
            }
        )
        return "expired"

    monkeypatch.setattr(
        token_service_module.ChatGPTTokenService,
        "get_subscription_status_raw",
        fake_get_subscription_status_raw,
        raising=False,
    )
    _install_fake_module(
        monkeypatch,
        "platforms.chatgpt.payment",
        check_subscription_status=fake_check_subscription_status,
    )

    with Session(isolated_modules.db.engine) as session:
        account = _make_chatgpt_account(session, isolated_modules.db, status="registered")

        result = api_module.check_subscription(
            account.id,
            proxy="http://proxy.example.com",
            session=session,
        )

        persisted = session.get(isolated_modules.db.AccountModel, account.id)

    assert result == {
        "subscription": "team",
        "email": "user@example.com",
    }
    assert raw_calls == [
        {
            "email": "user@example.com",
            "access_token": "extra-access-token",
            "cookies": "oai-did=device-id; session=abc",
            "proxy": "http://proxy.example.com",
        }
    ]
    assert legacy_calls == []
    assert persisted.status == "team"



def test_chatgpt_api_upload_cpa_uses_external_sync_service_raw_method(isolated_modules, monkeypatch):
    api_module = _fresh_import("api.chatgpt")
    external_sync_service_module = _fresh_import("platforms.chatgpt.services.external_sync")

    raw_calls = []
    legacy_calls = []

    def fake_upload_cpa_raw(self, account, api_url=None, api_key=None):
        raw_calls.append(
            {
                "email": account.email,
                "access_token": account.access_token,
                "refresh_token": account.refresh_token,
                "id_token": account.id_token,
                "api_url": api_url,
                "api_key": api_key,
            }
        )
        return True, "service CPA 上传成功"

    def fake_generate_token_json(account):
        legacy_calls.append(
            {
                "step": "generate_token_json",
                "email": account.email,
                "access_token": account.access_token,
            }
        )
        return {"email": account.email, "access_token": account.access_token}

    def fake_upload_to_cpa(token_data, api_url=None, api_key=None):
        legacy_calls.append(
            {
                "step": "upload_to_cpa",
                "token_data": token_data,
                "api_url": api_url,
                "api_key": api_key,
            }
        )
        return True, "legacy CPA 上传成功"

    monkeypatch.setattr(
        external_sync_service_module.ChatGPTExternalSyncService,
        "upload_cpa_raw",
        fake_upload_cpa_raw,
        raising=False,
    )
    _install_fake_module(
        monkeypatch,
        "platforms.chatgpt.cpa_upload",
        generate_token_json=fake_generate_token_json,
        upload_to_cpa=fake_upload_to_cpa,
    )

    with Session(isolated_modules.db.engine) as session:
        account = _make_chatgpt_account(session, isolated_modules.db)

        result = api_module.upload_cpa(
            account.id,
            api_module.CpaUploadReq(
                api_url="https://cpa.example.com",
                api_key="secret-key",
            ),
            session=session,
        )

    assert result == {
        "ok": True,
        "message": "service CPA 上传成功",
    }
    assert raw_calls == [
        {
            "email": "user@example.com",
            "access_token": "extra-access-token",
            "refresh_token": "extra-refresh-token",
            "id_token": "id-token",
            "api_url": "https://cpa.example.com",
            "api_key": "secret-key",
        }
    ]
    assert legacy_calls == []



def test_external_sync_routes_grok_through_grok_sync_service_raw_method(isolated_modules, monkeypatch):
    legacy_calls = []
    _install_fake_module(
        monkeypatch,
        "platforms.grok.grok2api_upload",
        upload_to_grok2api=lambda *args, **kwargs: legacy_calls.append((args, kwargs)) or (False, "legacy direct upload path"),
    )

    external_sync_module = _fresh_import("services.external_sync")
    grok_sync_service_module = _fresh_import("platforms.grok.services.sync")

    call_sequence = []
    raw_calls = []

    class FakeConfigStore:
        def get(self, key, default=""):
            if key == "grok2api_url":
                return "http://127.0.0.1:8011"
            return default

    def fake_ensure_grok2api_ready():
        call_sequence.append("ensure")
        return True, "ready"

    def fake_upload_grok2api_raw(self, account, api_url=None, app_key=None):
        call_sequence.append("raw")
        raw_calls.append(
            {
                "account": account,
                "api_url": api_url,
                "app_key": app_key,
            }
        )
        return True, "service upload ok"

    monkeypatch.setattr(isolated_modules.config_store, "config_store", FakeConfigStore())
    monkeypatch.setattr(
        grok_sync_service_module.GrokSyncService,
        "upload_grok2api_raw",
        fake_upload_grok2api_raw,
        raising=False,
    )
    _install_fake_module(
        monkeypatch,
        "services.grok2api_runtime",
        ensure_grok2api_ready=fake_ensure_grok2api_ready,
    )

    account = isolated_modules.base_platform.Account(
        platform="grok",
        email="grok@example.com",
        password="secret",
        extra={"sso": "token"},
    )

    result = external_sync_module.sync_account(account)

    assert result == [{"name": "grok2api", "ok": True, "msg": "service upload ok"}]
    assert call_sequence == ["ensure", "raw"]
    assert raw_calls == [{"account": account, "api_url": None, "app_key": None}]
    assert legacy_calls == []



def test_integrations_backfill_routes_grok_through_grok_sync_service_raw_method(isolated_modules, monkeypatch):
    legacy_calls = []
    _install_fake_module(
        monkeypatch,
        "platforms.grok.grok2api_upload",
        upload_to_grok2api=lambda *args, **kwargs: legacy_calls.append((args, kwargs)) or (False, "legacy direct upload path"),
    )

    integrations_module = _fresh_import("api.integrations")
    grok_sync_service_module = _fresh_import("platforms.grok.services.sync")

    call_sequence = []
    raw_calls = []

    class FakeConfigStore:
        def get(self, key, default=""):
            if key == "grok2api_url":
                return "https://grok2api.example.com"
            if key == "grok2api_app_key":
                return "service-app-key"
            return default

    def fake_ensure_grok2api_ready():
        call_sequence.append("ensure")
        return True, "ready"

    def fake_upload_grok2api_raw(self, account, api_url=None, app_key=None):
        call_sequence.append("raw")
        raw_calls.append(
            {
                "email": account.email,
                "api_url": api_url,
                "app_key": app_key,
            }
        )
        return True, "service backfill ok"

    monkeypatch.setattr(isolated_modules.config_store, "config_store", FakeConfigStore())
    monkeypatch.setattr(
        grok_sync_service_module.GrokSyncService,
        "upload_grok2api_raw",
        fake_upload_grok2api_raw,
        raising=False,
    )
    _install_fake_module(
        monkeypatch,
        "services.grok2api_runtime",
        ensure_grok2api_ready=fake_ensure_grok2api_ready,
    )

    with Session(isolated_modules.db.engine) as session:
        _make_grok_account(session, isolated_modules.db)

    result = integrations_module.backfill_integrations(
        integrations_module.BackfillRequest(platforms=["grok"])
    )

    assert result == {
        "total": 1,
        "success": 1,
        "failed": 0,
        "items": [
            {
                "platform": "grok",
                "email": "grok@example.com",
                "results": [
                    {
                        "name": "grok2api",
                        "ok": True,
                        "msg": "service backfill ok",
                    }
                ],
            }
        ],
    }
    assert call_sequence == ["ensure", "raw"]
    assert raw_calls == [
        {
            "email": "grok@example.com",
            "api_url": "https://grok2api.example.com",
            "app_key": "service-app-key",
        }
    ]
    assert legacy_calls == []


def test_chatgpt_token_service_refresh_account_raw_uses_request_proxy_and_duck_typed_account(monkeypatch):
    token_service_module = _fresh_import("platforms.chatgpt.services.token")

    captured = {}

    class FakeTokenRefreshManager:
        def __init__(self, proxy_url=None):
            captured["proxy_url"] = proxy_url

        def refresh_account(self, account):
            captured["email"] = account.email
            captured["access_token"] = account.access_token
            captured["refresh_token"] = account.refresh_token
            captured["id_token"] = account.id_token
            captured["session_token"] = account.session_token
            captured["client_id"] = account.client_id
            captured["cookies"] = account.cookies
            return SimpleNamespace(
                success=True,
                access_token="fresh-access-token",
                refresh_token="fresh-refresh-token",
                error_message="",
            )

    monkeypatch.setattr(token_service_module, "TokenRefreshManager", FakeTokenRefreshManager)

    service = token_service_module.ChatGPTTokenService(
        token_service_module.RegisterConfig(proxy="http://config-proxy.example.com")
    )
    account = SimpleNamespace(
        email="user@example.com",
        access_token="duck-access-token",
        refresh_token="duck-refresh-token",
        id_token="duck-id-token",
        session_token="duck-session-token",
        client_id="duck-client-id",
        cookies="oai-did=device-id; session=abc",
    )

    result = service.refresh_account_raw(account, proxy="http://request-proxy.example.com")

    assert result.success is True
    assert captured == {
        "proxy_url": "http://request-proxy.example.com",
        "email": "user@example.com",
        "access_token": "duck-access-token",
        "refresh_token": "duck-refresh-token",
        "id_token": "duck-id-token",
        "session_token": "duck-session-token",
        "client_id": "duck-client-id",
        "cookies": "oai-did=device-id; session=abc",
    }


def test_chatgpt_token_service_get_subscription_status_raw_uses_proxy_override(monkeypatch):
    token_service_module = _fresh_import("platforms.chatgpt.services.token")

    captured = {}

    def fake_check_subscription_status(account, proxy=None):
        captured["email"] = account.email
        captured["access_token"] = account.access_token
        captured["cookies"] = account.cookies
        captured["proxy"] = proxy
        return "team"

    monkeypatch.setattr(token_service_module, "check_subscription_status", fake_check_subscription_status)

    service = token_service_module.ChatGPTTokenService(
        token_service_module.RegisterConfig(proxy="http://config-proxy.example.com")
    )
    account = SimpleNamespace(
        email="user@example.com",
        access_token="duck-access-token",
        cookies="oai-did=device-id; session=abc",
    )

    result = service.get_subscription_status_raw(account, proxy="http://request-proxy.example.com")

    assert result == "team"
    assert captured == {
        "email": "user@example.com",
        "access_token": "duck-access-token",
        "cookies": "oai-did=device-id; session=abc",
        "proxy": "http://request-proxy.example.com",
    }


def test_chatgpt_billing_service_generate_payment_link_raw_preserves_team_params(monkeypatch):
    billing_service_module = _fresh_import("platforms.chatgpt.services.billing")

    captured = {}

    def fake_generate_team_link(
        account,
        workspace_name="MyTeam",
        price_interval="month",
        seat_quantity=5,
        proxy=None,
        country="SG",
    ):
        captured["email"] = account.email
        captured["access_token"] = account.access_token
        captured["cookies"] = account.cookies
        captured["workspace_name"] = workspace_name
        captured["price_interval"] = price_interval
        captured["seat_quantity"] = seat_quantity
        captured["proxy"] = proxy
        captured["country"] = country
        return "https://service.example.com/team-link"

    monkeypatch.setattr(billing_service_module, "generate_team_link", fake_generate_team_link)

    service = billing_service_module.ChatGPTBillingService(
        billing_service_module.RegisterConfig(proxy="http://config-proxy.example.com")
    )
    account = SimpleNamespace(
        email="user@example.com",
        access_token="duck-access-token",
        cookies="oai-did=device-id; session=abc",
    )

    result = service.generate_payment_link_raw(
        account,
        plan="team",
        country="JP",
        proxy="http://request-proxy.example.com",
        workspace_name="My Squad",
        seat_quantity=8,
        price_interval="year",
    )

    assert result == "https://service.example.com/team-link"
    assert captured == {
        "email": "user@example.com",
        "access_token": "duck-access-token",
        "cookies": "oai-did=device-id; session=abc",
        "workspace_name": "My Squad",
        "price_interval": "year",
        "seat_quantity": 8,
        "proxy": "http://request-proxy.example.com",
        "country": "JP",
    }


def test_chatgpt_external_sync_service_upload_cpa_raw_returns_tuple(monkeypatch):
    sync_service_module = _fresh_import("platforms.chatgpt.services.external_sync")

    captured = {}

    def fake_generate_token_json(account):
        captured["email"] = account.email
        captured["access_token"] = account.access_token
        captured["refresh_token"] = account.refresh_token
        captured["id_token"] = account.id_token
        return {"email": account.email, "access_token": account.access_token}

    def fake_upload_to_cpa(token_data, api_url=None, api_key=None):
        captured["token_data"] = token_data
        captured["api_url"] = api_url
        captured["api_key"] = api_key
        return True, "CPA raw 上传成功"

    monkeypatch.setattr(sync_service_module, "generate_token_json", fake_generate_token_json)
    monkeypatch.setattr(sync_service_module, "upload_to_cpa", fake_upload_to_cpa)

    service = sync_service_module.ChatGPTExternalSyncService()
    account = SimpleNamespace(
        email="user@example.com",
        access_token="duck-access-token",
        refresh_token="duck-refresh-token",
        id_token="duck-id-token",
    )

    result = service.upload_cpa_raw(
        account,
        api_url="https://cpa.example.com",
        api_key="secret-key",
    )

    assert result == (True, "CPA raw 上传成功")
    assert captured == {
        "email": "user@example.com",
        "access_token": "duck-access-token",
        "refresh_token": "duck-refresh-token",
        "id_token": "duck-id-token",
        "token_data": {
            "email": "user@example.com",
            "access_token": "duck-access-token",
        },
        "api_url": "https://cpa.example.com",
        "api_key": "secret-key",
    }


def test_grok_sync_service_upload_grok2api_raw_uses_explicit_args(monkeypatch):
    grok_sync_module = _fresh_import("platforms.grok.services.sync")

    captured = {}

    def fake_upload_to_grok2api(account, api_url=None, app_key=None):
        captured["account"] = account
        captured["api_url"] = api_url
        captured["app_key"] = app_key
        return True, "raw upload ok"

    monkeypatch.setattr(grok_sync_module, "upload_to_grok2api", fake_upload_to_grok2api)

    service = grok_sync_module.GrokSyncService()
    account = SimpleNamespace(email="grok@example.com", extra={"sso": "token"})

    result = service.upload_grok2api_raw(
        account,
        api_url="https://grok2api.example.com",
        app_key="service-app-key",
    )

    assert result == (True, "raw upload ok")
    assert captured == {
        "account": account,
        "api_url": "https://grok2api.example.com",
        "app_key": "service-app-key",
    }


def test_grok_sync_service_upload_grok2api_raw_uses_lower_layer_fallback_when_args_omitted(monkeypatch):
    grok_sync_module = _fresh_import("platforms.grok.services.sync")

    captured = {}

    def fake_upload_to_grok2api(account):
        captured["account"] = account
        return True, "fallback upload ok"

    monkeypatch.setattr(grok_sync_module, "upload_to_grok2api", fake_upload_to_grok2api)

    service = grok_sync_module.GrokSyncService()
    account = SimpleNamespace(email="grok@example.com", extra={"sso": "token"})

    result = service.upload_grok2api_raw(account)

    assert result == (True, "fallback upload ok")
    assert captured == {"account": account}
