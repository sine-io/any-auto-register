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


def test_trae_account_service_check_valid_uses_token(monkeypatch):
    from platforms.trae.services.account import TraeAccountService

    calls = {}

    class FakeResponse:
        status_code = 200

    def fake_fetch(token: str):
        calls["token"] = token
        return FakeResponse()

    service = TraeAccountService(RegisterConfig())
    monkeypatch.setattr(service, "_fetch_user_token", fake_fetch)

    account = Account(platform="trae", email="user@example.com", password="secret", token="trae-token")
    assert service.check_valid(account) is True
    assert calls["token"] == "trae-token"


def test_trae_account_service_get_user_info_wraps_failure():
    from platforms.trae.services.account import TraeAccountService

    service = TraeAccountService(RegisterConfig())
    account = Account(platform="trae", email="user@example.com", password="secret", token="")

    result = service.get_user_info(account)

    assert result["ok"] is False
    assert isinstance(result.get("error"), str)
    assert result["error"] == "账号缺少 token"


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
    assert created["create_order_token"] == "account-token"

