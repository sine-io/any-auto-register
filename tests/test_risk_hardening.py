import pathlib
import importlib
import time
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

from fastapi import BackgroundTasks
from fastapi import HTTPException
from sqlmodel import Session, select


def register_dummy_platform(modules, *, name="dummy", available=True, reason=""):
    BasePlatform = modules.base_platform.BasePlatform
    Account = modules.base_platform.Account
    AccountStatus = modules.base_platform.AccountStatus
    RegisterConfig = modules.base_platform.RegisterConfig

    class DummyPlatform(BasePlatform):
        display_name = "Dummy"
        version = "1.0.0"
        supported_executors = ["protocol", "headed"]

        def __init__(self, config: RegisterConfig = None, mailbox=None):
            super().__init__(config)
            self.mailbox = mailbox

        @classmethod
        def is_available(cls) -> bool:
            return available

        @classmethod
        def get_unavailable_reason(cls) -> str:
            return reason

        def register(self, email: str, password: str = None):
            return Account(
                platform=name,
                email=email or "dummy@example.com",
                password=password or "secret",
                status=AccountStatus.TRIAL,
                trial_end_time=1735689600,
                extra={"cashier_url": "https://example.com/upgrade"},
            )

        def check_valid(self, account):
            return True

        def get_platform_actions(self):
            return [{"id": "sync_external", "label": "同步外部系统", "params": []}]

        def get_action_availability(self, action_id: str):
            if action_id == "sync_external" and not available:
                return False, reason
            return True, ""

        def execute_action(self, action_id: str, account, params: dict):
            if action_id == "sync_external":
                return {"ok": True, "data": {"message": "done"}}
            return super().execute_action(action_id, account, params)

    DummyPlatform.name = name
    modules.registry.register(DummyPlatform)
    return DummyPlatform


def test_save_account_persists_trial_end_time(isolated_modules):
    Account = isolated_modules.base_platform.Account
    AccountStatus = isolated_modules.base_platform.AccountStatus

    record = isolated_modules.db.save_account(
        Account(
            platform="dummy",
            email="trial@example.com",
            password="secret",
            status=AccountStatus.TRIAL,
            trial_end_time=1735689600,
        )
    )

    with Session(isolated_modules.db.engine) as session:
        saved = session.get(isolated_modules.db.AccountModel, record.id)

    assert saved is not None
    assert saved.trial_end_time == 1735689600


def test_register_task_persists_run_state_and_events(isolated_modules):
    register_dummy_platform(isolated_modules)

    req = isolated_modules.tasks_api.RegisterTaskRequest(
        platform="dummy",
        email="dummy@example.com",
        password="secret",
        count=1,
        extra={"mail_provider": "laoudo"},
    )
    background_tasks = BackgroundTasks()

    response = isolated_modules.tasks_api.create_register_task(req, background_tasks)
    task_id = response["task_id"]
    isolated_modules.tasks_api._run_register(task_id, req)

    with Session(isolated_modules.db.engine) as session:
        task = session.get(isolated_modules.db.TaskRunModel, task_id)
        events = session.exec(
            select(isolated_modules.db.TaskEventModel).where(
                isolated_modules.db.TaskEventModel.task_id == task_id
            )
        ).all()

    payload = isolated_modules.tasks_api.get_task(task_id)

    assert task is not None
    assert task.status == "done"
    assert task.success_count == 1
    assert payload["status"] == "done"
    assert payload["success"] == 1
    assert payload["progress"] == "1/1"
    assert any("完成" in event.message for event in events)


def test_worker_register_executes_sync_and_returns_summary(isolated_modules):
    register_dummy_platform(isolated_modules)

    response = isolated_modules.worker_api.register_worker(
        isolated_modules.worker_api.RegisterWorkerRequest(
            platform="dummy",
            email="worker@example.com",
            password="secret",
            count=1,
            extra={"mail_provider": "laoudo"},
        )
    )

    with Session(isolated_modules.db.engine) as session:
        account = session.exec(
            select(isolated_modules.db.AccountModel).where(
                isolated_modules.db.AccountModel.email == "worker@example.com"
            )
        ).first()

    assert response["ok"] is True
    assert response["success_count"] == 1
    assert response["error_count"] == 0
    assert "https://example.com/upgrade" in response["cashier_urls"]
    assert any("完成" in line for line in response["logs"])
    assert account is not None


