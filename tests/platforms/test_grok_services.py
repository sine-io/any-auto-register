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


def test_grok_registration_service_uses_global_yescaptcha_fallback_and_builds_solver(monkeypatch):
    from platforms.grok.services.registration import GrokRegistrationService
    import core.config_store as config_store_module
    import platforms.grok.services.registration as registration_module

    class FakeConfigStore:
        def __init__(self):
            self.calls = []

        def get(self, key, default=""):
            self.calls.append((key, default))
            return "global-captcha-key"

    fake_config_store = FakeConfigStore()
    monkeypatch.setattr(config_store_module, "config_store", fake_config_store)
    monkeypatch.setattr(registration_module, "config_store", fake_config_store, raising=False)

    def run_case(*, extra, expected_key, email):
        captured = {}

        class TestableGrokRegistrationService(GrokRegistrationService):
            def _make_captcha(self, **kwargs):
                captured["captcha_kwargs"] = kwargs
                captured["built_solver"] = object()
                return captured["built_solver"]

        class FakeGrokRegister:
            def __init__(self, captcha_solver=None, yescaptcha_key="", proxy=None, log_fn=None):
                captured["captcha_solver"] = captcha_solver
                captured["yescaptcha_key"] = yescaptcha_key
                captured["proxy"] = proxy
                captured["log_fn"] = log_fn

            def register(self, email=None, password=None, otp_callback=None):
                captured["email"] = email
                captured["password"] = password
                captured["otp_callback"] = otp_callback
                return {
                    "email": email,
                    "password": password or "generated-secret",
                    "sso": "sso-token",
                    "sso_rw": "sso-rw-token",
                    "given_name": "Grok",
                    "family_name": "User",
                }

        monkeypatch.setattr(registration_module, "GrokRegister", FakeGrokRegister)

        service = TestableGrokRegistrationService(
            config=RegisterConfig(proxy="http://proxy.example.com", extra=extra),
            mailbox=None,
            log_fn=lambda msg: None,
        )

        account = service.register(email=email, password="secret")

        assert account.email == email
        assert captured["captcha_kwargs"] == {"key": expected_key}
        assert captured["captcha_solver"] is captured["built_solver"]
        assert captured["yescaptcha_key"] == expected_key
        assert captured["proxy"] == "http://proxy.example.com"
        assert captured["otp_callback"] is None

    run_case(
        extra={"yescaptcha_key": "task-captcha-key"},
        expected_key="task-captcha-key",
        email="task@example.com",
    )
    run_case(
        extra={},
        expected_key="global-captcha-key",
        email="fallback@example.com",
    )
    assert fake_config_store.calls == [("yescaptcha_key", "")]


def test_grok_registration_service_surfaces_global_yescaptcha_fallback_errors(monkeypatch):
    from platforms.grok.services.registration import GrokRegistrationService
    import platforms.grok.services.registration as registration_module

    class ExplodingConfigStore:
        def get(self, key, default=""):
            raise RuntimeError("config store unavailable")

    class UnexpectedGrokRegister:
        def __init__(self, *args, **kwargs):
            raise AssertionError("register flow should not proceed when config fallback fails")

    monkeypatch.setattr(registration_module, "config_store", ExplodingConfigStore(), raising=False)
    monkeypatch.setattr(registration_module, "GrokRegister", UnexpectedGrokRegister)

    service = GrokRegistrationService(
        config=RegisterConfig(extra={}),
        mailbox=None,
        log_fn=lambda msg: None,
    )

    with pytest.raises(RuntimeError, match="config store unavailable"):
        service.register(email="fallback-error@example.com", password="secret")


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
        config=RegisterConfig(extra={"grok_mailbox_attempts": 3, "yescaptcha_key": "captcha-key"}),
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
        config=RegisterConfig(extra={"grok_mailbox_attempts": 5, "yescaptcha_key": "captcha-key"}),
        mailbox=fixed_mailbox,
        log_fn=lambda msg: None,
    )

    with pytest.raises(RuntimeError, match="邮箱域名被拒绝"):
        fixed_service.register(email="fixed@example.com", password="secret")

    assert fixed_email_attempts == ["fixed@example.com"]
    assert fixed_mailbox.get_email_calls == 0


