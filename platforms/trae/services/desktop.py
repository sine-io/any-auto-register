from platforms.trae.switch import restart_trae_ide, switch_trae_account


class TraeDesktopService:
    def restart_ide(self) -> dict:
        ok, msg = restart_trae_ide()
        if not ok:
            return {"ok": False, "error": msg}
        return {"ok": True, "data": {"message": msg}}

    def switch_account(self, account) -> dict:
        token = account.token
        if not token:
            return {"ok": False, "error": "账号缺少 token"}

        ok, msg = switch_trae_account(
            token,
            account.user_id or "",
            account.email or "",
            account.region or "",
        )
        if not ok:
            return {"ok": False, "error": msg}

        restart_ok, restart_msg = restart_trae_ide()
        return {
            "ok": True,
            "data": {
                "message": f"{msg}。{restart_msg}" if restart_ok else msg,
            },
        }