def test_worker_register_posts_callbacks_when_configured(isolated_modules):
    register_dummy_platform(isolated_modules)
    received = []

    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            length = int(self.headers.get("Content-Length", "0"))
            body = self.rfile.read(length).decode("utf-8")
            received.append((self.path, dict(self.headers), body))
            self.send_response(200)
            self.end_headers()

        def log_message(self, format, *args):
            return

    server = HTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    port = server.server_address[1]

    try:
        response = isolated_modules.worker_api.register_worker(
            isolated_modules.worker_api.RegisterWorkerRequest(
                platform="dummy",
                email="callback@example.com",
                password="secret",
                count=1,
                task_id="task_cb_1",
                callback_base_url=f"http://127.0.0.1:{port}",
                callback_token="secret-token",
                extra={"mail_provider": "laoudo"},
            )
        )
    finally:
        server.shutdown()
        thread.join(timeout=2)

    assert response["ok"] is True
    paths = [item[0] for item in received]
    assert "/internal/worker/tasks/task_cb_1/started" in paths
    assert "/internal/worker/tasks/task_cb_1/log" in paths
    assert "/internal/worker/tasks/task_cb_1/succeeded" in paths
    started_headers = next(headers for path, headers, _ in received if path.endswith("/started"))
    assert started_headers["X-AAR-Internal-Callback-Token"] == "secret-token"


def test_main_reads_cors_allow_origins_from_env(monkeypatch):
    monkeypatch.setenv(
        "APP_CORS_ALLOW_ORIGINS",
        "http://localhost:3000, https://app.example.com",
    )
    main = importlib.import_module("main")

    assert main.get_cors_allow_origins() == [
        "http://localhost:3000",
        "https://app.example.com",
    ]


def test_python_config_masks_secret_values_on_read(isolated_modules):
    isolated_modules.config_store.config_store.set("yescaptcha_key", "secret-key")
    isolated_modules.config_store.config_store.set("mail_provider", "moemail")

    payload = isolated_modules.config_api.get_config()

    assert payload["yescaptcha_key"] == isolated_modules.config_api.MASKED_SECRET_VALUE
    assert payload["mail_provider"] == "moemail"


def test_python_config_ignores_masked_secret_placeholder_on_update(isolated_modules):
    isolated_modules.config_store.config_store.set("yescaptcha_key", "secret-key")

    response = isolated_modules.config_api.update_config(
        isolated_modules.config_api.ConfigUpdate(
            data={
                "yescaptcha_key": isolated_modules.config_api.MASKED_SECRET_VALUE,
                "mail_provider": "duckmail",
            }
        )
    )

    assert response["ok"] is True
    assert "mail_provider" in response["updated"]
    assert "yescaptcha_key" not in response["updated"]
    assert isolated_modules.config_store.config_store.get("yescaptcha_key") == "secret-key"
    assert isolated_modules.config_store.config_store.get("mail_provider") == "duckmail"


def test_worker_check_account_returns_validity(isolated_modules):
    register_dummy_platform(isolated_modules)
    account = isolated_modules.db.AccountModel(
        platform="dummy",
        email="check@example.com",
        password="secret",
    )
    with Session(isolated_modules.db.engine) as session:
        session.add(account)
        session.commit()
        session.refresh(account)
        account_id = account.id

    response = isolated_modules.worker_api.check_account_worker(
        isolated_modules.worker_api.CheckAccountWorkerRequest(
            platform="dummy",
            account_id=account_id,
        )
    )

    assert response["ok"] is True
    assert response["valid"] is True


def test_worker_execute_action_returns_platform_result(isolated_modules):
    register_dummy_platform(isolated_modules)
    account = isolated_modules.db.AccountModel(
        platform="dummy",
        email="action@example.com",
        password="secret",
    )
    with Session(isolated_modules.db.engine) as session:
        session.add(account)
        session.commit()
        session.refresh(account)
        account_id = account.id

    response = isolated_modules.worker_api.execute_action_worker(
        isolated_modules.worker_api.ExecuteActionWorkerRequest(
            platform="dummy",
            account_id=account_id,
            action_id="sync_external",
            params={},
        )
    )

    assert response["ok"] is True
    assert response["data"]["message"] == "done"


