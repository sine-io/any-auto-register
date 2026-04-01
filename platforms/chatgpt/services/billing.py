import sys
from importlib.util import module_from_spec, spec_from_file_location
from pathlib import Path
from types import SimpleNamespace

from core.base_platform import Account as PlatformAccount
from core.base_platform import BasePlatform, RegisterConfig


def _load_payment_module():
    module_name = "platforms.chatgpt.payment"
    module = sys.modules.get(module_name)
    if module is not None and hasattr(module, "generate_plus_link") and hasattr(module, "generate_team_link"):
        return module

    sys.modules.pop(module_name, None)
    spec = spec_from_file_location(module_name, Path(__file__).resolve().parent.parent / "payment.py")
    if spec is None or spec.loader is None:
        raise ImportError(f"Unable to load {module_name}")

    module = module_from_spec(spec)
    module.Account = PlatformAccount
    sys.modules[module_name] = module
    spec.loader.exec_module(module)
    return module


def generate_plus_link(account, proxy=None, country="SG"):
    return _load_payment_module().generate_plus_link(account, proxy=proxy, country=country)


def generate_team_link(
    account,
    workspace_name="MyTeam",
    seat_quantity=5,
    price_interval="month",
    proxy=None,
    country="SG",
):
    return _load_payment_module().generate_team_link(
        account,
        workspace_name=workspace_name,
        seat_quantity=seat_quantity,
        price_interval=price_interval,
        proxy=proxy,
        country=country,
    )


def _build_account_adapter(account) -> SimpleNamespace:
    extra = getattr(account, "extra", None) or {}
    return SimpleNamespace(
        email=getattr(account, "email", ""),
        access_token=getattr(account, "access_token", None) or extra.get("access_token") or getattr(account, "token", ""),
        cookies=getattr(account, "cookies", None) or extra.get("cookies", ""),
    )


class ChatGPTBillingService:
    def __init__(self, config: RegisterConfig | None = None):
        self.config = config or RegisterConfig()

    def generate_payment_link_raw(
        self,
        account,
        plan,
        country,
        proxy=None,
        workspace_name="MyTeam",
        seat_quantity=5,
        price_interval="month",
    ):
        account_adapter = _build_account_adapter(account)
        proxy = self.config.proxy if proxy is None else proxy
        if plan == "plus":
            return generate_plus_link(account_adapter, proxy=proxy, country=country)
        return generate_team_link(
            account_adapter,
            workspace_name=workspace_name,
            seat_quantity=seat_quantity,
            price_interval=price_interval,
            proxy=proxy,
            country=country,
        )

    def payment_link(self, account, plan: str = "plus", country: str = "US") -> dict:
        account_adapter = _build_account_adapter(account)
        if plan == "plus":
            url = generate_plus_link(account_adapter, proxy=self.config.proxy, country=country)
        else:
            url = generate_team_link(account_adapter, proxy=self.config.proxy, country=country)

        if url:
            return BasePlatform._action_success({"url": url})
        return BasePlatform._action_error("生成支付链接失败")
