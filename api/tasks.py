from fastapi import APIRouter, BackgroundTasks, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from sqlmodel import Session, select, func
from typing import Callable, Optional
from datetime import datetime, timezone
from core.db import TaskEventModel, TaskLog, TaskRunModel, engine
import time, json, asyncio, threading, logging

router = APIRouter(prefix="/tasks", tags=["tasks"])
logger = logging.getLogger(__name__)

EVENT_BATCH_SIZE = 10
EVENT_FLUSH_INTERVAL_SECONDS = 0.5
_task_event_buffer: list[dict] = []
_task_event_buffer_lock = threading.Lock()
_task_event_flusher_thread: threading.Thread | None = None
_task_event_flusher_stop = threading.Event()

class RegisterTaskRequest(BaseModel):
    platform: str
    email: Optional[str] = None
    password: Optional[str] = None
    count: int = 1
    concurrency: int = 1
    register_delay_seconds: float = 0
    proxy: Optional[str] = None
    executor_type: str = "protocol"
    captcha_solver: str = "yescaptcha"
    extra: dict = Field(default_factory=dict)


class TaskLogBatchDeleteRequest(BaseModel):
    ids: list[int]


def _utcnow():
    return datetime.now(timezone.utc)


def _loads_json(raw: str, default):
    try:
        return json.loads(raw or "")
    except Exception:
        return default


def _dump_json(data) -> str:
    return json.dumps(data, ensure_ascii=False)


def _task_payload(task: TaskRunModel) -> dict:
    errors = _loads_json(task.errors_json, [])
    cashier_urls = _loads_json(task.cashier_urls_json, [])
    return {
        "id": task.id,
        "status": task.status,
        "progress": f"{task.progress_current}/{task.progress_total}",
        "success": task.success_count,
        "errors": errors,
        "error": task.error_summary,
        "cashier_urls": cashier_urls,
        "created_at": task.created_at,
        "updated_at": task.updated_at,
        "platform": task.platform,
    }


def _create_task_run(task_id: str, req: RegisterTaskRequest) -> None:
    with Session(engine) as s:
        run = TaskRunModel(
            id=task_id,
            platform=req.platform,
            status="pending",
            progress_current=0,
            progress_total=req.count,
            request_json=_dump_json(req.model_dump()),
        )
        s.add(run)
        s.commit()


def _get_task_run(task_id: str) -> Optional[TaskRunModel]:
    with Session(engine) as s:
        return s.get(TaskRunModel, task_id)


def _update_task_run(task_id: str, **fields) -> Optional[TaskRunModel]:
    with Session(engine) as s:
        run = s.get(TaskRunModel, task_id)
        if not run:
            return None
        for key, value in fields.items():
            setattr(run, key, value)
        run.updated_at = _utcnow()
        s.add(run)
        s.commit()
        s.refresh(run)
        return run


def _append_task_event(task_id: str, message: str, level: str = "info") -> None:
    queued_at = time.monotonic()
    should_flush = False
    with _task_event_buffer_lock:
        _task_event_buffer.append(
            {
                "task_id": task_id,
                "level": level,
                "message": message,
                "created_at": _utcnow(),
                "queued_at": queued_at,
            }
        )
        oldest = _task_event_buffer[0]["queued_at"] if _task_event_buffer else queued_at
        should_flush = (
            len(_task_event_buffer) >= EVENT_BATCH_SIZE
            or (queued_at - oldest) >= EVENT_FLUSH_INTERVAL_SECONDS
        )
    if should_flush:
        _flush_task_event_buffer()


def _flush_task_event_buffer(force: bool = False) -> int:
    batch: list[dict] = []
    now = time.monotonic()
    with _task_event_buffer_lock:
        if not _task_event_buffer:
            return 0
        oldest = _task_event_buffer[0]["queued_at"]
        if not force and len(_task_event_buffer) < EVENT_BATCH_SIZE and (now - oldest) < EVENT_FLUSH_INTERVAL_SECONDS:
            return 0
        batch = list(_task_event_buffer)
        _task_event_buffer.clear()

    with Session(engine) as s:
        s.add_all(
            [
                TaskEventModel(
                    task_id=item["task_id"],
                    level=item["level"],
                    message=item["message"],
                    created_at=item["created_at"],
                )
                for item in batch
            ]
        )
        s.commit()
    return len(batch)


