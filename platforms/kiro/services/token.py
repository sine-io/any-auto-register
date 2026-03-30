from __future__ import annotations

from core.base_mailbox import MailboxAccount, create_mailbox
from core.base_platform import RegisterConfig
from platforms.kiro.core import KiroRegister
from platforms.kiro.switch import refresh_kiro_token


OTP_CODE_PATTERN = r'(?is)(?:verification\s+code|验证码)[^0-9]{0,20}(\d{6})'


class KiroTokenService:
    def __init__(self, config: RegisterConfig | None = None, log_fn=print):
        self.config = config or RegisterConfig()
        self.log = log_fn

    def check_valid(self, account) -> bool:
        extra = account.extra or {}
        refresh_token = extra.get("refreshToken", "")
        if not refresh_token:
            return False

        try:
            ok, _ = refresh_kiro_token(
                refresh_token,
                extra.get("clientId", ""),
                extra.get("clientSecret", ""),
            )
            return ok
        except Exception:
            return False

    def refresh_token(self, account) -> dict:
        extra = account.extra or {}
        refresh_token = extra.get("refreshToken", "")
        client_id = extra.get("clientId", "")
        client_secret = extra.get("clientSecret", "")

        ok, result = refresh_kiro_token(refresh_token, client_id, client_secret)
        if ok:
            new_access = result["accessToken"]
            new_refresh = result.get("refreshToken", refresh_token)
            return {
                "ok": True,
                "data": {
                    "access_token": new_access,
                    "accessToken": new_access,
                    "refreshToken": new_refresh,
                },
            }
        return {"ok": False, "error": result.get("error", "刷新失败")}

    def ensure_desktop_tokens(self, account) -> dict:
        extra = account.extra or {}
        access_token = extra.get("accessToken", "") or account.token
        refresh_token = extra.get("refreshToken", "")
        client_id = extra.get("clientId", "")
        client_secret = extra.get("clientSecret", "")

        if not access_token:
            return {"ok": False, "error": "当前账号缺少 accessToken，无法切换到桌面应用"}

        if not refresh_token or not client_id or not client_secret:
            if account.email and account.password:
                ok, desktop_info = self._bootstrap_desktop_tokens(account)
                if not ok:
                    return {
                        "ok": False,
                        "error": (
                            "当前账号缺少 refreshToken / clientId / clientSecret，"
                            f"且自动补抓桌面端 Token 失败: {desktop_info.get('error', 'unknown error')}"
                        ),
                    }
                access_token = desktop_info.get("accessToken", "") or access_token
                refresh_token = desktop_info.get("refreshToken", "")
                client_id = desktop_info.get("clientId", "")
                client_secret = desktop_info.get("clientSecret", "")
            else:
                return {
                    "ok": False,
                    "error": (
                        "当前账号只有网页登录态，缺少 refreshToken / clientId / clientSecret，"
                        "并且没有可用的邮箱/密码用于自动补抓桌面端 Token。"
                    ),
                }

        return {
            "ok": True,
            "data": {
                "accessToken": access_token,
                "refreshToken": refresh_token,
                "clientId": client_id,
                "clientSecret": client_secret,
            },
        }

    def _bootstrap_desktop_tokens(self, account) -> tuple[bool, dict]:
        reg = KiroRegister(proxy=self.config.proxy, tag="KIRO-SWITCH")
        reg.log = self.log
        otp_callback = self._build_desktop_otp_callback(account, reg)
        return reg.fetch_desktop_tokens(
            account.email,
            account.password,
            otp_callback=otp_callback,
        )

    def _build_desktop_otp_callback(self, account, reg):
        mail_provider = self.config.extra.get("mail_provider", "")
        if not mail_provider:
            return None

        try:
            mailbox = create_mailbox(
                provider=mail_provider,
                extra=self.config.extra,
                proxy=self.config.proxy,
            )
            mail_account = MailboxAccount(email=account.email, account_id="")
            before_ids = mailbox.get_current_ids(mail_account)
        except Exception:
            return None

        def otp_cb():
            reg.log("桌面授权等待邮箱验证码 ...")
            try:
                code = mailbox.wait_for_code(
                    mail_account,
                    keyword="",
                    timeout=45,
                    before_ids=before_ids,
                    code_pattern=OTP_CODE_PATTERN,
                )
            except Exception:
                reg.log("未等到新验证码，回退读取最近一封身份验证邮件 ...")
                code = mailbox.wait_for_code(
                    mail_account,
                    keyword="",
                    timeout=15,
                    before_ids=None,
                    code_pattern=OTP_CODE_PATTERN,
                )
            if code:
                reg.log(f"桌面授权验证码: {code}")
            return code

        return otp_cb
