from __future__ import annotations

from core.base_platform import BasePlatform
from platforms.grok.grok2api_upload import upload_to_grok2api


class GrokSyncService:
    def upload_grok2api_raw(self, account, api_url=None, app_key=None) -> tuple[bool, str]:
        kwargs = {}
        if api_url is not None:
            kwargs["api_url"] = api_url
        if app_key is not None:
            kwargs["app_key"] = app_key
        if kwargs:
            return upload_to_grok2api(account, **kwargs)
        return upload_to_grok2api(account)

    def upload_grok2api(self, account) -> dict:
        ok, msg = self.upload_grok2api_raw(account)
        if ok:
            return BasePlatform._action_success(message=msg)
        return BasePlatform._action_error(msg)
