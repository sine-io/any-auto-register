from types import SimpleNamespace

from core.base_mailbox import MailboxAccount
from core.base_platform import Account, AccountStatus, RegisterConfig


DEFAULT_CHATGPT_CLIENT_ID = "app_EMoamEEZ73f0CkXaXp7hrann"


def _make_chatgpt_account(*, token: str = "fallback-access", extra: dict | None = None) -> Account:
    return Account(
        platform="chatgpt",
        email="user@example.com",
        password="secret",
        user_id="acct-1",
        token=token,
        status=AccountStatus.REGISTERED,
        extra=extra or {},
    )


def test_chatgpt_registration_service_uses_generic_mailbox_adapter(monkeypatch):
    from platforms.chatgpt.services.registration import ChatGPTRegistrationService
    import platforms.chatgpt.services.registration as registration_module

    class FakeMailbox:
        def __init__(self):
            self.get_email_calls = 0
            self.wait_calls = []

        def get_email(self):
            self.get_email_calls += 1
            return MailboxAccount(email="fixed@example.com", account_id="mailbox-1")

        def wait_for_code(self, account, keyword="", timeout=120, otp_sent_at=None, exclude_codes=None, **kwargs):
            self.wait_calls.append(
                {
                    "email": account.email,
                    "account_id": account.account_id,
                    "keyword": keyword,
                    "timeout": timeout,
                    "otp_sent_at": otp_sent_at,
                    "exclude_codes": exclude_codes,
                }
            )
            return "654321"

    captured = {}

    class FakeRegistrationEngine:
        def __init__(self, email_service=None, proxy_url=None, callback_logger=None, max_retries=3):
            captured["service_type"] = email_service.service_type.value
            captured["proxy_url"] = proxy_url
            captured["callback_logger"] = callback_logger
            captured["max_retries"] = max_retries
            self.email_service = email_service
            self.email = None
            self.password = None

        def run(self):
            captured["created_email"] = self.email_service.create_email()
            captured["verification_code"] = self.email_service.get_verification_code(
                timeout=45,
                otp_sent_at="otp-sent-at",
                exclude_codes={"used-code"},
            )
            captured["engine_email"] = self.email
            captured["engine_password"] = self.password
            return SimpleNamespace(
                success=True,
                email="fixed@example.com",
                password=self.password,
                account_id="chatgpt-account-id",
                access_token="access-token",
                refresh_token="refresh-token",
                id_token="id-token",
                session_token="session-token",
                workspace_id="workspace-id",
            )

    fake_mailbox = FakeMailbox()
    monkeypatch.setattr(registration_module, "RegistrationEngineV2", FakeRegistrationEngine)

    service = ChatGPTRegistrationService(
        config=RegisterConfig(
            proxy="http://proxy.example.com",
            extra={"register_max_retries": "5"},
        ),
        mailbox=fake_mailbox,
        log_fn=lambda msg: None,
    )

    account = service.register(email="fixed@example.com", password="secret")

    assert account.email == "fixed@example.com"
    assert account.password == "secret"
    assert account.user_id == "chatgpt-account-id"
    assert account.token == "access-token"
    assert account.extra == {
        "access_token": "access-token",
        "refresh_token": "refresh-token",
        "id_token": "id-token",
        "session_token": "session-token",
        "workspace_id": "workspace-id",
    }
    assert captured["service_type"] == "custom_provider"
    assert captured["proxy_url"] == "http://proxy.example.com"
    assert callable(captured["callback_logger"])
    assert captured["max_retries"] == 5
    assert captured["created_email"] == {
        "email": "fixed@example.com",
        "service_id": "mailbox-1",
        "token": "",
    }
    assert captured["verification_code"] == "654321"
    assert captured["engine_email"] == "fixed@example.com"
    assert captured["engine_password"] == "secret"
    assert fake_mailbox.get_email_calls == 1
    assert fake_mailbox.wait_calls == [
        {
            "email": "fixed@example.com",
            "account_id": "mailbox-1",
            "keyword": "",
            "timeout": 45,
            "otp_sent_at": "otp-sent-at",
            "exclude_codes": {"used-code"},
        }
    ]


