"""Turnstile Solver 进程管理 - 后端启动时自动拉起"""
import subprocess
import sys
import os
import time
import threading
import requests

SOLVER_PORT = 8889
SOLVER_URL = f"http://localhost:{SOLVER_PORT}"
_proc: subprocess.Popen = None
_log_file = None
_lock = threading.Lock()
_state = "stopped"
_reason = ""


def _set_state(state: str, reason: str = "") -> None:
    global _state, _reason
    _state = state
    _reason = reason


def is_running() -> bool:
    try:
        r = requests.get(f"{SOLVER_URL}/", timeout=2)
        return r.status_code < 500
    except Exception:
        return False


def get_status() -> dict:
    with _lock:
        state = _state
        reason = _reason

    running = is_running()
    if running and state != "running":
        return {"running": True, "status": "running", "reason": ""}
    if state == "running" and not running:
        return {"running": False, "status": "failed", "reason": reason or "solver health check failed"}
    return {"running": running, "status": state, "reason": reason}


def start():
    global _proc, _log_file
    with _lock:
        if is_running():
            _set_state("running")
            print("[Solver] 已在运行")
            return
        _set_state("starting")
        solver_script = os.path.join(
            os.path.dirname(__file__), "turnstile_solver", "start.py"
        )
        log_path = os.path.join(
            os.path.dirname(__file__), "turnstile_solver", "solver.log"
        )
        _log_file = open(log_path, "a", encoding="utf-8")
        _proc = subprocess.Popen(
            [
                sys.executable,
                "-u",
                solver_script,
                "--browser_type",
                "camoufox",
                "--port",
                str(SOLVER_PORT),
            ],
            stdout=_log_file,
            stderr=subprocess.STDOUT,
        )
        # 等待服务就绪（最多30s）
        for _ in range(30):
            time.sleep(1)
            if is_running():
                _set_state("running")
                print(f"[Solver] 已启动 PID={_proc.pid}")
                return
            if _proc.poll() is not None:
                _set_state("failed", f"exit code {_proc.returncode}")
                print(f"[Solver] 启动失败，退出码={_proc.returncode}，日志: {log_path}")
                _proc = None
                if _log_file:
                    _log_file.close()
                    _log_file = None
                return
        _set_state("failed", "startup timeout")
        print(f"[Solver] 启动超时，日志: {log_path}")


def stop():
    global _proc, _log_file
    with _lock:
        if _proc and _proc.poll() is None:
            _proc.terminate()
            try:
                _proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                _proc.kill()
                _proc.wait(timeout=5)
            print("[Solver] 已停止")
        _proc = None
        if _log_file:
            _log_file.close()
            _log_file = None
        _set_state("stopped")


def start_async():
    """在后台线程启动，不阻塞主进程"""
    with _lock:
        if not is_running():
            _set_state("starting")
    t = threading.Thread(target=start, daemon=True)
    t.start()
