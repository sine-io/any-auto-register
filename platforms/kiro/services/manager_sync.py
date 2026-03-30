from __future__ import annotations

from platforms.kiro.account_manager_upload import upload_to_kiro_manager


class KiroManagerSyncService:
    def upload(self, account) -> dict:
        ok, msg = upload_to_kiro_manager(account)
        if ok:
            return {"ok": True, "data": {"message": msg}}
        return {"ok": False, "error": msg}
