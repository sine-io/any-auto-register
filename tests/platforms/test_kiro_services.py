from core.base_platform import Account, RegisterConfig


OTP_CODE_PATTERN = r'(?is)(?:verification\s+code|验证码)[^0-9]{0,20}(\d{6})'


def test_kiro_registration_service_builds_otp_callback(monkeypatch):
    from platforms.kiro.services.registration import KiroRegistrationService
    import platforms.kiro.services.registration as registration_module

    class FakeMailboxAccount:
        email = "kiro@example.com"

    class FakeMailbox:
        def __init__(self):
            self.wait_calls = []

        def get_email(self):
            return FakeMailboxAccount()

        def get_current_ids(self, acct):
            return {"existing"}

        def wait_for_code(self, acct, keyword="", timeout=0, before_ids=None, code_pattern=""):
            self.wait_calls.append(
                {
                    "email": acct.email,
                    "keyword": keyword,
                    "timeout": timeout,
                    "before_ids": before_ids,
                    "code_pattern": code_pattern,
                }
            )
            return "654321"

    fake_mailbox = FakeMailbox()
    captured = {}

    class FakeKiroRegister:
        def __init__(self, proxy=None, tag="KIRO", headless=False):
            captured["proxy"] = proxy
            captured["tag"] = tag
            captured["headless"] = headless

        def register(
            self,
            email=None,
            pwd=None,
            name="Kiro User",
            mail_token=None,
            otp_timeout=120,
            otp_callback=None,
        ):
            captured["email"] = email
            captured["pwd"] = pwd
            captured["name"] = name
            captured["mail_token"] = mail_token
            captured["otp_timeout"] = otp_timeout
            captured["otp"] = otp_callback()
            return True, {
                "email": email,
                "password": pwd or "generated-secret",
                "name": "Kiro User",
                "accessToken": "access-token",
                "sessionToken": "session-token",
                "clientId": "client-id",
                "clientSecret": "client-secret",
                "clientIdHash": "client-id-hash",
                "refreshToken": "refresh-token",
                "webAccessToken": "web-access-token",
                "region": "us-east-1",
            }

    monkeypatch.setattr(registration_module, "KiroRegister", FakeKiroRegister)

    service = KiroRegistrationService(
        config=RegisterConfig(
            proxy="http://proxy.example.com",
            extra={
                "laoudo_account_id": "mail-token",
                "name": "Kiro User",
                "otp_timeout": 45,
            },
        ),
        mailbox=fake_mailbox,
        log_fn=lambda msg: None,
    )

    account = service.register(email=None, password="secret")

    assert account.email == "kiro@example.com"
    assert account.password == "secret"
    assert account.extra == {
        "name": "Kiro User",
        "accessToken": "access-token",
        "sessionToken": "session-token",
        "clientId": "client-id",
        "clientSecret": "client-secret",
        "clientIdHash": "client-id-hash",
        "refreshToken": "refresh-token",
        "webAccessToken": "web-access-token",
        "region": "us-east-1",
        "provider": "BuilderId",
        "authMethod": "IdC",
    }
    assert captured["proxy"] == "http://proxy.example.com"
    assert captured["tag"] == "KIRO"
    assert captured["email"] == "kiro@example.com"
    assert captured["pwd"] == "secret"
    assert captured["mail_token"] == "mail-token"
    assert captured["otp_timeout"] == 45
    assert captured["otp"] == "654321"
    assert fake_mailbox.wait_calls == [
        {
            "email": "kiro@example.com",
            "keyword": "builder id",
            "timeout": 45,
            "before_ids": {"existing"},
            "code_pattern": OTP_CODE_PATTERN,
        }
    ]


def test_kiro_token_service_check_valid_uses_refresh_credentials(monkeypatch):
    from platforms.kiro.services.token import KiroTokenService
    import platforms.kiro.services.token as token_module

    calls = {}

    def fake_refresh_kiro_token(refresh_token, client_id, client_secret):
        calls["refresh_token"] = refresh_token
        calls["client_id"] = client_id
        calls["client_secret"] = client_secret
        return True, {"accessToken": "fresh-access-token"}

    monkeypatch.setattr(token_module, "refresh_kiro_token", fake_refresh_kiro_token)

    service = KiroTokenService(config=RegisterConfig())
    account = Account(
        platform="kiro",
        email="user@example.com",
        password="secret",
        extra={
            "refreshToken": "refresh-token",
            "clientId": "client-id",
            "clientSecret": "client-secret",
        },
    )

    assert service.check_valid(account) is True
    assert calls == {
        "refresh_token": "refresh-token",
        "client_id": "client-id",
        "client_secret": "client-secret",
    }