def test_chatgpt_registration_service_falls_back_to_tempmail_without_mailbox(monkeypatch):
    from platforms.chatgpt.services.registration import ChatGPTRegistrationService
    import platforms.chatgpt.services.registration as registration_module

    created = {}
    engine_generated_password = "engine-generated-password"

    class FakeTempMailLolMailbox:
        def __init__(self, proxy=None):
            created["proxy"] = proxy
            self.wait_calls = []

        def get_email(self):
            created["get_email_called"] = True
            return MailboxAccount(email="generated@tempmail.test", account_id="tm-1")

        def wait_for_code(self, account, keyword="", timeout=120, otp_sent_at=None, exclude_codes=None, **kwargs):
            self.wait_calls.append(
                {
                    "email": account.email,
                    "account_id": account.account_id,
                    "keyword": keyword,
                    "timeout": timeout,
                    "otp_sent_at": otp_sent_at,
                    "exclude_codes": exclude_codes,
                }
            )
            created["wait_calls"] = list(self.wait_calls)
            return "112233"

    class FakeRegistrationEngine:
        def __init__(self, email_service=None, proxy_url=None, callback_logger=None, max_retries=3):
            created["service_type"] = email_service.service_type.value
            created["proxy_url"] = proxy_url
            created["max_retries"] = max_retries
            self.email_service = email_service
            self.email = None
            self.password = None

        def run(self):
            created["created_email"] = self.email_service.create_email()
            created["verification_code"] = self.email_service.get_verification_code(
                timeout=30,
                otp_sent_at="otp-sent-at",
                exclude_codes={"old-code"},
            )
            created["engine_email"] = self.email
            created["engine_password"] = self.password
            return SimpleNamespace(
                success=True,
                email="generated@tempmail.test",
                password=engine_generated_password,
                account_id="chatgpt-account-id",
                access_token="access-token",
                refresh_token="refresh-token",
                id_token="id-token",
                session_token="session-token",
                workspace_id="workspace-id",
            )

    monkeypatch.setattr(registration_module, "TempMailLolMailbox", FakeTempMailLolMailbox)
    monkeypatch.setattr(registration_module, "RegistrationEngineV2", FakeRegistrationEngine)

    service = ChatGPTRegistrationService(
        config=RegisterConfig(proxy="http://proxy.example.com"),
        mailbox=None,
        log_fn=lambda msg: None,
    )

    account = service.register(email=None, password="secret")

    assert account.email == "generated@tempmail.test"
    assert account.password == engine_generated_password
    assert account.token == "access-token"
    assert created["proxy"] == "http://proxy.example.com"
    assert created["service_type"] == "tempmail_lol"
    assert created["proxy_url"] == "http://proxy.example.com"
    assert created["max_retries"] == 3
    assert created["get_email_called"] is True
    assert created["created_email"] == {
        "email": "generated@tempmail.test",
        "service_id": "tm-1",
        "token": "tm-1",
    }
    assert created["verification_code"] == "112233"
    assert created["engine_email"] is None
    assert created["engine_password"] is None
    assert created["wait_calls"] == [
        {
            "email": "generated@tempmail.test",
            "account_id": "tm-1",
            "keyword": "",
            "timeout": 30,
            "otp_sent_at": "otp-sent-at",
            "exclude_codes": {"old-code"},
        }
    ]


