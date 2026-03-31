from __future__ import annotations

from core.base_platform import Account, AccountStatus, BasePlatform, RegisterConfig
from core.config_store import config_store
from platforms.grok.core import GrokRegister


OTP_CODE_PATTERN = r"[A-Z0-9]{3}-[A-Z0-9]{3}"


class GrokRegistrationService:
    def __init__(self, config: RegisterConfig | None = None, mailbox=None, log_fn=print):
        self.config = config or RegisterConfig()
        self.mailbox = mailbox
        self.log = log_fn

    def _make_captcha(self, **kwargs):
        return BasePlatform._make_captcha(self, **kwargs)

    def _resolve_yescaptcha_key(self) -> str:
        return self.config.extra.get("yescaptcha_key") or config_store.get("yescaptcha_key", "")

    def _build_otp_callback(self, mail_acct, before_ids: set):
        def otp_cb():
            self.log("等待验证码...")
            code = self.mailbox.wait_for_code(
                mail_acct,
                keyword="",
                before_ids=before_ids,
                code_pattern=OTP_CODE_PATTERN,
            )
            if code:
                code = code.replace("-", "").replace(" ", "")
                self.log(f"验证码: {code}")
            return code

        return otp_cb

    def register(self, email: str | None, password: str | None = None) -> Account:
        yescaptcha_key = self._resolve_yescaptcha_key()
        captcha_solver = self._make_captcha(key=yescaptcha_key)
        reg = GrokRegister(
            captcha_solver=captcha_solver,
            yescaptcha_key=yescaptcha_key,
            proxy=self.config.proxy,
            log_fn=self.log,
        )
        mailbox_attempts = 1 if email else int(self.config.extra.get("grok_mailbox_attempts", 8))
        last_error = None

        for attempt in range(1, mailbox_attempts + 1):
            mail_acct = None
            current_email = email
            if self.mailbox and not current_email:
                mail_acct = self.mailbox.get_email()
                current_email = mail_acct.email if mail_acct else None

            self.log(f"邮箱: {current_email}")
            before_ids = self.mailbox.get_current_ids(mail_acct) if (self.mailbox and mail_acct) else set()
            otp_callback = self._build_otp_callback(mail_acct, before_ids) if self.mailbox else None

            try:
                result = reg.register(
                    email=current_email,
                    password=password,
                    otp_callback=otp_callback,
                )
                break
            except Exception as exc:
                last_error = exc
                if attempt < mailbox_attempts and "邮箱域名被拒绝" in str(exc):
                    self.log(f"Grok 邮箱域名被拒绝，切换新邮箱重试 {attempt + 1}/{mailbox_attempts}")
                    continue
                raise
        else:
            raise last_error if last_error else RuntimeError("Grok 注册失败")

        return Account(
            platform="grok",
            email=result["email"],
            password=result["password"],
            status=AccountStatus.REGISTERED,
            extra={
                "sso": result["sso"],
                "sso_rw": result["sso_rw"],
                "given_name": result["given_name"],
                "family_name": result["family_name"],
            },
        )
