from types import SimpleNamespace

from core.base_platform import BasePlatform
from platforms.chatgpt.constants import OAUTH_CLIENT_ID


def generate_token_json(account):
    from platforms.chatgpt.cpa_upload import generate_token_json as _generate_token_json

    return _generate_token_json(account)


def upload_to_cpa(token_data, api_url=None, api_key=None):
    from platforms.chatgpt.cpa_upload import upload_to_cpa as _upload_to_cpa

    return _upload_to_cpa(token_data, api_url=api_url, api_key=api_key)


def upload_to_team_manager(account, api_url=None, api_key=None):
    from platforms.chatgpt.cpa_upload import upload_to_team_manager as _upload_to_team_manager

    return _upload_to_team_manager(account, api_url=api_url, api_key=api_key)


def _build_account_adapter(account) -> SimpleNamespace:
    extra = account.extra or {}
    return SimpleNamespace(
        email=account.email,
        access_token=extra.get("access_token") or account.token,
        refresh_token=extra.get("refresh_token", ""),
        id_token=extra.get("id_token", ""),
        session_token=extra.get("session_token", ""),
        client_id=extra.get("client_id", OAUTH_CLIENT_ID),
        cookies=extra.get("cookies", ""),
    )


class ChatGPTExternalSyncService:
    def upload_cpa(self, account, api_url: str | None = None, api_key: str | None = None) -> dict:
        token_data = generate_token_json(_build_account_adapter(account))
        ok, msg = upload_to_cpa(token_data, api_url=api_url, api_key=api_key)
        if ok:
            return BasePlatform._action_success(message=msg)
        return BasePlatform._action_error(msg)

    def upload_tm(self, account, api_url: str | None = None, api_key: str | None = None) -> dict:
        ok, msg = upload_to_team_manager(
            _build_account_adapter(account),
            api_url=api_url,
            api_key=api_key,
        )
        if ok:
            return BasePlatform._action_success(message=msg)
        return BasePlatform._action_error(msg)