def test_dockerfile_prefetches_camoufox_for_solver_runtime():
    dockerfile = pathlib.Path("/root/any-auto-register/Dockerfile").read_text(encoding="utf-8")
    assert "python -m camoufox fetch" in dockerfile


def test_task_event_buffer_flushes_in_batch(isolated_modules):
    req = isolated_modules.tasks_api.RegisterTaskRequest(platform="dummy", count=1)
    isolated_modules.tasks_api._create_task_run("task_buffer", req)

    isolated_modules.tasks_api._append_task_event("task_buffer", "first")
    isolated_modules.tasks_api._append_task_event("task_buffer", "second")

    with Session(isolated_modules.db.engine) as session:
        before_flush = session.exec(
            select(isolated_modules.db.TaskEventModel).where(
                isolated_modules.db.TaskEventModel.task_id == "task_buffer"
            )
        ).all()

    assert before_flush == []

    flushed = isolated_modules.tasks_api._flush_task_event_buffer(force=True)

    with Session(isolated_modules.db.engine) as session:
        after_flush = session.exec(
            select(isolated_modules.db.TaskEventModel).where(
                isolated_modules.db.TaskEventModel.task_id == "task_buffer"
            )
        ).all()

    assert flushed == 2
    assert [event.message for event in after_flush] == ["first", "second"]


def test_task_event_flusher_thread_flushes_without_new_logs(isolated_modules):
    req = isolated_modules.tasks_api.RegisterTaskRequest(platform="dummy", count=1)
    isolated_modules.tasks_api._create_task_run("task_thread_flush", req)

    original_interval = isolated_modules.tasks_api.EVENT_FLUSH_INTERVAL_SECONDS
    original_batch_size = isolated_modules.tasks_api.EVENT_BATCH_SIZE
    isolated_modules.tasks_api.EVENT_FLUSH_INTERVAL_SECONDS = 0.05
    isolated_modules.tasks_api.EVENT_BATCH_SIZE = 100

    try:
        isolated_modules.tasks_api.start_task_event_flusher()
        isolated_modules.tasks_api._append_task_event("task_thread_flush", "buffered")
        time.sleep(0.2)

        with Session(isolated_modules.db.engine) as session:
            events = session.exec(
                select(isolated_modules.db.TaskEventModel).where(
                    isolated_modules.db.TaskEventModel.task_id == "task_thread_flush"
                )
            ).all()
    finally:
        isolated_modules.tasks_api.stop_task_event_flusher()
        isolated_modules.tasks_api.EVENT_FLUSH_INTERVAL_SECONDS = original_interval
        isolated_modules.tasks_api.EVENT_BATCH_SIZE = original_batch_size

    assert [event.message for event in events] == ["buffered"]


def test_platforms_endpoint_exposes_executor_and_availability_metadata(isolated_modules):
    register_dummy_platform(
        isolated_modules,
        name="windows-only",
        available=False,
        reason="Requires Windows desktop environment",
    )

    items = isolated_modules.platforms_api.get_platforms()
    target = next(item for item in items if item["name"] == "windows-only")

    assert target["supported_executors"] == ["protocol", "headed"]
    assert target["available"] is False
    assert target["availability_reason"] == "Requires Windows desktop environment"


def test_action_metadata_and_guard_respect_availability(isolated_modules):
    register_dummy_platform(
        isolated_modules,
        name="guarded",
        available=False,
        reason="Needs external service configuration",
    )

    account = isolated_modules.db.AccountModel(
        platform="guarded",
        email="guarded@example.com",
        password="secret",
    )
    with Session(isolated_modules.db.engine) as session:
        session.add(account)
        session.commit()
        session.refresh(account)
        account_id = account.id

    payload = isolated_modules.actions_api.list_actions("guarded")
    action = payload["actions"][0]

    assert action["available"] is False
    assert action["availability_reason"] == "Needs external service configuration"

    with Session(isolated_modules.db.engine) as session:
        try:
            isolated_modules.actions_api.execute_action(
                "guarded",
                account_id,
                "sync_external",
                isolated_modules.actions_api.ActionRequest(params={}),
                session,
            )
        except HTTPException as exc:
            assert exc.status_code == 400
            assert "Needs external service configuration" in str(exc.detail)
        else:
            raise AssertionError("expected action guard to reject unavailable action")


