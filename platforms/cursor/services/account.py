from platforms.cursor.core import CURSOR, UA


class CursorAccountService:
    def __init__(self, config=None):
        self.config = config

    def _fetch_auth_me(self, token: str):
        from curl_cffi import requests as curl_req

        return curl_req.get(
            f"{CURSOR}/api/auth/me",
            headers={
                "Cookie": f"WorkosCursorSessionToken={token}",
                "user-agent": UA,
            },
            impersonate="chrome124",
            timeout=15,
        )

    def check_valid(self, account) -> bool:
        if not account.token:
            return False
        try:
            response = self._fetch_auth_me(account.token)
            return response.status_code == 200
        except Exception:
            return False

    def get_user_info(self, account) -> dict:
        token = account.token
        if not token:
            return {"ok": False, "error": "账号缺少 token"}

        try:
            response = self._fetch_auth_me(token)
            if response.status_code == 200:
                return {"ok": True, "data": response.json()}
        except Exception:
            pass
        return {"ok": False, "error": "获取用户信息失败"}