def test_chatgpt_registration_service_generates_password_when_missing(monkeypatch):
    from platforms.chatgpt.services.registration import ChatGPTRegistrationService
    import platforms.chatgpt.services.registration as registration_module

    generated_password = "GeneratedPass123"
    captured = {}

    class FakeMailbox:
        def get_email(self):
            return MailboxAccount(email="generated@example.com", account_id="mailbox-1")

        def wait_for_code(self, account, keyword="", timeout=120, otp_sent_at=None, exclude_codes=None, **kwargs):
            return "654321"

    class FakeRegistrationEngine:
        def __init__(self, email_service=None, proxy_url=None, callback_logger=None, max_retries=3):
            self.email_service = email_service
            self.email = None
            self.password = None

        def run(self):
            captured["created_email"] = self.email_service.create_email()
            captured["engine_email"] = self.email
            captured["engine_password"] = self.password
            return SimpleNamespace(
                success=True,
                email="generated@example.com",
                password=self.password,
                account_id="chatgpt-account-id",
                access_token="access-token",
                refresh_token="refresh-token",
                id_token="id-token",
                session_token="session-token",
                workspace_id="workspace-id",
            )

    def fake_choices(population, k):
        captured["random_population"] = population
        captured["random_k"] = k
        return list(generated_password)

    monkeypatch.setattr(registration_module.random, "choices", fake_choices)
    monkeypatch.setattr(registration_module, "RegistrationEngineV2", FakeRegistrationEngine)

    service = ChatGPTRegistrationService(
        config=RegisterConfig(),
        mailbox=FakeMailbox(),
        log_fn=lambda msg: None,
    )

    account = service.register(email="generated@example.com", password=None)

    assert account.email == "generated@example.com"
    assert account.password == generated_password
    assert captured["created_email"] == {
        "email": "generated@example.com",
        "service_id": "mailbox-1",
        "token": "",
    }
    assert captured["engine_email"] == "generated@example.com"
    assert captured["engine_password"] == generated_password
    assert captured["random_k"] == 16
    assert captured["random_population"]


def test_chatgpt_token_service_check_valid_uses_subscription_status(monkeypatch):
    from platforms.chatgpt.services.token import ChatGPTTokenService
    import platforms.chatgpt.services.token as token_module

    observed_calls = []
    statuses = ["free", "plus", "team", "expired", "invalid", "banned", None]

    def fake_check_subscription_status(account, proxy=None):
        observed_calls.append(
            {
                "access_token": account.access_token,
                "cookies": account.cookies,
                "proxy": proxy,
            }
        )
        return statuses[len(observed_calls) - 1]

    monkeypatch.setattr(token_module, "check_subscription_status", fake_check_subscription_status)

    service = ChatGPTTokenService(RegisterConfig(proxy="http://proxy.example.com"))
    account = _make_chatgpt_account(
        extra={
            "access_token": "extra-access-token",
            "cookies": "oai-did=device-id; session=abc",
        }
    )

    results = [service.check_valid(account) for _ in statuses]

    assert results == [True, True, True, False, False, False, False]
    assert observed_calls == [
        {
            "access_token": "extra-access-token",
            "cookies": "oai-did=device-id; session=abc",
            "proxy": "http://proxy.example.com",
        }
        for _ in statuses
    ]



def test_chatgpt_token_service_refresh_token_wraps_success(monkeypatch):
    from platforms.chatgpt.services.token import ChatGPTTokenService
    import platforms.chatgpt.services.token as token_module

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
                access_token="new-access-token",
                refresh_token="new-refresh-token",
                error_message="",
            )

    monkeypatch.setattr(token_module, "TokenRefreshManager", FakeTokenRefreshManager)

    service = ChatGPTTokenService(RegisterConfig(proxy="http://proxy.example.com"))
    account = _make_chatgpt_account(
        extra={
            "access_token": "extra-access-token",
            "refresh_token": "old-refresh-token",
            "id_token": "id-token",
            "session_token": "session-token",
            "cookies": "oai-did=device-id; session=abc",
        }
    )

    result = service.refresh_token(account)

    assert result == {
        "ok": True,
        "data": {
            "access_token": "new-access-token",
            "refresh_token": "new-refresh-token",
        },
    }
    assert captured == {
        "proxy_url": "http://proxy.example.com",
        "email": "user@example.com",
        "access_token": "extra-access-token",
        "refresh_token": "old-refresh-token",
        "id_token": "id-token",
        "session_token": "session-token",
        "client_id": DEFAULT_CHATGPT_CLIENT_ID,
        "cookies": "oai-did=device-id; session=abc",
    }