def _task_event_flusher_loop() -> None:
    while not _task_event_flusher_stop.wait(EVENT_FLUSH_INTERVAL_SECONDS):
        try:
            _flush_task_event_buffer()
        except Exception as e:
            logger.exception("任务日志批量刷新失败: %s", e)


def start_task_event_flusher() -> None:
    global _task_event_flusher_thread
    if _task_event_flusher_thread and _task_event_flusher_thread.is_alive():
        return
    _task_event_flusher_stop.clear()
    _task_event_flusher_thread = threading.Thread(
        target=_task_event_flusher_loop,
        daemon=True,
        name="task-event-flusher",
    )
    _task_event_flusher_thread.start()


def stop_task_event_flusher() -> None:
    global _task_event_flusher_thread
    _task_event_flusher_stop.set()
    if _task_event_flusher_thread and _task_event_flusher_thread.is_alive():
        _task_event_flusher_thread.join(timeout=2)
    _task_event_flusher_thread = None
    _flush_task_event_buffer(force=True)


def _set_task_progress(task_id: str, current: int) -> None:
    with Session(engine) as s:
        run = s.get(TaskRunModel, task_id)
        if not run:
            return
        run.progress_current = max(run.progress_current, current)
        run.updated_at = _utcnow()
        s.add(run)
        s.commit()


def _log(task_id: str, msg: str):
    """向任务追加一条日志"""
    ts = time.strftime("%H:%M:%S")
    entry = f"[{ts}] {msg}"
    _append_task_event(task_id, entry)
    print(entry)


def _save_task_log(platform: str, email: str, status: str,
                   error: str = "", detail: dict = None):
    """Write a TaskLog record to the database."""
    with Session(engine) as s:
        log = TaskLog(
            platform=platform,
            email=email,
            status=status,
            error=error,
            detail_json=json.dumps(detail or {}, ensure_ascii=False),
        )
        s.add(log)
        s.commit()


def _auto_upload_integrations(emit_log: Callable[[str], None], account):
    """注册成功后自动导入外部系统。"""
    try:
        from services.external_sync import sync_account

        for result in sync_account(account):
            name = result.get("name", "Auto Upload")
            ok = bool(result.get("ok"))
            msg = result.get("msg", "")
            emit_log(f"  [{name}] {'✓ ' + msg if ok else '✗ ' + msg}")
    except Exception as e:
        emit_log(f"  [Auto Upload] 自动导入异常: {e}")


