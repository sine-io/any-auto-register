"""Cursor 平台插件"""
from core.base_platform import BasePlatform, Account, AccountStatus, RegisterConfig
from core.base_mailbox import BaseMailbox
from core.registry import register
from platforms.cursor.services import (
    CursorAccountService,
    CursorDesktopService,
    CursorRegistrationService,
)


@register
class CursorPlatform(BasePlatform):
    name = "cursor"
    display_name = "Cursor"
    version = "1.0.0"

    def __init__(self, config: RegisterConfig = None, mailbox: BaseMailbox = None):
        super().__init__(config)
        self.mailbox = mailbox

    def _registration_service(self) -> CursorRegistrationService:
        return CursorRegistrationService(
            config=self.config,
            mailbox=self.mailbox,
            log_fn=getattr(self, "_log_fn", print),
        )

    def _account_service(self) -> CursorAccountService:
        return CursorAccountService(self.config)

    def _desktop_service(self) -> CursorDesktopService:
        return CursorDesktopService()

    def register(self, email: str, password: str = None) -> Account:
        return self._registration_service().register(email=email, password=password)

    def check_valid(self, account: Account) -> bool:
        return self._account_service().check_valid(account)

    def get_platform_actions(self) -> list:
        """返回平台支持的操作列表"""
        return [
            {"id": "switch_account", "label": "切换到桌面应用", "params": []},
            {"id": "get_user_info", "label": "获取用户信息", "params": []},
        ]

    def execute_action(self, action_id: str, account: Account, params: dict) -> dict:
        """执行平台操作"""
        if action_id == "switch_account":
            return self._desktop_service().switch_account(account)
        
        elif action_id == "get_user_info":
            return self._account_service().get_user_info(account)
        
        raise NotImplementedError(f"未知操作: {action_id}")
