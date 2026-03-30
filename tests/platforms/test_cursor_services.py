from core.base_platform import Account, RegisterConfig


def test_cursor_registration_service_builds_otp_callback(monkeypatch):
    from platforms.cursor.services.registration import CursorRegistrationService
    import platforms.cursor.services.registration as registration_module

    class FakeMailboxAccount:
        email = "cursor@example.com"

    class FakeMailbox:
        def __init__(self):
            self.wait_calls = []

        def get_email(self):
            return FakeMailboxAccount()

        def get_current_ids(self, acct):
            return {"existing"}

        def wait_for_code(self, acct, keyword="", before_ids=None):
            self.wait_calls.append(
                {
                    "email": acct.email,
                    "keyword": keyword,
                    "before_ids": before_ids,
                }
            )
            return "654321"

    fake_mailbox = FakeMailbox()
    captured = {}

    class FakeCursorRegister:
        def __init__(self, proxy=None, log_fn=None):
            captured["proxy"] = proxy
            captured["log_fn"] = log_fn

        def register(self, email=None, password=None, otp_callback=None, yescaptcha_key=""):
            captured["email"] = email
            captured["password"] = password
            captured["yescaptcha_key"] = yescaptcha_key
            captured["otp"] = otp_callback()
            return {
                "email": email,
                "password": password or "generated-secret",
                "token": "cursor-token",
            }

    monkeypatch.setattr(registration_module, "CursorRegister", FakeCursorRegister)

    service = CursorRegistrationService(
        config=RegisterConfig(extra={"yescaptcha_key": "captcha-key"}),
        mailbox=fake_mailbox,
        log_fn=lambda msg: None,
    )

    account = service.register(email=None, password="secret")

    assert account.email == "cursor@example.com"
    assert account.password == "secret"
    assert account.token == "cursor-token"
    assert captured["otp"] == "654321"
    assert fake_mailbox.wait_calls == [
        {
            "email": "cursor@example.com",
            "keyword": "",
            "before_ids": {"existing"},
        }
    ]


def test_cursor_account_service_check_valid_uses_token(monkeypatch):
    from platforms.cursor.services.account import CursorAccountService

    calls = {}

    class FakeResponse:
        status_code = 200

    def fake_fetch(token: str):
        calls["token"] = token
        return FakeResponse()

    service = CursorAccountService(RegisterConfig())
    monkeypatch.setattr(service, "_fetch_auth_me", fake_fetch)

    account = Account(platform="cursor", email="user@example.com", password="secret", token="cursor-token")
    assert service.check_valid(account) is True
    assert calls["token"] == "cursor-token"


def test_cursor_account_service_get_user_info_wraps_failure():
    from platforms.cursor.services.account import CursorAccountService

    service = CursorAccountService(RegisterConfig())
    account = Account(platform="cursor", email="user@example.com", password="secret", token="")

    result = service.get_user_info(account)

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"] == "账号缺少 token"


def test_cursor_desktop_service_switch_account_wraps_success(monkeypatch):
    from platforms.cursor.services.desktop import CursorDesktopService
    import platforms.cursor.services.desktop as desktop_module

    monkeypatch.setattr(desktop_module, "switch_cursor_account", lambda token: (True, "切换成功"))
    monkeypatch.setattr(desktop_module, "restart_cursor_ide", lambda: (True, "Cursor IDE 已重启"))

    service = CursorDesktopService()
    account = Account(platform="cursor", email="user@example.com", password="secret", token="cursor-token")

    result = service.switch_account(account)

    assert result["ok"] is True
    assert result["data"]["message"] == "切换成功。Cursor IDE 已重启"


def test_cursor_desktop_service_switch_account_wraps_missing_token():
    from platforms.cursor.services.desktop import CursorDesktopService

    service = CursorDesktopService()
    account = Account(platform="cursor", email="user@example.com", password="secret", token="")

    result = service.switch_account(account)

    assert result["ok"] is False
    assert result["error"] == "账号缺少 token"