def execute_register_request(
    req: RegisterTaskRequest,
    emit_log: Callable[[str], None],
    set_progress: Optional[Callable[[int], None]] = None,
) -> dict:
    from core.registry import get
    from core.base_platform import RegisterConfig
    from core.db import save_account
    from core.base_mailbox import create_mailbox

    success = 0
    errors = []
    cashier_urls = []
    start_gate_lock = threading.Lock()
    next_start_time = time.time()

    PlatformCls = get(req.platform)
    if not PlatformCls.is_available():
        raise RuntimeError(PlatformCls.get_unavailable_reason() or f"{req.platform} 当前不可用")

    def _build_mailbox(proxy: Optional[str]):
        return create_mailbox(
            provider=req.extra.get("mail_provider", "laoudo"),
            extra=req.extra,
            proxy=proxy,
        )

    def _emit(message: str):
        emit_log(message)

    def _advance(current: int):
        if callable(set_progress):
            set_progress(current)

    def _do_one(i: int):
        nonlocal next_start_time
        try:
            from core.proxy_pool import proxy_pool

            _proxy = req.proxy
            if not _proxy:
                _proxy = proxy_pool.get_next()
            if req.register_delay_seconds > 0:
                with start_gate_lock:
                    now = time.time()
                    wait_seconds = max(0.0, next_start_time - now)
                    if wait_seconds > 0:
                        _emit(f"第 {i+1} 个账号启动前延迟 {wait_seconds:g} 秒")
                        time.sleep(wait_seconds)
                    next_start_time = time.time() + req.register_delay_seconds
            _config = RegisterConfig(
                executor_type=req.executor_type,
                captcha_solver=req.captcha_solver,
                proxy=_proxy,
                extra=req.extra,
            )
            _mailbox = _build_mailbox(_proxy)
            _platform = PlatformCls(config=_config, mailbox=_mailbox)
            _platform._log_fn = _emit
            if getattr(_platform, "mailbox", None) is not None:
                _platform.mailbox._log_fn = _platform._log_fn
            _advance(i + 1)
            _emit(f"开始注册第 {i+1}/{req.count} 个账号")
            if _proxy:
                _emit(f"使用代理: {_proxy}")
            account = _platform.register(
                email=req.email or None,
                password=req.password,
            )
            save_account(account)
            if _proxy:
                proxy_pool.report_success(_proxy)
            _emit(f"✓ 注册成功: {account.email}")
            _save_task_log(req.platform, account.email, "success")
            _auto_upload_integrations(_emit, account)
            cashier_url = (account.extra or {}).get("cashier_url", "")
            if cashier_url:
                _emit(f"  [升级链接] {cashier_url}")
            return {"ok": True, "cashier_url": cashier_url}
        except Exception as e:
            if _proxy:
                proxy_pool.report_fail(_proxy)
            _emit(f"✗ 注册失败: {e}")
            _save_task_log(req.platform, req.email or "", "failed", error=str(e))
            return {"ok": False, "error": str(e)}

    from concurrent.futures import ThreadPoolExecutor, as_completed
    max_workers = min(req.concurrency, req.count, 5)
    with ThreadPoolExecutor(max_workers=max_workers) as pool:
        futures = [pool.submit(_do_one, i) for i in range(req.count)]
        for f in as_completed(futures):
            try:
                result = f.result()
            except Exception as e:
                _emit(f"✗ 任务线程异常: {e}")
                errors.append(str(e))
                continue
            if result.get("ok"):
                success += 1
                cashier_url = result.get("cashier_url")
                if cashier_url:
                    cashier_urls.append(cashier_url)
            else:
                errors.append(result.get("error", "未知错误"))

    _emit(f"完成: 成功 {success} 个, 失败 {len(errors)} 个")
    return {
        "ok": len(errors) == 0,
        "success_count": success,
        "error_count": len(errors),
        "errors": errors,
        "cashier_urls": cashier_urls,
        "error": "; ".join(errors),
    }


def _run_register(task_id: str, req: RegisterTaskRequest):
    _update_task_run(task_id, status="running")
    try:
        result = execute_register_request(
            req,
            emit_log=lambda message: _log(task_id, message),
            set_progress=lambda current: _set_task_progress(task_id, current),
        )
    except Exception as e:
        _log(task_id, f"致命错误: {e}")
        _flush_task_event_buffer(force=True)
        _update_task_run(
            task_id,
            status="failed",
            error_count=1,
            error_summary=str(e),
            errors_json=_dump_json([str(e)]),
        )
        return

    _flush_task_event_buffer(force=True)
    _update_task_run(
        task_id,
        status="done" if result["ok"] else "failed",
        progress_current=req.count,
        success_count=result["success_count"],
        error_count=result["error_count"],
        error_summary=result["error"],
        errors_json=_dump_json(result["errors"]),
        cashier_urls_json=_dump_json(result["cashier_urls"]),
    )


