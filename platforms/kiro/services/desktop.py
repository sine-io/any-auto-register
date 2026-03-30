from __future__ import annotations

from core.base_platform import RegisterConfig
from platforms.kiro.services.token import KiroTokenService
from platforms.kiro.switch import restart_kiro_ide, switch_kiro_account


class KiroDesktopService:
    def __init__(
        self,
        config: RegisterConfig | None = None,
        token_service: KiroTokenService | None = None,
        log_fn=print,
    ):
        self.config = config or RegisterConfig()
        self.log = log_fn
        self.token_service = token_service or KiroTokenService(config=self.config, log_fn=self.log)

    def restart_ide(self) -> dict:
        ok, msg = restart_kiro_ide()
        if not ok:
            return {"ok": False, "error": msg}
        return {"ok": True, "data": {"message": msg}}

    def switch_account(self, account) -> dict:
        ensured = self.token_service.ensure_desktop_tokens(account)
        if not ensured.get("ok"):
            return ensured

        data = ensured.get("data", {})
        access_token = data.get("accessToken", "")
        refresh_token = data.get("refreshToken", "")
        client_id = data.get("clientId", "")
        client_secret = data.get("clientSecret", "")

        ok, msg = switch_kiro_account(
            access_token=access_token,
            refresh_token=refresh_token,
            client_id=client_id,
            client_secret=client_secret,
        )
        if not ok:
            return {"ok": False, "error": msg}

        restart_ok, restart_msg = restart_kiro_ide()
        return {
            "ok": True,
            "data": {
                "accessToken": access_token,
                "refreshToken": refresh_token,
                "clientId": client_id,
                "clientSecret": client_secret,
                "message": f"{msg}。{restart_msg}" if restart_ok else msg,
            },
        }
