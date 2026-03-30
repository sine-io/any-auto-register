from __future__ import annotations

from core.base_platform import Account, AccountStatus, RegisterConfig
from platforms.kiro.core import KiroRegister


OTP_CODE_PATTERN = r'(?is)(?:verification\s+code|验证码)[^0-9]{0,20}(\d{6})'


class KiroRegistrationService:
    def __init__(self, config: RegisterConfig | None = None, mailbox=None, log_fn=print):
        self.config = config or RegisterConfig()
        self.mailbox = mailbox
        self.log = log_fn

    def register(self, email: str | None, password: str | None = None) -> Account:
        proxy = self.config.proxy
        mail_token = self.config.extra.get("laoudo_account_id", "")
        otp_timeout = int(self.config.extra.get("otp_timeout", 120))

        reg = KiroRegister(proxy=proxy, tag="KIRO")
        reg.log = lambda msg: self.log(msg)

        mail_acct = self.mailbox.get_email() if self.mailbox else None
        current_email = email or (mail_acct.email if mail_acct else None)
        before_ids = self.mailbox.get_current_ids(mail_acct) if mail_acct else set()

        def otp_cb():
            self.log("等待验证码...")
            code = self.mailbox.wait_for_code(
                mail_acct,
                keyword="builder id",
                timeout=otp_timeout,
                before_ids=before_ids,
                code_pattern=OTP_CODE_PATTERN,
            )
            if code:
                self.log(f"验证码: {code}")
            return code

        ok, info = reg.register(
            email=current_email,
            pwd=password,
            name=self.config.extra.get("name", "Kiro User"),
            mail_token=mail_token or None,
            otp_timeout=otp_timeout,
            otp_callback=otp_cb if self.mailbox else None,
        )
        if not ok:
            raise RuntimeError(f"Kiro 注册失败: {info.get('error')}")

        return Account(
            platform="kiro",
            email=info["email"],
            password=info["password"],
            status=AccountStatus.REGISTERED,
            extra={
                "name": info.get("name", ""),
                "accessToken": info.get("accessToken", ""),
                "sessionToken": info.get("sessionToken", ""),
                "clientId": info.get("clientId", ""),
                "clientSecret": info.get("clientSecret", ""),
                "clientIdHash": info.get("clientIdHash", ""),
                "refreshToken": info.get("refreshToken", ""),
                "webAccessToken": info.get("webAccessToken", ""),
                "region": info.get("region", "us-east-1"),
                "provider": "BuilderId",
                "authMethod": "IdC",
            },
        )