@router.post("/register")
def create_register_task(
    req: RegisterTaskRequest,
    background_tasks: BackgroundTasks,
):
    from core.registry import get

    platform_cls = get(req.platform)
    if not platform_cls.is_available():
        raise HTTPException(400, platform_cls.get_unavailable_reason() or "当前平台不可用")

    task_id = f"task_{int(time.time()*1000)}"
    _create_task_run(task_id, req)
    background_tasks.add_task(_run_register, task_id, req)
    return {"task_id": task_id}


@router.get("/logs")
def get_logs(platform: str = None, page: int = 1, page_size: int = 50):
    with Session(engine) as s:
        q = select(TaskLog)
        count_q = select(func.count()).select_from(TaskLog)
        if platform:
            q = q.where(TaskLog.platform == platform)
            count_q = count_q.where(TaskLog.platform == platform)
        q = q.order_by(TaskLog.id.desc())
        total = s.exec(count_q).one()
        items = s.exec(q.offset((page - 1) * page_size).limit(page_size)).all()
    return {"total": total, "items": items}


@router.post("/logs/batch-delete")
def batch_delete_logs(body: TaskLogBatchDeleteRequest):
    if not body.ids:
        raise HTTPException(400, "任务历史 ID 列表不能为空")

    unique_ids = list(dict.fromkeys(body.ids))
    if len(unique_ids) > 1000:
        raise HTTPException(400, "单次最多删除 1000 条任务历史")

    with Session(engine) as s:
        try:
            logs = s.exec(select(TaskLog).where(TaskLog.id.in_(unique_ids))).all()
            found_ids = {log.id for log in logs if log.id is not None}

            for log in logs:
                s.delete(log)

            s.commit()
            deleted_count = len(found_ids)
            not_found_ids = [log_id for log_id in unique_ids if log_id not in found_ids]
            logger.info("批量删除任务历史成功: %s 条", deleted_count)

            return {
                "deleted": deleted_count,
                "not_found": not_found_ids,
                "total_requested": len(unique_ids),
            }
        except Exception as e:
            s.rollback()
            logger.exception("批量删除任务历史失败")
            raise HTTPException(500, f"批量删除任务历史失败: {str(e)}")


@router.get("/{task_id}/logs/stream")
async def stream_logs(task_id: str, since: int = 0):
    """SSE 实时日志流"""
    if not _get_task_run(task_id):
        raise HTTPException(404, "任务不存在")

    async def event_generator():
        last_event_id = since
        while True:
            _flush_task_event_buffer()
            with Session(engine) as s:
                task = s.get(TaskRunModel, task_id)
                if not task:
                    break
                events = s.exec(
                    select(TaskEventModel)
                    .where(TaskEventModel.task_id == task_id)
                    .where(TaskEventModel.id > last_event_id)
                    .order_by(TaskEventModel.id)
                ).all()
                status = task.status
            if status in ("done", "failed"):
                _flush_task_event_buffer(force=True)
                with Session(engine) as s:
                    events = s.exec(
                        select(TaskEventModel)
                        .where(TaskEventModel.task_id == task_id)
                        .where(TaskEventModel.id > last_event_id)
                        .order_by(TaskEventModel.id)
                    ).all()
            for event in events:
                last_event_id = event.id or last_event_id
                yield f"data: {json.dumps({'line': event.message, 'event_id': last_event_id})}\n\n"
            if status in ("done", "failed"):
                yield f"data: {json.dumps({'done': True, 'status': status})}\n\n"
                break
            await asyncio.sleep(0.5)

    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )


@router.get("/{task_id}")
def get_task(task_id: str):
    task = _get_task_run(task_id)
    if not task:
        raise HTTPException(404, "任务不存在")
    return _task_payload(task)


@router.get("")
def list_tasks(page: int = 1, page_size: int = 50):
    with Session(engine) as s:
        total = s.exec(select(func.count()).select_from(TaskRunModel)).one()
        tasks = s.exec(
            select(TaskRunModel)
            .order_by(TaskRunModel.created_at.desc())
            .offset((page - 1) * page_size)
            .limit(page_size)
        ).all()
    return {"total": total, "page": page, "items": [_task_payload(task) for task in tasks]}
