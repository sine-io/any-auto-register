from platforms.cursor.switch import get_cursor_user_info, restart_cursor_ide, switch_cursor_account


class CursorDesktopService:
    def switch_account(self, account) -> dict:
        token = account.token
        if not token:
            return {"ok": False, "error": "账号缺少 token"}

        ok, msg = switch_cursor_account(token)
        if not ok:
            return {"ok": False, "error": msg}

        restart_ok, restart_msg = restart_cursor_ide()
        return {
            "ok": True,
            "data": {
                "message": f"{msg}。{restart_msg}" if restart_ok else msg,
            },
        }

    def get_user_info(self, account) -> dict:
        token = account.token
        if not token:
            return {"ok": False, "error": "账号缺少 token"}

        user_info = get_cursor_user_info(token)
        if user_info:
            return {"ok": True, "data": user_info}
        return {"ok": False, "error": "获取用户信息失败"}
