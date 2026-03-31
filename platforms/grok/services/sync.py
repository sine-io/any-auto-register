from __future__ import annotations

from core.base_platform import BasePlatform
from platforms.grok.grok2api_upload import upload_to_grok2api


class GrokSyncService:
    def upload_grok2api(self, account) -> dict:
        ok, msg = upload_to_grok2api(account)
        if ok:
            return BasePlatform._action_success(message=msg)
        return BasePlatform._action_error(msg)
