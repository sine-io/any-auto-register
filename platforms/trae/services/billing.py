from platforms.trae.core import TraeRegister


class TraeBillingService:
    def __init__(self, platform, log_fn=print):
        self.platform = platform
        self.log = log_fn

    def get_cashier_url(self, account) -> dict:
        with self.platform._make_executor() as ex:
            reg = TraeRegister(executor=ex, log_fn=self.log)
            reg.step4_trae_login()
            token = reg.step5_get_token() or account.token
            cashier_url = reg.step7_create_order(token)

        if not cashier_url:
            return {"ok": False, "error": "获取升级链接失败，token 可能已过期，请重新注册"}

        return {
            "ok": True,
            "data": {
                "cashier_url": cashier_url,
                "message": "请在浏览器中打开升级链接完成 Pro 订阅",
            },
        }
