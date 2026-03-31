import random
import string

from core.base_mailbox import TempMailLolMailbox
from core.base_platform import Account, AccountStatus, RegisterConfig
from platforms.chatgpt.register_v2 import RegistrationEngineV2


_DEFAULT_REGISTER_MAX_RETRIES = 3
_PASSWORD_POPULATION = string.ascii_letters + string.digits + "!@#$"


class _GenericEmailService:
    service_type = type("ServiceType", (), {"value": "custom_provider"})()

    def __init__(self, mailbox, fixed_email: str | None = None):
        self._mailbox = mailbox
        self._acct = None
        self._email = fixed_email
        self._fixed_email = fixed_email

    def create_email(self, config=None):
        if self._email and self._acct and self._fixed_email:
            return {"email": self._email, "service_id": self._acct.account_id, "token": ""}

        self._acct = self._mailbox.get_email()
        if not self._email or not self._fixed_email:
            self._email = self._acct.email

        return {"email": self._email, "service_id": self._acct.account_id, "token": ""}

    def get_verification_code(
        self,
        email=None,
        email_id=None,
        timeout: int = 120,
        pattern=None,
        otp_sent_at=None,
        exclude_codes=None,
    ):
        if not self._acct:
            raise RuntimeError("邮箱账户尚未创建，无法获取验证码")

        return self._mailbox.wait_for_code(
            self._acct,
            keyword="",
            timeout=timeout,
            otp_sent_at=otp_sent_at,
            exclude_codes=exclude_codes,
        )

    def update_status(self, success, error=None):
        return None

    @property
    def status(self):
        return None


class _TempMailEmailService:
    service_type = type("ServiceType", (), {"value": "tempmail_lol"})()

    def __init__(self, mailbox):
        self._mailbox = mailbox
        self._acct = None

    def create_email(self, config=None):
        self._acct = self._mailbox.get_email()
        return {
            "email": self._acct.email,
            "service_id": self._acct.account_id,
            "token": self._acct.account_id,
        }

    def get_verification_code(
        self,
        email=None,
        email_id=None,
        timeout: int = 120,
        pattern=None,
        otp_sent_at=None,
        exclude_codes=None,
    ):
        return self._mailbox.wait_for_code(
            self._acct,
            keyword="",
            timeout=timeout,
            otp_sent_at=otp_sent_at,
            exclude_codes=exclude_codes,
        )

    def update_status(self, success, error=None):
        return None

    @property
    def status(self):
        return None


class ChatGPTRegistrationService:
    def __init__(self, config: RegisterConfig | None = None, mailbox=None, log_fn=print):
        self.config = config or RegisterConfig()
        self.mailbox = mailbox
        self.log = log_fn

    def register(self, email: str | None = None, password: str | None = None) -> Account:
        if not password:
            password = "".join(random.choices(_PASSWORD_POPULATION, k=16))

        proxy = self.config.proxy
        max_retries = self._get_register_max_retries()

        if self.mailbox:
            engine = RegistrationEngineV2(
                email_service=_GenericEmailService(self.mailbox, fixed_email=email),
                proxy_url=proxy,
                callback_logger=self.log,
                max_retries=max_retries,
            )
            engine.email = email
            engine.password = password
        else:
            engine = RegistrationEngineV2(
                email_service=_TempMailEmailService(TempMailLolMailbox(proxy=proxy)),
                proxy_url=proxy,
                callback_logger=self.log,
                max_retries=max_retries,
            )
            if email:
                engine.email = email
                engine.password = password

        result = engine.run()
        if not result or not result.success:
            raise RuntimeError(result.error_message if result else "注册失败")

        return Account(
            platform="chatgpt",
            email=result.email,
            password=result.password or password,
            user_id=result.account_id,
            token=result.access_token,
            status=AccountStatus.REGISTERED,
            extra={
                "access_token": getattr(result, "access_token", ""),
                "refresh_token": getattr(result, "refresh_token", ""),
                "id_token": getattr(result, "id_token", ""),
                "session_token": getattr(result, "session_token", ""),
                "workspace_id": getattr(result, "workspace_id", ""),
            },
        )

    def _get_register_max_retries(self) -> int:
        try:
            return int((self.config.extra or {}).get("register_max_retries", _DEFAULT_REGISTER_MAX_RETRIES) or _DEFAULT_REGISTER_MAX_RETRIES)
        except Exception:
            return _DEFAULT_REGISTER_MAX_RETRIES
