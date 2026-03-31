"""ChatGPT / Codex CLI 平台插件"""
from core.base_mailbox import BaseMailbox
from core.base_platform import Account, BasePlatform, RegisterConfig
from core.registry import register
from platforms.chatgpt.services import (
    ChatGPTBillingService,
    ChatGPTExternalSyncService,
    ChatGPTRegistrationService,
    ChatGPTTokenService,
)


@register
class ChatGPTPlatform(BasePlatform):
    name = "chatgpt"
    display_name = "ChatGPT"
    version = "1.0.0"

    def __init__(self, config: RegisterConfig = None, mailbox: BaseMailbox = None):
        super().__init__(config)
        self.mailbox = mailbox

    def _registration_service(self) -> ChatGPTRegistrationService:
        return ChatGPTRegistrationService(
            config=self.config,
            mailbox=self.mailbox,
            log_fn=getattr(self, "_log_fn", print),
        )

    def _token_service(self) -> ChatGPTTokenService:
        return ChatGPTTokenService(self.config)

    def _billing_service(self) -> ChatGPTBillingService:
        return ChatGPTBillingService(self.config)

    def _external_sync_service(self) -> ChatGPTExternalSyncService:
        return ChatGPTExternalSyncService()

    def check_valid(self, account: Account) -> bool:
        return self._token_service().check_valid(account)

    def register(self, email: str = None, password: str = None) -> Account:
        return self._registration_service().register(email=email, password=password)

    def get_platform_actions(self) -> list:
        return [
            {"id": "refresh_token", "label": "刷新 Token", "params": []},
            {
                "id": "payment_link",
                "label": "生成支付链接",
                "params": [
                    {"key": "country", "label": "地区", "type": "select", "options": ["US", "SG", "TR", "HK", "JP", "GB", "AU", "CA"]},
                    {"key": "plan", "label": "套餐", "type": "select", "options": ["plus", "team"]},
                ],
            },
            {
                "id": "upload_cpa",
                "label": "上传 CPA",
                "params": [
                    {"key": "api_url", "label": "CPA API URL", "type": "text"},
                    {"key": "api_key", "label": "CPA API Key", "type": "text"},
                ],
            },
            {
                "id": "upload_tm",
                "label": "上传 Team Manager",
                "params": [
                    {"key": "api_url", "label": "TM API URL", "type": "text"},
                    {"key": "api_key", "label": "TM API Key", "type": "text"},
                ],
            },
        ]

    def execute_action(self, action_id: str, account: Account, params: dict) -> dict:
        if action_id == "refresh_token":
            return self._token_service().refresh_token(account)

        elif action_id == "payment_link":
            return self._billing_service().payment_link(
                account,
                plan=params.get("plan", "plus"),
                country=params.get("country", "US"),
            )

        elif action_id == "upload_cpa":
            return self._external_sync_service().upload_cpa(
                account,
                api_url=params.get("api_url"),
                api_key=params.get("api_key"),
            )

        elif action_id == "upload_tm":
            return self._external_sync_service().upload_tm(
                account,
                api_url=params.get("api_url"),
                api_key=params.get("api_key"),
            )

        raise NotImplementedError(f"未知操作: {action_id}")
