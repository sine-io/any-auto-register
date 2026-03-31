from __future__ import annotations

from importlib import import_module
from typing import TYPE_CHECKING


__all__ = [
    "ChatGPTRegistrationService",
    "ChatGPTTokenService",
    "ChatGPTBillingService",
    "ChatGPTExternalSyncService",
]

_SERVICE_MODULES = {
    "ChatGPTRegistrationService": "registration",
    "ChatGPTTokenService": "token",
    "ChatGPTBillingService": "billing",
    "ChatGPTExternalSyncService": "external_sync",
}


if TYPE_CHECKING:
    from .billing import ChatGPTBillingService
    from .external_sync import ChatGPTExternalSyncService
    from .registration import ChatGPTRegistrationService
    from .token import ChatGPTTokenService


def __getattr__(name: str):
    module_name = _SERVICE_MODULES.get(name)
    if module_name is None:
        raise AttributeError(f"module {__name__!r} has no attribute {name!r}")

    module = import_module(f".{module_name}", __name__)
    value = getattr(module, name)
    globals()[name] = value
    return value


def __dir__():
    return sorted(set(globals()) | set(__all__))
