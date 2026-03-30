from __future__ import annotations

from importlib import import_module
from typing import TYPE_CHECKING


__all__ = [
    "KiroRegistrationService",
    "KiroTokenService",
    "KiroDesktopService",
    "KiroManagerSyncService",
]

_SERVICE_MODULES = {
    "KiroRegistrationService": "registration",
    "KiroTokenService": "token",
    "KiroDesktopService": "desktop",
    "KiroManagerSyncService": "manager_sync",
}


if TYPE_CHECKING:
    from .desktop import KiroDesktopService
    from .manager_sync import KiroManagerSyncService
    from .registration import KiroRegistrationService
    from .token import KiroTokenService


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
