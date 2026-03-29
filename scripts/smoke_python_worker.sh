#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${PORT:-18000}"
HOST="${HOST:-127.0.0.1}"
LOG_PATH="${SMOKE_PYTHON_WORKER_LOG:-/tmp/any-auto-register-python-worker.log}"

wait_for_url() {
  local url="$1"
  local attempts="${2:-30}"
  local delay="${3:-1}"

  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done

  return 1
}

cleanup() {
  if [[ -n "${WORKER_PID:-}" ]] && kill -0 "$WORKER_PID" >/dev/null 2>&1; then
    kill "$WORKER_PID" >/dev/null 2>&1 || true
    wait "$WORKER_PID" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

cd "$ROOT_DIR"
source .venv/bin/activate

HOST="$HOST" PORT="$PORT" APP_RELOAD=0 python main.py >"$LOG_PATH" 2>&1 &
WORKER_PID=$!

wait_for_url "http://${HOST}:${PORT}/api/platforms" 60 1 || {
  echo "python worker failed to start; log follows:" >&2
  tail -n 200 "$LOG_PATH" >&2 || true
  exit 1
}

curl -fsS "http://${HOST}:${PORT}/api/platforms" >/dev/null
curl -fsS "http://${HOST}:${PORT}/api/config" >/dev/null
curl -fsS "http://${HOST}:${PORT}/api/solver/status" >/dev/null

echo "python worker smoke passed on http://${HOST}:${PORT}"
