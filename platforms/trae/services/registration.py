from core.base_platform import Account, AccountStatus, RegisterConfig, make_executor_from_config
from platforms.trae.core import TraeRegister


class TraeRegistrationService:
    def __init__(self, config: RegisterConfig | None = None, mailbox=None, log_fn=print):
        self.config = config or RegisterConfig()
        self.mailbox = mailbox
        self.log = log_fn

    def _make_executor(self):
        return make_executor_from_config(self.config)

    def register(self, email: str | None, password: str | None = None) -> Account:
        mail_acct = self.mailbox.get_email() if self.mailbox else None
        current_email = email or (mail_acct.email if mail_acct else None)
        before_ids = self.mailbox.get_current_ids(mail_acct) if mail_acct else set()

        def otp_cb():
            self.log("等待验证码...")
            code = self.mailbox.wait_for_code(mail_acct, keyword="", before_ids=before_ids)
            if code:
                self.log(f"验证码: {code}")
            return code

        with self._make_executor() as ex:
            reg = TraeRegister(executor=ex, log_fn=self.log)
            result = reg.register(
                email=current_email,
                password=password,
                otp_callback=otp_cb if self.mailbox else None,
            )

        return Account(
            platform="trae",
            email=result["email"],
            password=result["password"],
            user_id=result["user_id"],
            token=result["token"],
            region=result["region"],
            status=AccountStatus.REGISTERED,
            extra={
                "cashier_url": result["cashier_url"],
                "ai_pay_host": result["ai_pay_host"],
            },
        )
