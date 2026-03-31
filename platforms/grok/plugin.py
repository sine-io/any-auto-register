"""Grok (x.ai) 平台插件"""
from core.base_platform import BasePlatform, Account, RegisterConfig
from core.base_mailbox import BaseMailbox
from core.registry import register
from platforms.grok.services import (
    GrokCookieService,
    GrokRegistrationService,
    GrokSyncService,
)


@register
class GrokPlatform(BasePlatform):
    name = "grok"
    display_name = "Grok"
    version = "1.0.0"

    def __init__(self, config: RegisterConfig = None, mailbox: BaseMailbox = None):
        super().__init__(config)
        self.mailbox = mailbox

    def _registration_service(self) -> GrokRegistrationService:
        return GrokRegistrationService(
            config=self.config,
            mailbox=self.mailbox,
            log_fn=getattr(self, "_log_fn", print),
        )

    def _cookie_service(self) -> GrokCookieService:
        return GrokCookieService(self.config)

    def _sync_service(self) -> GrokSyncService:
        return GrokSyncService()

    def register(self, email: str, password: str = None) -> Account:
        return self._registration_service().register(email=email, password=password)

    def check_valid(self, account: Account) -> bool:
        return self._cookie_service().check_valid(account)

    def get_platform_actions(self) -> list:
        return [
            {"id": "upload_grok2api", "label": "导入 grok2api", "params": []},
        ]

    def execute_action(self, action_id: str, account: Account, params: dict) -> dict:
        if action_id == "upload_grok2api":
            return self._sync_service().upload_grok2api(account)
        raise NotImplementedError(f"未知操作: {action_id}")
