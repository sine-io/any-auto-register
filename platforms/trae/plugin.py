"""Trae.ai 平台插件"""
from core.base_platform import BasePlatform, Account, RegisterConfig
from core.base_mailbox import BaseMailbox
from core.registry import register
from platforms.trae.services import (
    TraeAccountService,
    TraeBillingService,
    TraeDesktopService,
    TraeRegistrationService,
)


@register
class TraePlatform(BasePlatform):
    name = "trae"
    display_name = "Trae.ai"
    version = "1.0.0"

    def __init__(self, config: RegisterConfig = None, mailbox: BaseMailbox = None):
        super().__init__(config)
        self.mailbox = mailbox

    def _registration_service(self) -> TraeRegistrationService:
        return TraeRegistrationService(
            config=self.config,
            mailbox=self.mailbox,
            log_fn=getattr(self, "_log_fn", print),
        )

    def _account_service(self) -> TraeAccountService:
        return TraeAccountService(self.config)

    def _desktop_service(self) -> TraeDesktopService:
        return TraeDesktopService()

    def _billing_service(self) -> TraeBillingService:
        return TraeBillingService(self, log_fn=getattr(self, "_log_fn", print))

    def _service_action_result(self, result: dict) -> dict:
        if result.get("ok"):
            return self._action_success(result.get("data"))
        return self._action_error(result.get("error", "操作失败"))

    def register(self, email: str, password: str = None) -> Account:
        log = getattr(self, '_log_fn', print)
        mail_acct = self.mailbox.get_email() if self.mailbox and not email else None
        logged_email = email or (mail_acct.email if mail_acct else None)
        log(f"邮箱: {logged_email}")
        return self._registration_service().register(email=email, password=password)

    def check_valid(self, account: Account) -> bool:
        return self._account_service().check_valid(account)

    def get_platform_actions(self) -> list:
        """返回平台支持的操作列表"""
        return [
            {"id": "switch_account", "label": "切换到桌面应用", "params": []},
            {"id": "get_user_info", "label": "获取用户信息", "params": []},
            {"id": "get_cashier_url", "label": "获取升级链接", "params": []},
        ]

    def execute_action(self, action_id: str, account: Account, params: dict) -> dict:
        """执行平台操作"""
        if action_id == "switch_account":
            return self._service_action_result(self._desktop_service().switch_account(account))

        elif action_id == "get_user_info":
            return self._service_action_result(self._account_service().get_user_info(account))

        elif action_id == "get_cashier_url":
            return self._service_action_result(self._billing_service().get_cashier_url(account))

        raise NotImplementedError(f"未知操作: {action_id}")
