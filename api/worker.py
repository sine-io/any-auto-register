from fastapi import APIRouter
from pydantic import BaseModel, Field
import requests
from sqlmodel import Session

from api.tasks import RegisterTaskRequest, execute_register_request
from core.config_store import config_store
from core.db import AccountModel, engine
from core.registry import get
from core.base_platform import Account, AccountStatus, RegisterConfig

router = APIRouter(prefix="/worker", tags=["worker"])


class RegisterWorkerRequest(RegisterTaskRequest):
    task_id: str = ""
    callback_base_url: str = ""
    callback_token: str = ""


class CheckAccountWorkerRequest(BaseModel):
    platform: str
    account_id: int


class ExecuteActionWorkerRequest(BaseModel):
    platform: str
    account_id: int
    action_id: str
    params: dict = Field(default_factory=dict)


@router.post("/register")
def register_worker(body: RegisterWorkerRequest):
    logs: list[str] = []
    callback_base = str(body.callback_base_url or "").rstrip("/")
    task_id = str(body.task_id or "").strip()
    callback_token = str(body.callback_token or "").strip()

    def _callback(event: str, payload: dict):
        if not callback_base or not task_id:
            return
        try:
            response = requests.post(
                f"{callback_base}/internal/worker/tasks/{task_id}/{event}",
                json=payload,
                headers={
                    "X-AAR-Internal-Callback-Token": callback_token,
                } if callback_token else {},
                timeout=5,
            )
            response.raise_for_status()
        except Exception as e:
            logs.append(f"[callback:{event}] {e}")

    def _collect(message: str):
        logs.append(message)
        _callback("log", {"message": message})

    try:
        _callback("started", {})
        result = execute_register_request(
            body,
            emit_log=_collect,
            set_progress=lambda current: _callback("progress", {"current": current, "total": body.count}),
        )
    except Exception as e:
        logs.append(f"致命错误: {e}")
        _callback("failed", {
            "success_count": 0,
            "error_count": 1,
            "errors": [str(e)],
            "cashier_urls": [],
            "error": str(e),
        })
        return {
            "ok": False,
            "success_count": 0,
            "error_count": 1,
            "errors": [str(e)],
            "cashier_urls": [],
            "logs": logs,
            "error": str(e),
        }

    if result["ok"]:
        _callback("succeeded", {
            "success_count": result["success_count"],
            "error_count": result["error_count"],
            "errors": result["errors"],
            "cashier_urls": result["cashier_urls"],
            "error": result["error"],
        })
    else:
        _callback("failed", {
            "success_count": result["success_count"],
            "error_count": result["error_count"],
            "errors": result["errors"],
            "cashier_urls": result["cashier_urls"],
            "error": result["error"],
        })

    result["logs"] = logs
    return result


@router.post("/check-account")
def check_account_worker(body: CheckAccountWorkerRequest):
    with Session(engine) as session:
        acc_model = session.get(AccountModel, body.account_id)
        if not acc_model or acc_model.platform != body.platform:
            return {"ok": False, "error": "账号不存在", "valid": False}

    PlatformCls = get(body.platform)
    instance = PlatformCls(config=RegisterConfig(extra=config_store.get_all()))
    account = Account(
        platform=acc_model.platform,
        email=acc_model.email,
        password=acc_model.password,
        user_id=acc_model.user_id,
        token=acc_model.token,
        status=AccountStatus(acc_model.status),
        extra=acc_model.get_extra(),
    )
    valid = instance.check_valid(account)
    return {"ok": True, "valid": valid}


@router.post("/execute-action")
def execute_action_worker(body: ExecuteActionWorkerRequest):
    with Session(engine) as session:
        acc_model = session.get(AccountModel, body.account_id)
        if not acc_model or acc_model.platform != body.platform:
            return {"ok": False, "error": "账号不存在"}

    PlatformCls = get(body.platform)
    instance = PlatformCls(config=RegisterConfig(extra=config_store.get_all()))
    available, reason = instance.get_action_availability(body.action_id)
    if not available:
        return {"ok": False, "error": reason or f"操作 {body.action_id} 当前不可用"}

    account = Account(
        platform=acc_model.platform,
        email=acc_model.email,
        password=acc_model.password,
        user_id=acc_model.user_id,
        token=acc_model.token,
        status=AccountStatus(acc_model.status),
        extra=acc_model.get_extra(),
    )
    try:
        return instance.execute_action(body.action_id, account, body.params)
    except NotImplementedError as e:
        return {"ok": False, "error": str(e)}
    except Exception as e:
        return {"ok": False, "error": str(e)}
