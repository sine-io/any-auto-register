import sys
from importlib.util import module_from_spec, spec_from_file_location
from pathlib import Path
from types import SimpleNamespace

from core.base_platform import Account as PlatformAccount
from core.base_platform import BasePlatform, RegisterConfig
from platforms.chatgpt.constants import OAUTH_CLIENT_ID


_INVALID_SUBSCRIPTION_STATUSES = {"expired", "invalid", "banned", None}


def _load_token_refresh_module():
    module_name = "platforms.chatgpt.token_refresh"
    module = sys.modules.get(module_name)
    if module is not None and hasattr(module, "TokenRefreshManager"):
        return module

    sys.modules.pop(module_name, None)
    spec = spec_from_file_location(module_name, Path(__file__).resolve().parent.parent / "token_refresh.py")
    if spec is None or spec.loader is None:
        raise ImportError(f"Unable to load {module_name}")

    module = module_from_spec(spec)
    module.Account = PlatformAccount
    sys.modules[module_name] = module
    spec.loader.exec_module(module)
    return module


def _load_payment_module():
    module_name = "platforms.chatgpt.payment"
    module = sys.modules.get(module_name)
    if module is not None and hasattr(module, "check_subscription_status"):
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


def check_subscription_status(account, proxy=None):
    return _load_payment_module().check_subscription_status(account, proxy=proxy)


class TokenRefreshManager:
    def __init__(self, proxy_url=None):
        self._manager = _load_token_refresh_module().TokenRefreshManager(proxy_url=proxy_url)

    def refresh_account(self, account):
        return self._manager.refresh_account(account)


def _build_account_adapter(account) -> SimpleNamespace:
    extra = getattr(account, "extra", None) or {}
    return SimpleNamespace(
        email=getattr(account, "email", ""),
        access_token=getattr(account, "access_token", None) or extra.get("access_token") or getattr(account, "token", ""),
        refresh_token=getattr(account, "refresh_token", None) or extra.get("refresh_token", ""),
        id_token=getattr(account, "id_token", None) or extra.get("id_token", ""),
        session_token=getattr(account, "session_token", None) or extra.get("session_token", ""),
        client_id=getattr(account, "client_id", None) or extra.get("client_id", OAUTH_CLIENT_ID),
        cookies=getattr(account, "cookies", None) or extra.get("cookies", ""),
    )


class ChatGPTTokenService:
    def __init__(self, config: RegisterConfig | None = None):
        self.config = config or RegisterConfig()

    def refresh_account_raw(self, account, proxy=None):
        manager = TokenRefreshManager(proxy_url=self.config.proxy if proxy is None else proxy)
        return manager.refresh_account(_build_account_adapter(account))

    def get_subscription_status_raw(self, account, proxy=None):
        return check_subscription_status(
            _build_account_adapter(account),
            proxy=self.config.proxy if proxy is None else proxy,
        )

    def check_valid(self, account) -> bool:
        try:
            status = self.get_subscription_status_raw(account)
            return status not in _INVALID_SUBSCRIPTION_STATUSES
        except Exception:
            return False

    def refresh_token(self, account) -> dict:
        result = self.refresh_account_raw(account)
        if result.success:
            return BasePlatform._action_success(
                {
                    "access_token": result.access_token,
                    "refresh_token": result.refresh_token,
                }
            )
        return BasePlatform._action_error(result.error_message)
