#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/go-control-plane"

export AAR_SERVER_PORT="${AAR_SERVER_PORT:-8080}"
export AAR_SERVER_PUBLIC_BASE_URL="${AAR_SERVER_PUBLIC_BASE_URL:-http://127.0.0.1:${AAR_SERVER_PORT}}"
export AAR_WORKER_BASE_URL="${AAR_WORKER_BASE_URL:-http://127.0.0.1:8000}"
export AAR_DATABASE_URL="${AAR_DATABASE_URL:-../account_manager.db}"

exec go run ./cmd/server server
