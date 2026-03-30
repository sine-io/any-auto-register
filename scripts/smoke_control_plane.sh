#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/docker-compose.control-plane.yml}"
KEEP_UP="${SMOKE_KEEP_UP:-0}"
PREFETCH_CAMOUFOX="${PREFETCH_CAMOUFOX:-0}"
GATEWAY_PORT="${GATEWAY_PORT:-18080}"
PYTHON_VNC_PORT="${PYTHON_VNC_PORT:-16080}"
BASE_URL="${SMOKE_BASE_URL:-http://127.0.0.1:${GATEWAY_PORT}/api-go}"
PYTHON_BIN="${PYTHON_BIN:-python3}"
export PREFETCH_CAMOUFOX
export GATEWAY_PORT
export PYTHON_VNC_PORT
export SMOKE_BASE_URL="$BASE_URL"

if [[ -x "$ROOT_DIR/.venv/bin/python" ]]; then
  PYTHON_BIN="$ROOT_DIR/.venv/bin/python"
fi

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

docker compose -f "$COMPOSE_FILE" down --remove-orphans >/dev/null 2>&1 || true
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

"$PYTHON_BIN" -m pytest tests/e2e -q

echo "control plane smoke passed via ${BASE_URL}"
