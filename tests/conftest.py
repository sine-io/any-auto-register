import importlib
import pathlib
import sys
from types import SimpleNamespace

import pytest
from sqlmodel import SQLModel
import sqlmodel.main as sqlmodel_main


ROOT = pathlib.Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))


MODULES_TO_RESET = [
    "main",
    "api.accounts",
    "api.actions",
    "api.tasks",
    "api.worker",
    "api.platforms",
    "core.db",
    "core.config_store",
    "core.proxy_pool",
    "core.registry",
    "core.base_platform",
]


@pytest.fixture
def isolated_modules(tmp_path, monkeypatch):
    db_path = tmp_path / "test.db"
    monkeypatch.setenv("APP_DB_URL", f"sqlite:///{db_path}")

    for name in MODULES_TO_RESET:
        sys.modules.pop(name, None)

    SQLModel.metadata.clear()
    sqlmodel_main.default_registry.dispose()
    importlib.invalidate_caches()

    db = importlib.import_module("core.db")
    registry = importlib.import_module("core.registry")
    base_platform = importlib.import_module("core.base_platform")
    accounts_api = importlib.import_module("api.accounts")
    actions_api = importlib.import_module("api.actions")
    tasks_api = importlib.import_module("api.tasks")
    worker_api = importlib.import_module("api.worker")
    platforms_api = importlib.import_module("api.platforms")

    db.init_db()
    registry._registry.clear()

    return SimpleNamespace(
        db=db,
        registry=registry,
        base_platform=base_platform,
        accounts_api=accounts_api,
        actions_api=actions_api,
        tasks_api=tasks_api,
        worker_api=worker_api,
        platforms_api=platforms_api,
    )