def test_grok_registration_service_passes_otp_callback_with_fixed_email_when_mailbox_exists(monkeypatch):
    from platforms.grok.services.registration import GrokRegistrationService
    import platforms.grok.services.registration as registration_module

    captured = {}

    class FakeMailbox:
        def get_email(self):
            raise AssertionError("fixed email path should not rotate mailbox")

        def get_current_ids(self, acct):
            raise AssertionError("fixed email path should not load mailbox ids")

        def wait_for_code(self, acct, keyword="", before_ids=None, code_pattern=""):
            captured["wait_call"] = {
                "acct": acct,
                "keyword": keyword,
                "before_ids": before_ids,
                "code_pattern": code_pattern,
            }
            raise RuntimeError("no mailbox account available")

    class TestableGrokRegistrationService(GrokRegistrationService):
        def _make_captcha(self, **kwargs):
            return "captcha-solver"

    class FakeGrokRegister:
        def __init__(self, captcha_solver=None, yescaptcha_key="", proxy=None, log_fn=None):
            captured["captcha_solver"] = captcha_solver

        def register(self, email=None, password=None, otp_callback=None):
            captured["email"] = email
            captured["password"] = password
            captured["otp_callback"] = otp_callback
            return {
                "email": email,
                "password": password or "generated-secret",
                "sso": "sso-token",
                "sso_rw": "sso-rw-token",
                "given_name": "Fixed",
                "family_name": "Email",
            }

    monkeypatch.setattr(registration_module, "GrokRegister", FakeGrokRegister)

    service = TestableGrokRegistrationService(
        config=RegisterConfig(extra={"yescaptcha_key": "captcha-key"}),
        mailbox=FakeMailbox(),
        log_fn=lambda msg: None,
    )

    account = service.register(email="fixed@example.com", password="secret")

    assert account.email == "fixed@example.com"
    assert captured["otp_callback"] is not None
    with pytest.raises(RuntimeError, match="no mailbox account available"):
        captured["otp_callback"]()
    assert captured["wait_call"] == {
        "acct": None,
        "keyword": "",
        "before_ids": set(),
        "code_pattern": OTP_CODE_PATTERN,
    }


def test_grok_registration_service_passes_none_otp_callback_without_mailbox(monkeypatch):
    from platforms.grok.services.registration import GrokRegistrationService
    import platforms.grok.services.registration as registration_module

    captured = {}

    class TestableGrokRegistrationService(GrokRegistrationService):
        def _make_captcha(self, **kwargs):
            captured["captcha_kwargs"] = kwargs
            return "captcha-solver"

    class FakeGrokRegister:
        def __init__(self, captcha_solver=None, yescaptcha_key="", proxy=None, log_fn=None):
            captured["captcha_solver"] = captcha_solver
            captured["yescaptcha_key"] = yescaptcha_key

        def register(self, email=None, password=None, otp_callback=None):
            captured["email"] = email
            captured["password"] = password
            captured["otp_callback"] = otp_callback
            return {
                "email": email,
                "password": password or "generated-secret",
                "sso": "sso-token",
                "sso_rw": "sso-rw-token",
                "given_name": "Manual",
                "family_name": "OTP",
            }

    monkeypatch.setattr(registration_module, "GrokRegister", FakeGrokRegister)

    service = TestableGrokRegistrationService(
        config=RegisterConfig(extra={"yescaptcha_key": "captcha-key"}),
        mailbox=None,
        log_fn=lambda msg: None,
    )

    account = service.register(email="manual@example.com", password="secret")

    assert account.email == "manual@example.com"
    assert captured["captcha_kwargs"] == {"key": "captcha-key"}
    assert captured["captcha_solver"] == "captcha-solver"
    assert captured["yescaptcha_key"] == "captcha-key"
    assert captured["email"] == "manual@example.com"
    assert captured["password"] == "secret"
    assert captured["otp_callback"] is None


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


