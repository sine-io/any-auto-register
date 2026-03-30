from core.base_platform import RegisterConfig
from platforms.trae.switch import get_trae_user_info


class TraeAccountService:
    def __init__(self, config: RegisterConfig | None = None):
        self.config = config or RegisterConfig()

    def check_valid(self, account) -> bool:
        return bool(account.token)

    def get_user_info(self, account) -> dict:
        token = account.token
        if not token:
            return {"ok": False, "error": "账号缺少 token"}

        user_info = get_trae_user_info(token)
        if user_info:
            return {"ok": True, "data": user_info}
        return {"ok": False, "error": "获取用户信息失败"}
