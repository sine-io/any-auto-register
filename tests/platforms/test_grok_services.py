from core.base_platform import Account, RegisterConfig
import pytest


OTP_CODE_PATTERN = r"[A-Z0-9]{3}-[A-Z0-9]{3}"


def test_grok_registration_service_builds_otp_callback(monkeypatch):
    from platforms.grok.services.registration import GrokRegistrationService
    import platforms.grok.services.registration as registration_module

    class FakeMailboxAccount:
        email = "grok@example.com"

    class FakeMailbox:
        def __init__(self):
            self.wait_calls = []

        def get_email(self):
            return FakeMailboxAccount()

        def get_current_ids(self, acct):
            return {"existing"}

        def wait_for_code(self, acct, keyword="", before_ids=None, code_pattern=""):
            self.wait_calls.append(
                {
                    "email": acct.email,
                    "keyword": keyword,
                    "before_ids": before_ids,
                    "code_pattern": code_pattern,
                }
            )
            return "ABC-123"

    fake_mailbox = FakeMailbox()
    captured = {}

    class FakeGrokRegister:
        def __init__(self, captcha_solver=None, yescaptcha_key="", proxy=None, log_fn=None):
            captured["captcha_solver"] = captcha_solver
            captured["yescaptcha_key"] = yescaptcha_key
            captured["proxy"] = proxy
            captured["log_fn"] = log_fn

        def register(self, email=None, password=None, otp_callback=None):
            captured["email"] = email
            captured["password"] = password
            captured["otp"] = otp_callback()
            return {
                "email": email,
                "password": password or "generated-secret",
                "sso": "sso-token",
                "sso_rw": "sso-rw-token",
                "given_name": "Grok",
                "family_name": "User",
            }

    monkeypatch.setattr(registration_module, "GrokRegister", FakeGrokRegister)

    service = GrokRegistrationService(
        config=RegisterConfig(extra={"yescaptcha_key": "captcha-key"}),
        mailbox=fake_mailbox,
        log_fn=lambda msg: None,
    )

    account = service.register(email=None, password="secret")

    assert account.email == "grok@example.com"
    assert account.password == "secret"
    assert account.extra == {
        "sso": "sso-token",
        "sso_rw": "sso-rw-token",
        "given_name": "Grok",
        "family_name": "User",
    }
    assert captured["yescaptcha_key"] == "captcha-key"
    assert captured["otp"] == "ABC123"
    assert fake_mailbox.wait_calls == [
        {
            "email": "grok@example.com",
            "keyword": "",
            "before_ids": {"existing"},
            "code_pattern": OTP_CODE_PATTERN,
        }
    ]


def test_grok_registration_service_retries_rejected_mailbox_domain(monkeypatch):
    from platforms.grok.services.registration import GrokRegistrationService
    import platforms.grok.services.registration as registration_module

    class FakeMailboxAccount:
        def __init__(self, email):
            self.email = email

    class FakeMailbox:
        def __init__(self, emails):
            self._accounts = [FakeMailboxAccount(email) for email in emails]
            self.get_email_calls = 0
            self.current_id_calls = []
            self.wait_calls = []

        def get_email(self):
            acct = self._accounts[self.get_email_calls]
            self.get_email_calls += 1
            return acct

        def get_current_ids(self, acct):
            self.current_id_calls.append(acct.email)
            return {f"before:{acct.email}"}

        def wait_for_code(self, acct, keyword="", before_ids=None, code_pattern=""):
            self.wait_calls.append(
                {
                    "email": acct.email,
                    "keyword": keyword,
                    "before_ids": before_ids,
                    "code_pattern": code_pattern,
                }
            )
            return "ZXQ-987"

    auto_mailbox = FakeMailbox(["blocked@example.com", "fresh@example.com"])
    register_attempts = []

    class RetryOnceGrokRegister:
        def __init__(self, captcha_solver=None, yescaptcha_key="", proxy=None, log_fn=None):
            pass

        def register(self, email=None, password=None, otp_callback=None):
            register_attempts.append(email)
            if len(register_attempts) == 1:
                raise RuntimeError("邮箱域名被拒绝: disposable domain")
            return {
                "email": email,
                "password": password or "generated-secret",
                "sso": f"otp:{otp_callback()}",
                "sso_rw": "sso-rw-token",
                "given_name": "Fresh",
                "family_name": "Mailbox",
            }

    monkeypatch.setattr(registration_module, "GrokRegister", RetryOnceGrokRegister)

    auto_service = GrokRegistrationService(
        config=RegisterConfig(extra={"grok_mailbox_attempts": 3}),
        mailbox=auto_mailbox,
        log_fn=lambda msg: None,
    )

    account = auto_service.register(email=None, password="secret")

    assert account.email == "fresh@example.com"
    assert register_attempts == ["blocked@example.com", "fresh@example.com"]
    assert auto_mailbox.get_email_calls == 2
    assert auto_mailbox.current_id_calls == ["blocked@example.com", "fresh@example.com"]
    assert auto_mailbox.wait_calls == [
        {
            "email": "fresh@example.com",
            "keyword": "",
            "before_ids": {"before:fresh@example.com"},
            "code_pattern": OTP_CODE_PATTERN,
        }
    ]

    fixed_mailbox = FakeMailbox(["unused@example.com"])
    fixed_email_attempts = []

    class RejectFixedEmailGrokRegister:
        def __init__(self, captcha_solver=None, yescaptcha_key="", proxy=None, log_fn=None):
            pass

        def register(self, email=None, password=None, otp_callback=None):
            fixed_email_attempts.append(email)
            raise RuntimeError("邮箱域名被拒绝: fixed email should not rotate mailbox")

    monkeypatch.setattr(registration_module, "GrokRegister", RejectFixedEmailGrokRegister)

    fixed_service = GrokRegistrationService(
        config=RegisterConfig(extra={"grok_mailbox_attempts": 5}),
        mailbox=fixed_mailbox,
        log_fn=lambda msg: None,
    )

    with pytest.raises(RuntimeError, match="邮箱域名被拒绝"):
        fixed_service.register(email="fixed@example.com", password="secret")

    assert fixed_email_attempts == ["fixed@example.com"]
    assert fixed_mailbox.get_email_calls == 0


def test_grok_cookie_service_check_valid_uses_sso():
    from platforms.grok.services.cookie import GrokCookieService

    service = GrokCookieService(RegisterConfig())

    assert service.check_valid(Account(platform="grok", email="user@example.com", password="secret", extra={"sso": "token"})) is True
    assert service.check_valid(Account(platform="grok", email="user@example.com", password="secret", extra={"sso_rw": "rw-only"})) is False
    assert service.check_valid(Account(platform="grok", email="user@example.com", password="secret", extra={})) is False


def test_grok_sync_service_upload_wraps_success(monkeypatch):
    from platforms.grok.services.sync import GrokSyncService
    import platforms.grok.services.sync as sync_module

    captured = {}

    def fake_upload_to_grok2api(account):
        captured["account"] = account
        return True, "导入成功"

    monkeypatch.setattr(sync_module, "upload_to_grok2api", fake_upload_to_grok2api)

    service = GrokSyncService()
    account = Account(platform="grok", email="user@example.com", password="secret", extra={"sso": "token"})

    result = service.upload_grok2api(account)

    assert captured["account"] is account
    assert result == {
        "ok": True,
        "data": {"message": "导入成功"},
    }