def test_accounts_list_uses_count_query(isolated_modules):
    class CountResult:
        def all(self):
            raise AssertionError("count query should not materialize all rows")

        def one(self):
            return 3

        def first(self):
            return 3

    class ItemsResult:
        def all(self):
            return [{"id": 1}, {"id": 2}]

    class FakeSession:
        def __init__(self):
            self.statements = []

        def exec(self, stmt):
            sql = str(stmt).lower()
            self.statements.append(sql)
            if "count(" in sql:
                return CountResult()
            return ItemsResult()

    session = FakeSession()
    payload = isolated_modules.accounts_api.list_accounts(page=1, page_size=2, session=session)

    assert any("count(" in stmt for stmt in session.statements)
    assert payload["total"] == 3
    assert len(payload["items"]) == 2


def test_accounts_stats_uses_grouped_queries(isolated_modules):
    class CountResult:
        def one(self):
            return 5

    class RowsResult:
        def __init__(self, rows):
            self._rows = rows

        def all(self):
            return self._rows

    class FakeSession:
        def __init__(self):
            self.statements = []

        def exec(self, stmt):
            sql = str(stmt).lower()
            self.statements.append(sql)
            if "count(" in sql and "group by" not in sql:
                return CountResult()
            if "group by accounts.platform" in sql:
                return RowsResult([("cursor", 2), ("trae", 3)])
            if "group by accounts.status" in sql:
                return RowsResult([("registered", 4), ("trial", 1)])
            raise AssertionError(f"unexpected query: {sql}")

    session = FakeSession()
    payload = isolated_modules.accounts_api.get_stats(session=session)

    assert any("group by accounts.platform" in stmt for stmt in session.statements)
    assert any("group by accounts.status" in stmt for stmt in session.statements)
    assert payload["total"] == 5
    assert payload["by_platform"] == {"cursor": 2, "trae": 3}
    assert payload["by_status"] == {"registered": 4, "trial": 1}


def test_task_logs_list_uses_count_query(isolated_modules, monkeypatch):
    class CountResult:
        def all(self):
            raise AssertionError("count query should not materialize all rows")

        def one(self):
            return 3

        def first(self):
            return 3

    class ItemsResult:
        def all(self):
            return [{"id": 1}, {"id": 2}]

    class FakeSession:
        def __init__(self):
            self.statements = []

        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, tb):
            return False

        def exec(self, stmt):
            sql = str(stmt).lower()
            self.statements.append(sql)
            if "count(" in sql:
                return CountResult()
            return ItemsResult()

    fake_session = FakeSession()
    monkeypatch.setattr(isolated_modules.tasks_api, "Session", lambda engine: fake_session)

    payload = isolated_modules.tasks_api.get_logs(page=1, page_size=2)

    assert any("count(" in stmt for stmt in fake_session.statements)
    assert payload["total"] == 3
    assert len(payload["items"]) == 2


def test_list_tasks_supports_pagination(isolated_modules, monkeypatch):
    class CountResult:
        def one(self):
            return 4

    class ItemsResult:
        def all(self):
            return [
                isolated_modules.db.TaskRunModel(
                    id="task_1",
                    platform="dummy",
                    status="done",
                    progress_current=1,
                    progress_total=1,
                    success_count=1,
                    error_count=0,
                ),
                isolated_modules.db.TaskRunModel(
                    id="task_2",
                    platform="dummy",
                    status="done",
                    progress_current=1,
                    progress_total=1,
                    success_count=1,
                    error_count=0,
                ),
            ]

    class FakeSession:
        def __init__(self):
            self.statements = []

        def __enter__(self):
            return self

        def __exit__(self, exc_type, exc, tb):
            return False

        def exec(self, stmt):
            sql = str(stmt).lower()
            self.statements.append(sql)
            if "count(" in sql:
                return CountResult()
            return ItemsResult()

    fake_session = FakeSession()
    monkeypatch.setattr(isolated_modules.tasks_api, "Session", lambda engine: fake_session)

    payload = isolated_modules.tasks_api.list_tasks(page=2, page_size=2)

    assert payload["total"] == 4
    assert payload["page"] == 2
    assert len(payload["items"]) == 2
    assert any("count(" in stmt for stmt in fake_session.statements)
