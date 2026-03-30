from core.base_platform import Account, RegisterConfig


def test_trae_registration_service_builds_otp_callback(monkeypatch):
    from platforms.trae.services.registration import TraeRegistrationService
    import platforms.trae.services.registration as registration_module

    class FakeMailboxAccount:
        email = "trae@example.com"

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

    class FakeTraeRegister:
        def __init__(self, executor=None, log_fn=None):
            captured["executor"] = executor
            captured["log_fn"] = log_fn

        def register(self, email=None, password=None, otp_callback=None):
            captured["email"] = email
            captured["password"] = password
            captured["otp"] = otp_callback()
            return {
                "email": email,
                "password": password or "generated-secret",
                "user_id": "trae-user-id",
                "token": "trae-token",
                "region": "sg",
                "cashier_url": "https://cashier.example.com",
                "ai_pay_host": "https://pay.example.com",
            }

    monkeypatch.setattr(registration_module, "TraeRegister", FakeTraeRegister)

    service = TraeRegistrationService(
        config=RegisterConfig(),
        mailbox=fake_mailbox,
        log_fn=lambda msg: None,
    )

    account = service.register(email=None, password="secret")

    assert account.email == "trae@example.com"
    assert account.password == "secret"
    assert account.user_id == "trae-user-id"
    assert account.token == "trae-token"
    assert account.region == "sg"
    assert account.extra == {
        "cashier_url": "https://cashier.example.com",
        "ai_pay_host": "https://pay.example.com",
    }
    assert captured["otp"] == "654321"
    assert fake_mailbox.wait_calls == [
        {
            "email": "trae@example.com",
            "keyword": "",
            "before_ids": {"existing"},
        }
    ]


def test_trae_platform_register_logs_email_and_delegates_to_registration_service(monkeypatch):
    from platforms.trae.plugin import TraePlatform

    mailbox = object()
    logged = []
    captured = {}
    expected = Account(
        platform="trae",
        email="delegated@example.com",
        password="secret",
        user_id="user-id",
        token="token",
        region="sg",
    )

    class FakeRegistrationService:
        def register(self, email=None, password=None):
            captured["email"] = email
            captured["password"] = password
            return expected

    monkeypatch.setattr(TraePlatform, "_registration_service", lambda self: FakeRegistrationService())

    instance = TraePlatform(RegisterConfig(), mailbox=mailbox)
    instance._log_fn = logged.append

    result = instance.register("delegated@example.com", "secret")

    assert result is expected
    assert captured == {"email": "delegated@example.com", "password": "secret"}
    assert logged == ["邮箱: delegated@example.com"]


def test_trae_platform_register_logs_mailbox_email_when_input_missing(monkeypatch):
    from platforms.trae.plugin import TraePlatform

    logged = []
    captured = {}
    expected = Account(
        platform="trae",
        email="mailbox@example.com",
        password="secret",
    )

    class FakeMailboxAccount:
        email = "mailbox@example.com"

    class FakeMailbox:
        def get_email(self):
            return FakeMailboxAccount()

    class FakeRegistrationService:
        def register(self, email=None, password=None):
            captured["email"] = email
            captured["password"] = password
            return expected

    monkeypatch.setattr(TraePlatform, "_registration_service", lambda self: FakeRegistrationService())

    instance = TraePlatform(RegisterConfig(), mailbox=FakeMailbox())
    instance._log_fn = logged.append

    result = instance.register(None, "secret")

    assert result is expected
    assert captured == {"email": None, "password": "secret"}
    assert logged == ["邮箱: mailbox@example.com"]


def test_trae_account_service_check_valid_uses_token():
    from platforms.trae.services.account import TraeAccountService

    service = TraeAccountService(RegisterConfig())

    account = Account(platform="trae", email="user@example.com", password="secret", token="trae-token")
    assert service.check_valid(account) is True

    account_without_token = Account(platform="trae", email="user@example.com", password="secret", token="")
    assert service.check_valid(account_without_token) is False


def test_trae_account_service_get_user_info_wraps_failure(monkeypatch):
    from platforms.trae.services.account import TraeAccountService
    import platforms.trae.services.account as account_module

    service = TraeAccountService(RegisterConfig())
    calls = {}

    def fake_get_user_info(token: str):
        calls["token"] = token
        return None

    monkeypatch.setattr(account_module, "get_trae_user_info", fake_get_user_info)

    account = Account(platform="trae", email="user@example.com", password="secret", token="trae-token")

    result = service.get_user_info(account)

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"] == "获取用户信息失败"
    assert calls["token"] == "trae-token"


