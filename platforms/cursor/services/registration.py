from core.base_platform import Account, AccountStatus, RegisterConfig
from platforms.cursor.core import CursorRegister


class CursorRegistrationService:
    def __init__(self, config: RegisterConfig | None = None, mailbox=None, log_fn=print):
        self.config = config or RegisterConfig()
        self.mailbox = mailbox
        self.log = log_fn

    def register(self, email: str | None, password: str | None = None) -> Account:
        proxy = self.config.proxy
        yescaptcha_key = self.config.extra.get("yescaptcha_key", "")

        reg = CursorRegister(proxy=proxy, log_fn=self.log)

        mail_acct = self.mailbox.get_email() if self.mailbox else None
        current_email = email or (mail_acct.email if mail_acct else None)
        before_ids = self.mailbox.get_current_ids(mail_acct) if mail_acct else set()

        def otp_cb():
            self.log("等待验证码...")
            code = self.mailbox.wait_for_code(mail_acct, keyword="", before_ids=before_ids)
            if code:
                self.log(f"验证码: {code}")
            return code

        result = reg.register(
            email=current_email,
            password=password,
            otp_callback=otp_cb if self.mailbox else None,
            yescaptcha_key=yescaptcha_key,
        )

        return Account(
            platform="cursor",
            email=result["email"],
            password=result["password"],
            token=result["token"],
            status=AccountStatus.REGISTERED,
        )