def test_kiro_token_service_refresh_token_wraps_success(monkeypatch):
    from platforms.kiro.services.token import KiroTokenService
    import platforms.kiro.services.token as token_module

    monkeypatch.setattr(
        token_module,
        "refresh_kiro_token",
        lambda refresh_token, client_id, client_secret: (
            True,
            {
                "accessToken": "fresh-access-token",
                "refreshToken": "fresh-refresh-token",
            },
        ),
    )

    service = KiroTokenService(config=RegisterConfig())
    account = Account(
        platform="kiro",
        email="user@example.com",
        password="secret",
        extra={
            "refreshToken": "refresh-token",
            "clientId": "client-id",
            "clientSecret": "client-secret",
        },
    )

    result = service.refresh_token(account)

    assert result == {
        "ok": True,
        "data": {
            "access_token": "fresh-access-token",
            "accessToken": "fresh-access-token",
            "refreshToken": "fresh-refresh-token",
        },
    }


def test_kiro_token_service_ensure_desktop_tokens_wraps_missing_credentials():
    from platforms.kiro.services.token import KiroTokenService

    service = KiroTokenService(config=RegisterConfig())
    account = Account(
        platform="kiro",
        email="",
        password="",
        extra={"accessToken": "web-access-token"},
    )

    result = service.ensure_desktop_tokens(account)

    assert result["ok"] is False
    assert result["error"] == (
        "当前账号只有网页登录态，缺少 refreshToken / clientId / clientSecret，"
        "并且没有可用的邮箱/密码用于自动补抓桌面端 Token。"
    )


def test_kiro_token_service_ensure_desktop_tokens_bootstraps_with_mailbox_otp(monkeypatch):
    from platforms.kiro.services.token import KiroTokenService
    import platforms.kiro.services.token as token_module

    captured = {"refresh_calls": 0}

    class FakeMailbox:
        def __init__(self):
            self.wait_calls = []

        def get_current_ids(self, acct):
            captured["mail_account"] = acct
            return {"existing"}

        def wait_for_code(self, acct, keyword="", timeout=0, before_ids=None, code_pattern=""):
            self.wait_calls.append(
                {
                    "email": acct.email,
                    "keyword": keyword,
                    "timeout": timeout,
                    "before_ids": before_ids,
                    "code_pattern": code_pattern,
                }
            )
            return "654321"

    fake_mailbox = FakeMailbox()

    def fake_create_mailbox(provider="", extra=None, proxy=None):
        captured["provider"] = provider
        captured["extra"] = extra
        captured["proxy"] = proxy
        return fake_mailbox

    class FakeMailboxAccount:
        def __init__(self, email, account_id=""):
            self.email = email
            self.account_id = account_id

    class FakeKiroRegister:
        def __init__(self, proxy=None, tag="KIRO-SWITCH", headless=False):
            captured["register_proxy"] = proxy
            captured["register_tag"] = tag
            captured["register_headless"] = headless
            self.log = lambda msg: None

        def fetch_desktop_tokens(self, email, pwd, otp_callback=None):
            captured["email"] = email
            captured["password"] = pwd
            captured["otp"] = otp_callback()
            return True, {
                "accessToken": "",
                "refreshToken": "desktop-refresh-token",
                "clientId": "desktop-client-id",
                "clientSecret": "desktop-client-secret",
            }

    monkeypatch.setattr(token_module, "create_mailbox", fake_create_mailbox)
    monkeypatch.setattr(token_module, "MailboxAccount", FakeMailboxAccount)
    monkeypatch.setattr(token_module, "KiroRegister", FakeKiroRegister)

    def fake_refresh_kiro_token(refresh_token, client_id, client_secret):
        captured["refresh_calls"] += 1
        return True, {
            "accessToken": "unexpected-refreshed-access-token",
            "refreshToken": "unexpected-refreshed-refresh-token",
        }

    monkeypatch.setattr(token_module, "refresh_kiro_token", fake_refresh_kiro_token)

    service = KiroTokenService(
        config=RegisterConfig(
            proxy="http://proxy.example.com",
            extra={"mail_provider": "duckmail"},
        ),
        log_fn=lambda msg: None,
    )
    account = Account(
        platform="kiro",
        email="user@example.com",
        password="secret",
        extra={"accessToken": "web-access-token"},
    )

    result = service.ensure_desktop_tokens(account)

    assert result == {
        "ok": True,
        "data": {
            "accessToken": "web-access-token",
            "refreshToken": "desktop-refresh-token",
            "clientId": "desktop-client-id",
            "clientSecret": "desktop-client-secret",
        },
    }
    assert captured["provider"] == "duckmail"
    assert captured["extra"] == {"mail_provider": "duckmail"}
    assert captured["proxy"] == "http://proxy.example.com"
    assert captured["register_proxy"] == "http://proxy.example.com"
    assert captured["register_tag"] == "KIRO-SWITCH"
    assert captured["email"] == "user@example.com"
    assert captured["password"] == "secret"
    assert captured["otp"] == "654321"
    assert captured["refresh_calls"] == 0
    assert fake_mailbox.wait_calls == [
        {
            "email": "user@example.com",
            "keyword": "",
            "timeout": 45,
            "before_ids": {"existing"},
            "code_pattern": OTP_CODE_PATTERN,
        }
    ]


