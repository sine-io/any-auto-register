"""Kiro 平台插件 - 基于 AWS Builder ID 注册"""
from core.base_platform import BasePlatform, Account, RegisterConfig
from core.base_mailbox import BaseMailbox
from core.registry import register
from platforms.kiro.services import (
    KiroDesktopService,
    KiroManagerSyncService,
    KiroRegistrationService,
    KiroTokenService,
)


@register
class KiroPlatform(BasePlatform):
    name = "kiro"
    display_name = "Kiro (AWS Builder ID)"
    version = "1.0.0"

    def __init__(self, config: RegisterConfig = None, mailbox: BaseMailbox = None):
        super().__init__(config)
        self.mailbox = mailbox

    def _registration_service(self) -> KiroRegistrationService:
        return KiroRegistrationService(
            config=self.config,
            mailbox=self.mailbox,
            log_fn=getattr(self, "_log_fn", print),
        )

    def _token_service(self) -> KiroTokenService:
        return KiroTokenService(
            config=self.config,
            log_fn=getattr(self, "_log_fn", print),
        )

    def _desktop_service(self) -> KiroDesktopService:
        return KiroDesktopService(
            config=self.config,
            token_service=self._token_service(),
            log_fn=getattr(self, "_log_fn", print),
        )

    def _manager_sync_service(self) -> KiroManagerSyncService:
        return KiroManagerSyncService()

    def register(self, email: str, password: str = None) -> Account:
        log_fn = getattr(self, "_log_fn", print)
        mailbox_account = None
        if self.mailbox:
            mailbox_account = self.mailbox.get_email()
            log_fn(f"邮箱: {mailbox_account.email}")
        return self._registration_service().register(
            email=email,
            password=password,
            mailbox_account=mailbox_account,
        )

    def check_valid(self, account: Account) -> bool:
        """通过 refreshToken 检测账号是否有效"""
        return self._token_service().check_valid(account)

    def get_platform_actions(self) -> list:
        return [
            {"id": "switch_account", "label": "切换到桌面应用", "params": []},
            {"id": "refresh_token", "label": "刷新 Token", "params": []},
            {"id": "upload_kiro_manager", "label": "导入 Kiro Manager", "params": []},
        ]

    def execute_action(self, action_id: str, account: Account, params: dict) -> dict:
        if action_id == "switch_account":
            return self._desktop_service().switch_account(account)

        elif action_id == "refresh_token":
            return self._token_service().refresh_token(account)

        elif action_id == "upload_kiro_manager":
            return self._manager_sync_service().upload(account)

        raise NotImplementedError(f"未知操作: {action_id}")