def test_trae_desktop_service_switch_account_wraps_restart_result(monkeypatch):
    from platforms.trae.services.desktop import TraeDesktopService
    import platforms.trae.services.desktop as desktop_module

    monkeypatch.setattr(
        desktop_module,
        "switch_trae_account",
        lambda token, user_id, email, region: (True, "切换成功，请重启 Trae IDE 使新账号生效"),
    )
    monkeypatch.setattr(desktop_module, "restart_trae_ide", lambda: (True, "Trae IDE 已重启"))

    service = TraeDesktopService()
    account = Account(
        platform="trae",
        email="user@example.com",
        password="secret",
        user_id="trae-user-id",
        region="sg",
        token="trae-token",
    )

    result = service.switch_account(account)

    assert result["ok"] is True
    assert result["data"]["message"] == "切换成功，请重启 Trae IDE 使新账号生效。Trae IDE 已重启"


def test_trae_desktop_service_restart_ide_wraps_restart_result(monkeypatch):
    from platforms.trae.services.desktop import TraeDesktopService
    import platforms.trae.services.desktop as desktop_module

    monkeypatch.setattr(desktop_module, "restart_trae_ide", lambda: (True, "Trae IDE 已重启"))

    service = TraeDesktopService()

    result = service.restart_ide()

    assert result == {
        "ok": True,
        "data": {"message": "Trae IDE 已重启"},
    }


def test_trae_billing_service_falls_back_to_account_token(monkeypatch):
    from platforms.trae.services.billing import TraeBillingService
    import platforms.trae.services.billing as billing_module

    created = {}

    class FakeExecutor:
        def __enter__(self):
            created["entered"] = True
            return self

        def __exit__(self, exc_type, exc, tb):
            created["exited"] = True

    class FakePlatform:
        def _make_executor(self):
            created["make_executor_called"] = True
            return FakeExecutor()

    class FakeTraeRegister:
        def __init__(self, executor=None, log_fn=None):
            created["executor"] = executor
            created["log_fn"] = log_fn

        def step4_trae_login(self):
            created["step4_called"] = True

        def step5_get_token(self):
            created["step5_called"] = True
            return ""

        def step7_create_order(self, token):
            created["create_order_token"] = token
            return "https://cashier.example.com"

    monkeypatch.setattr(billing_module, "TraeRegister", FakeTraeRegister)

    service = TraeBillingService(FakePlatform(), log_fn=lambda msg: None)
    account = Account(platform="trae", email="user@example.com", password="secret", token="account-token")

    result = service.get_cashier_url(account)

    assert result["ok"] is True
    assert result["data"]["cashier_url"] == "https://cashier.example.com"
    assert result["data"]["message"] == "请在浏览器中打开升级链接完成 Pro 订阅"
    assert created["make_executor_called"] is True
    assert created["entered"] is True
    assert created["exited"] is True
    assert created["step4_called"] is True
    assert created["step5_called"] is True
    assert created["create_order_token"] == "account-token"


def test_trae_platform_execute_action_delegates_to_services(monkeypatch):
    from platforms.trae.plugin import TraePlatform

    instance = TraePlatform(RegisterConfig())
    account = Account(platform="trae", email="user@example.com", password="secret", token="token")
    calls = []

    class FakeAccountService:
        def get_user_info(self, delegated_account):
            calls.append(("get_user_info", delegated_account))
            return {"ok": True, "data": {"name": "Trae User"}}

    class FakeDesktopService:
        def switch_account(self, delegated_account):
            calls.append(("switch_account", delegated_account))
            return {"ok": True, "data": {"message": "desktop switched"}}

    class FakeBillingService:
        def get_cashier_url(self, delegated_account):
            calls.append(("get_cashier_url", delegated_account))
            return {
                "ok": True,
                "data": {
                    "cashier_url": "https://cashier.example.com",
                    "message": "请在浏览器中打开升级链接完成 Pro 订阅",
                },
            }

    monkeypatch.setattr(TraePlatform, "_account_service", lambda self: FakeAccountService())
    monkeypatch.setattr(TraePlatform, "_desktop_service", lambda self: FakeDesktopService())
    monkeypatch.setattr(TraePlatform, "_billing_service", lambda self: FakeBillingService())

    switch_result = instance.execute_action("switch_account", account, {})
    user_info_result = instance.execute_action("get_user_info", account, {})
    cashier_result = instance.execute_action("get_cashier_url", account, {})

    assert switch_result == {"ok": True, "data": {"message": "desktop switched"}}
    assert user_info_result == {"ok": True, "data": {"name": "Trae User"}}
    assert cashier_result == {
        "ok": True,
        "data": {
            "cashier_url": "https://cashier.example.com",
            "message": "请在浏览器中打开升级链接完成 Pro 订阅",
        },
    }
    assert calls == [
        ("switch_account", account),
        ("get_user_info", account),
        ("get_cashier_url", account),
    ]