def test_kiro_desktop_service_switch_account_wraps_restart_result(monkeypatch):
    from platforms.kiro.services.desktop import KiroDesktopService
    import platforms.kiro.services.desktop as desktop_module

    captured = {"calls": []}

    class FakeKiroTokenService:
        def __init__(self, config=None, log_fn=None):
            captured["config"] = config
            captured["log_fn"] = log_fn

        def ensure_desktop_tokens(self, account):
            captured["calls"].append("ensure")
            captured["ensured_account"] = account
            return {
                "ok": True,
                "data": {
                    "accessToken": "desktop-access-token",
                    "refreshToken": "desktop-refresh-token",
                    "clientId": "desktop-client-id",
                    "clientSecret": "desktop-client-secret",
                },
            }

        def refresh_token(self, account):
            captured["calls"].append("refresh")
            captured["refresh_account"] = account
            return {
                "ok": True,
                "data": {
                    "access_token": "fresh-access-token",
                    "accessToken": "fresh-access-token",
                    "refreshToken": "fresh-refresh-token",
                },
            }

    def fake_switch_kiro_account(access_token, refresh_token, client_id, client_secret):
        captured["calls"].append("switch")
        captured["switch_args"] = {
            "access_token": access_token,
            "refresh_token": refresh_token,
            "client_id": client_id,
            "client_secret": client_secret,
        }
        return True, "切换成功，Kiro IDE 将自动使用新账号"

    monkeypatch.setattr(desktop_module, "KiroTokenService", FakeKiroTokenService)
    monkeypatch.setattr(desktop_module, "switch_kiro_account", fake_switch_kiro_account)

    def fake_restart_kiro_ide():
        captured["calls"].append("restart")
        return True, "Kiro IDE 已重启"

    monkeypatch.setattr(desktop_module, "restart_kiro_ide", fake_restart_kiro_ide)

    service = KiroDesktopService(config=RegisterConfig(), log_fn=lambda msg: None)
    account = Account(
        platform="kiro",
        email="user@example.com",
        password="secret",
        extra={"accessToken": "web-access-token"},
    )

    result = service.switch_account(account)

    assert captured["ensured_account"] is account
    assert captured["refresh_account"].extra == {
        "accessToken": "desktop-access-token",
        "refreshToken": "desktop-refresh-token",
        "clientId": "desktop-client-id",
        "clientSecret": "desktop-client-secret",
    }
    assert captured["calls"] == ["ensure", "refresh", "switch", "restart"]
    assert captured["switch_args"] == {
        "access_token": "fresh-access-token",
        "refresh_token": "fresh-refresh-token",
        "client_id": "desktop-client-id",
        "client_secret": "desktop-client-secret",
    }
    assert result == {
        "ok": True,
        "data": {
            "accessToken": "fresh-access-token",
            "refreshToken": "fresh-refresh-token",
            "clientId": "desktop-client-id",
            "clientSecret": "desktop-client-secret",
            "message": "切换成功，Kiro IDE 将自动使用新账号。Kiro IDE 已重启",
        },
    }


def test_kiro_manager_sync_service_upload_wraps_success(monkeypatch):
    from platforms.kiro.services.manager_sync import KiroManagerSyncService
    import platforms.kiro.services.manager_sync as manager_sync_module

    captured = {}

    def fake_upload_to_kiro_manager(account):
        captured["account"] = account
        return True, "导入成功: /tmp/accounts.json"

    monkeypatch.setattr(manager_sync_module, "upload_to_kiro_manager", fake_upload_to_kiro_manager)

    service = KiroManagerSyncService()
    account = Account(
        platform="kiro",
        email="user@example.com",
        password="secret",
        extra={
            "accessToken": "access-token",
            "refreshToken": "refresh-token",
            "clientId": "client-id",
            "clientSecret": "client-secret",
        },
    )

    result = service.upload(account)

    assert captured["account"] is account
    assert result == {
        "ok": True,
        "data": {"message": "导入成功: /tmp/accounts.json"},
    }
