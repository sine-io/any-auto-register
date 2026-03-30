"""平台插件基类"""
from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Optional
from enum import Enum
import time


class AccountStatus(str, Enum):
    REGISTERED   = "registered"
    TRIAL        = "trial"
    SUBSCRIBED   = "subscribed"
    EXPIRED      = "expired"
    INVALID      = "invalid"


@dataclass
class Account:
    platform: str
    email: str
    password: str
    user_id: str = ""
    region: str = ""
    token: str = ""
    status: AccountStatus = AccountStatus.REGISTERED
    trial_end_time: int = 0       # unix timestamp
    extra: dict = field(default_factory=dict)  # 平台自定义字段
    created_at: int = field(default_factory=lambda: int(time.time()))


@dataclass
class RegisterConfig:
    """注册任务配置"""
    executor_type: str = "protocol"   # protocol | headless | headed
    captcha_solver: str = "yescaptcha"  # yescaptcha | 2captcha | manual
    proxy: Optional[str] = None
    extra: dict = field(default_factory=dict)


def make_executor_from_config(config: RegisterConfig):
    """根据 config 创建执行器"""
    from .executors.protocol import ProtocolExecutor

    t = config.executor_type
    if t == "protocol":
        return ProtocolExecutor(proxy=config.proxy)
    elif t == "headless":
        from .executors.playwright import PlaywrightExecutor

        return PlaywrightExecutor(proxy=config.proxy, headless=True)
    elif t == "headed":
        from .executors.playwright import PlaywrightExecutor

        return PlaywrightExecutor(proxy=config.proxy, headless=False)
    raise ValueError(f"未知执行器类型: {t}")


class BasePlatform(ABC):
    # 子类必须定义
    name: str = ""
    display_name: str = ""
    version: str = "1.0.0"
    # 子类声明支持的执行器类型，未列出的自动降级到 protocol
    supported_executors: list = ["protocol", "headless", "headed"]

    @classmethod
    def is_available(cls) -> bool:
        """Whether the platform can be used in the current runtime."""
        return True

    @classmethod
    def get_unavailable_reason(cls) -> str:
        return ""

    def __init__(self, config: RegisterConfig = None):
        self.config = config or RegisterConfig()
        if self.config.executor_type not in self.supported_executors:
            raise NotImplementedError(
                f"{self.display_name} 暂不支持 '{self.config.executor_type}' 执行器，"
                f"当前支持: {self.supported_executors}"
            )

    @abstractmethod
    def register(self, email: str, password: str = None) -> Account:
        """执行注册流程，返回 Account"""
        ...

    @abstractmethod
    def check_valid(self, account: Account) -> bool:
        """检测账号是否有效"""
        ...

    def get_trial_url(self, account: Account) -> Optional[str]:
        """生成试用激活链接（可选实现）"""
        return None

    def get_platform_actions(self) -> list:
        """
        返回平台支持的额外操作列表，每项格式:
        {"id": str, "label": str, "params": [{"key": str, "label": str, "type": str}]}
        """
        return []

    def get_action_availability(self, action_id: str) -> tuple[bool, str]:
        return True, ""

    def execute_action(self, action_id: str, account: Account, params: dict) -> dict:
        """
        执行平台特定操作，返回 {"ok": bool, "data": any, "error": str}
        """
        raise NotImplementedError(f"平台 {self.name} 不支持操作: {action_id}")

    @staticmethod
    def _action_success(data=None, message: str = "") -> dict:
        if data is None:
            payload = {}
        elif isinstance(data, dict):
            payload = dict(data)
        else:
            payload = {"value": data}

        if message and "message" not in payload:
            payload["message"] = message
        return {"ok": True, "data": payload}

    @staticmethod
    def _action_error(error: object) -> dict:
        return {"ok": False, "error": str(error)}

    def get_quota(self, account: Account) -> dict:
        """查询账号配额（可选实现）"""
        return {}

    def _make_executor(self):
        """根据 config 创建执行器"""
        return make_executor_from_config(self.config)

    def _make_captcha(self, **kwargs):
        """根据 config 创建验证码解决器"""
        from .base_captcha import YesCaptcha, ManualCaptcha, LocalSolverCaptcha
        t = self.config.captcha_solver
        if t == "yescaptcha":
            key = kwargs.get("key") or self.config.extra.get("yescaptcha_key", "")
            return YesCaptcha(key)
        elif t == "manual":
            return ManualCaptcha()
        elif t == "local_solver":
            url = self.config.extra.get("solver_url", "http://localhost:8889")
            return LocalSolverCaptcha(url)
        raise ValueError(f"未知验证码解决器: {t}")
