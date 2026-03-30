import json
import os
import time
import urllib.request

import pytest


def _get_base_url() -> str:
    return os.getenv("SMOKE_BASE_URL", "http://127.0.0.1:18080/api-go").rstrip("/")


def _request_json(path: str, method: str = "GET", body: dict | None = None, timeout: int = 30) -> dict:
    data = None if body is None else json.dumps(body).encode("utf-8")
    request = urllib.request.Request(
        _get_base_url() + path,
        data=data,
        method=method,
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=timeout) as response:
        return json.load(response)


def _request_text(path: str, timeout: int = 30) -> str:
    with urllib.request.urlopen(_get_base_url() + path, timeout=timeout) as response:
        return response.read().decode("utf-8", "ignore")


@pytest.fixture
def api_get_json():
    return lambda path, timeout=30: _request_json(path, timeout=timeout)


@pytest.fixture
def api_post_json():
    return lambda path, body=None, timeout=30: _request_json(path, method="POST", body=body or {}, timeout=timeout)


@pytest.fixture
def api_get_text():
    return lambda path, timeout=30: _request_text(path, timeout=timeout)


@pytest.fixture
def wait_for_task(api_get_json):
    def _wait_for_task(task_id: str, timeout_seconds: int = 60) -> dict:
        deadline = time.time() + timeout_seconds
        last_payload = None
        while time.time() < deadline:
            last_payload = api_get_json(f"/tasks/{task_id}")
            if last_payload.get("status") in {"done", "failed"}:
                return last_payload
            time.sleep(1)
        raise AssertionError(f"task {task_id} did not settle within timeout: {last_payload!r}")

    return _wait_for_task


@pytest.fixture
def wait_for_solver_status(api_get_json):
    def _wait_for_solver_status(
        *,
        allowed_terminal: set[str] | None = None,
        timeout_seconds: int = 60,
    ) -> dict:
        allowed_terminal = allowed_terminal or {"running", "failed", "stopped"}
        deadline = time.time() + timeout_seconds
        last_payload = None
        while time.time() < deadline:
            last_payload = api_get_json("/solver/status", timeout=10)
            status = last_payload.get("status")
            if status in allowed_terminal:
                return last_payload
            time.sleep(2)
        raise AssertionError(f"solver status did not settle within timeout: {last_payload!r}")

    return _wait_for_solver_status
