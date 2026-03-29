#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.control-plane.yml}"
KEEP_UP="${SMOKE_KEEP_UP:-0}"
PREFETCH_CAMOUFOX="${PREFETCH_CAMOUFOX:-0}"
GATEWAY_PORT="${GATEWAY_PORT:-18080}"
PYTHON_VNC_PORT="${PYTHON_VNC_PORT:-16080}"
BASE_URL="${SMOKE_BASE_URL:-http://127.0.0.1:${GATEWAY_PORT}/api-go}"
export PREFETCH_CAMOUFOX
export GATEWAY_PORT
export PYTHON_VNC_PORT

wait_for_url() {
  local url="$1"
  local attempts="${2:-60}"
  local delay="${3:-2}"

  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done

  return 1
}

cleanup() {
  if [[ "$KEEP_UP" == "1" ]]; then
    return
  fi
  docker compose -f "$COMPOSE_FILE" down --remove-orphans >/dev/null 2>&1 || true
}

trap cleanup EXIT

docker compose -f "$COMPOSE_FILE" up --build -d

wait_for_url "${BASE_URL}/health" 90 2 || {
  echo "go control plane did not become healthy" >&2
  docker compose -f "$COMPOSE_FILE" logs --tail=200 >&2 || true
  exit 1
}

wait_for_url "${BASE_URL}/solver/status" 90 2 || {
  echo "worker-backed solver status did not become reachable" >&2
  docker compose -f "$COMPOSE_FILE" logs --tail=200 >&2 || true
  exit 1
}

python3 - <<'PY' "$BASE_URL"
import json
import sys
import urllib.request

base = sys.argv[1]
with urllib.request.urlopen(base + "/solver/status", timeout=10) as resp:
    payload = json.load(resp)

if "running" not in payload or "status" not in payload or "reason" not in payload:
    raise SystemExit(f"unexpected solver status payload: {payload!r}")

if payload.get("status") not in {"starting", "running", "failed", "stopped"}:
    raise SystemExit(f"unexpected solver status value: {payload!r}")
PY

curl -fsS "${BASE_URL}/health" >/dev/null
curl -fsS "${BASE_URL}/platforms" >/dev/null
curl -fsS "${BASE_URL}/config" >/dev/null
curl -fsS "${BASE_URL}/solver/status" >/dev/null

python3 - <<'PY' "$BASE_URL"
import json
import sys
import time
import urllib.request

base = sys.argv[1]
req = urllib.request.Request(
    base + "/solver/restart",
    data=b"{}",
    headers={"Content-Type": "application/json"},
)
with urllib.request.urlopen(req, timeout=30):
    pass

deadline = time.time() + 60
last = None
while time.time() < deadline:
    with urllib.request.urlopen(base + "/solver/status", timeout=10) as resp:
        payload = json.load(resp)
    last = payload
    if payload.get("status") == "running":
        break
    if payload.get("status") == "failed":
        raise SystemExit(f"solver restart failed: {payload.get('reason', '')}")
    time.sleep(2)
else:
    raise SystemExit(f"solver restart did not settle: {last!r}")
PY

TASK_ID="$(
  python3 - <<'PY' "$BASE_URL"
import json
import sys
import urllib.request

base = sys.argv[1]
req = urllib.request.Request(
    base + "/tasks/register",
    data=json.dumps({"platform": "dummy", "count": 1}).encode(),
    headers={"Content-Type": "application/json"},
)
with urllib.request.urlopen(req, timeout=30) as resp:
    payload = json.load(resp)
print(payload["task_id"])
PY
)"

python3 - <<'PY' "$BASE_URL" "$TASK_ID"
import json
import sys
import time
import urllib.request

base = sys.argv[1]
task_id = sys.argv[2]

for _ in range(30):
    with urllib.request.urlopen(base + f"/tasks/{task_id}", timeout=30) as resp:
        payload = json.load(resp)
    if payload.get("status") in {"done", "failed"}:
        break
    time.sleep(1)
else:
    raise SystemExit(f"task {task_id} did not reach terminal state")

with urllib.request.urlopen(base + f"/tasks/{task_id}/logs/stream", timeout=30) as resp:
    body = resp.read(512).decode("utf-8", "ignore")

if "data:" not in body:
    raise SystemExit("task log stream did not return SSE data")
PY

echo "control plane smoke passed via ${BASE_URL}"
