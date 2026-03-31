from __future__ import annotations

from core.base_platform import RegisterConfig


class GrokCookieService:
    def __init__(self, config: RegisterConfig | None = None):
        self.config = config or RegisterConfig()

    def check_valid(self, account) -> bool:
        return bool((account.extra or {}).get("sso"))
