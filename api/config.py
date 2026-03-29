from fastapi import APIRouter
from pydantic import BaseModel
from core.config_store import config_store

router = APIRouter(prefix="/config", tags=["config"])

MASKED_SECRET_VALUE = "********"

CONFIG_KEYS = [
    "laoudo_auth", "laoudo_email", "laoudo_account_id",
    "yescaptcha_key", "twocaptcha_key",
    "default_executor", "default_captcha_solver",
    "duckmail_api_url", "duckmail_provider_url", "duckmail_bearer",
    "freemail_api_url", "freemail_admin_token", "freemail_username", "freemail_password",
    "moemail_api_url",
    "mail_provider",
    "cfworker_api_url", "cfworker_admin_token", "cfworker_domain", "cfworker_fingerprint",
    "cpa_api_url", "cpa_api_key",
    "team_manager_url", "team_manager_key",
    "cliproxyapi_management_key",
    "grok2api_url", "grok2api_app_key", "grok2api_pool", "grok2api_quota",
    "kiro_manager_path", "kiro_manager_exe",
]

SECRET_CONFIG_KEYS = {
    "laoudo_auth",
    "yescaptcha_key",
    "twocaptcha_key",
    "duckmail_bearer",
    "freemail_admin_token",
    "freemail_password",
    "cfworker_admin_token",
    "cpa_api_key",
    "team_manager_key",
    "cliproxyapi_management_key",
    "grok2api_app_key",
}


class ConfigUpdate(BaseModel):
    data: dict


@router.get("")
def get_config():
    all_cfg = config_store.get_all()
    payload = {}
    for key in CONFIG_KEYS:
        value = all_cfg.get(key, "")
        if key in SECRET_CONFIG_KEYS and value:
            payload[key] = MASKED_SECRET_VALUE
        else:
            payload[key] = value
    return payload


@router.put("")
def update_config(body: ConfigUpdate):
    safe = {}
    for key, value in body.data.items():
        if key not in CONFIG_KEYS:
            continue
        if key in SECRET_CONFIG_KEYS and value == MASKED_SECRET_VALUE:
            continue
        safe[key] = value
    config_store.set_many(safe)
    return {"ok": True, "updated": list(safe.keys())}