def test_grok_platform_register_delegates_to_registration_service(monkeypatch):
    from platforms.grok.plugin import GrokPlatform
    import platforms.grok.core as core_module
    import platforms.grok.plugin as plugin_module

    mailbox = object()
    captured = {}
    expected = Account(
        platform="grok",
        email="delegated@example.com",
        password="secret",
        extra={"sso": "token"},
    )

    class FakeRegistrationService:
        def __init__(self, config=None, mailbox=None, log_fn=None):
            captured["init"] = {
                "config": config,
                "mailbox": mailbox,
                "log_fn": log_fn,
            }

        def register(self, email=None, password=None):
            captured["call"] = {
                "email": email,
                "password": password,
            }
            return expected

    class LegacyGrokRegister:
        def __init__(self, *args, **kwargs):
            captured["legacy_path_used"] = True

        def register(self, email=None, password=None, otp_callback=None):
            return {
                "email": "legacy@example.com",
                "password": password or "generated-secret",
                "sso": "legacy-sso",
                "sso_rw": "legacy-sso-rw",
                "given_name": "Legacy",
                "family_name": "Path",
            }

    monkeypatch.setattr(plugin_module, "GrokRegistrationService", FakeRegistrationService, raising=False)
    monkeypatch.setattr(core_module, "GrokRegister", LegacyGrokRegister)
    monkeypatch.setattr(GrokPlatform, "_make_captcha", lambda self, **kwargs: "captcha-solver")

    instance = GrokPlatform(RegisterConfig(extra={"yescaptcha_key": "captcha-key"}), mailbox=mailbox)
    log_fn = lambda msg: None
    instance._log_fn = log_fn

    result = instance.register("delegated@example.com", "secret")

    assert result is expected
    assert captured["init"] == {
        "config": instance.config,
        "mailbox": mailbox,
        "log_fn": log_fn,
    }
    assert captured["call"] == {
        "email": "delegated@example.com",
        "password": "secret",
    }
    assert "legacy_path_used" not in captured


def test_grok_platform_register_preserves_retry_logging_via_registration_service(monkeypatch):
    from platforms.grok.plugin import GrokPlatform
    import platforms.grok.services.registration as registration_module

    logged = []

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

    class RetryOnceGrokRegister:
        def __init__(self, captcha_solver=None, yescaptcha_key="", proxy=None, log_fn=None):
            pass

        def register(self, email=None, password=None, otp_callback=None):
            if email == "blocked@example.com":
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
    monkeypatch.setattr(
        registration_module.GrokRegistrationService,
        "_make_captcha",
        lambda self, **kwargs: "captcha-solver",
    )

    mailbox = FakeMailbox(["blocked@example.com", "fresh@example.com"])
    instance = GrokPlatform(
        RegisterConfig(extra={"grok_mailbox_attempts": 3, "yescaptcha_key": "captcha-key"}),
        mailbox=mailbox,
    )
    instance._log_fn = logged.append

    account = instance.register(None, "secret")

    assert account.email == "fresh@example.com"
    assert logged == [
        "邮箱: blocked@example.com",
        "Grok 邮箱域名被拒绝，切换新邮箱重试 2/3",
        "邮箱: fresh@example.com",
        "等待验证码...",
        "验证码: ZXQ987",
    ]
    assert mailbox.get_email_calls == 2
    assert mailbox.current_id_calls == ["blocked@example.com", "fresh@example.com"]
    assert mailbox.wait_calls == [
        {
            "email": "fresh@example.com",
            "keyword": "",
            "before_ids": {"before:fresh@example.com"},
            "code_pattern": OTP_CODE_PATTERN,
        }
    ]


def test_grok_platform_check_valid_delegates_to_cookie_service(monkeypatch):
    from platforms.grok.plugin import GrokPlatform
    import platforms.grok.plugin as plugin_module

    account = Account(platform="grok", email="user@example.com", password="secret", extra={"sso": "token"})
    captured = {}

    class FakeCookieService:
        def __init__(self, config=None):
            captured["config"] = config

        def check_valid(self, delegated_account):
            captured["account"] = delegated_account
            return True

    monkeypatch.setattr(plugin_module, "GrokCookieService", FakeCookieService, raising=False)

    instance = GrokPlatform(RegisterConfig())

    assert instance.check_valid(account) is True
    assert captured == {
        "config": instance.config,
        "account": account,
    }


def test_grok_platform_execute_action_delegates_to_sync_service(monkeypatch):
    from platforms.grok.plugin import GrokPlatform
    import platforms.grok.grok2api_upload as upload_module
    import platforms.grok.plugin as plugin_module

    account = Account(platform="grok", email="user@example.com", password="secret", extra={"sso": "token"})
    captured = {}

    class FakeSyncService:
        def upload_grok2api(self, delegated_account):
            captured["account"] = delegated_account
            return {"ok": True, "data": {"message": "delegated upload"}}

    monkeypatch.setattr(plugin_module, "GrokSyncService", FakeSyncService, raising=False)
    monkeypatch.setattr(upload_module, "upload_to_grok2api", lambda account: (False, "legacy direct upload path"))

    instance = GrokPlatform(RegisterConfig())

    result = instance.execute_action("upload_grok2api", account, {})

    assert result == {"ok": True, "data": {"message": "delegated upload"}}
    assert captured == {"account": account}