def test_chatgpt_billing_service_routes_plus_and_team_links(monkeypatch):
    from platforms.chatgpt.services.billing import ChatGPTBillingService
    import platforms.chatgpt.services.billing as billing_module

    calls = []

    def fake_generate_plus_link(account, proxy=None, country="SG"):
        calls.append(
            {
                "plan": "plus",
                "access_token": account.access_token,
                "cookies": account.cookies,
                "proxy": proxy,
                "country": country,
            }
        )
        return "https://plus.example.com"

    def fake_generate_team_link(account, proxy=None, country="SG"):
        calls.append(
            {
                "plan": "team",
                "access_token": account.access_token,
                "cookies": account.cookies,
                "proxy": proxy,
                "country": country,
            }
        )
        return "https://team.example.com"

    monkeypatch.setattr(billing_module, "generate_plus_link", fake_generate_plus_link)
    monkeypatch.setattr(billing_module, "generate_team_link", fake_generate_team_link)

    service = ChatGPTBillingService(RegisterConfig(proxy="http://proxy.example.com"))
    account = _make_chatgpt_account(
        extra={
            "access_token": "extra-access-token",
            "cookies": "oai-did=device-id; session=abc",
        }
    )

    plus_result = service.payment_link(account, plan="plus", country="JP")
    team_result = service.payment_link(account, plan="team", country="US")

    assert plus_result == {"ok": True, "data": {"url": "https://plus.example.com"}}
    assert team_result == {"ok": True, "data": {"url": "https://team.example.com"}}
    assert calls == [
        {
            "plan": "plus",
            "access_token": "extra-access-token",
            "cookies": "oai-did=device-id; session=abc",
            "proxy": "http://proxy.example.com",
            "country": "JP",
        },
        {
            "plan": "team",
            "access_token": "extra-access-token",
            "cookies": "oai-did=device-id; session=abc",
            "proxy": "http://proxy.example.com",
            "country": "US",
        },
    ]



def test_chatgpt_external_sync_service_upload_cpa_wraps_success(monkeypatch):
    from platforms.chatgpt.services.external_sync import ChatGPTExternalSyncService
    import platforms.chatgpt.services.external_sync as sync_module

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
        return True, "CPA 上传成功"

    monkeypatch.setattr(sync_module, "generate_token_json", fake_generate_token_json)
    monkeypatch.setattr(sync_module, "upload_to_cpa", fake_upload_to_cpa)

    service = ChatGPTExternalSyncService()
    account = _make_chatgpt_account(
        extra={
            "access_token": "extra-access-token",
            "refresh_token": "refresh-token",
            "id_token": "id-token",
        }
    )

    result = service.upload_cpa(
        account,
        api_url="https://cpa.example.com",
        api_key="secret-key",
    )

    assert result == {"ok": True, "data": {"message": "CPA 上传成功"}}
    assert captured == {
        "email": "user@example.com",
        "access_token": "extra-access-token",
        "refresh_token": "refresh-token",
        "id_token": "id-token",
        "token_data": {
            "email": "user@example.com",
            "access_token": "extra-access-token",
        },
        "api_url": "https://cpa.example.com",
        "api_key": "secret-key",
    }



def test_chatgpt_external_sync_service_upload_tm_wraps_success(monkeypatch):
    from platforms.chatgpt.services.external_sync import ChatGPTExternalSyncService
    import platforms.chatgpt.services.external_sync as sync_module

    captured = {}

    def fake_upload_to_team_manager(account, api_url=None, api_key=None):
        captured["email"] = account.email
        captured["access_token"] = account.access_token
        captured["refresh_token"] = account.refresh_token
        captured["session_token"] = account.session_token
        captured["client_id"] = account.client_id
        captured["api_url"] = api_url
        captured["api_key"] = api_key
        return True, "TM 上传成功"

    monkeypatch.setattr(sync_module, "upload_to_team_manager", fake_upload_to_team_manager)

    service = ChatGPTExternalSyncService()
    account = _make_chatgpt_account(
        token="fallback-access-token",
        extra={
            "refresh_token": "refresh-token",
            "session_token": "session-token",
        },
    )

    result = service.upload_tm(
        account,
        api_url="https://tm.example.com",
        api_key="secret-key",
    )

    assert result == {"ok": True, "data": {"message": "TM 上传成功"}}
    assert captured == {
        "email": "user@example.com",
        "access_token": "fallback-access-token",
        "refresh_token": "refresh-token",
        "session_token": "session-token",
        "client_id": DEFAULT_CHATGPT_CLIENT_ID,
        "api_url": "https://tm.example.com",
        "api_key": "secret-key",
    }
