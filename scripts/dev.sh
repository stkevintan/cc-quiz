#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -x "$ROOT_DIR/.tools/go/bin/go" ]]; then
  GO_BIN="$ROOT_DIR/.tools/go/bin/go"
else
  GO_BIN="${GO_BIN:-go}"
fi

cleanup() {
  local exit_code=$?
  if [[ -n "${BACKEND_PID:-}" ]] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
  fi
  if [[ -n "${FRONTEND_PID:-}" ]] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
    kill "$FRONTEND_PID" 2>/dev/null || true
  fi
  wait "${BACKEND_PID:-}" "${FRONTEND_PID:-}" 2>/dev/null || true
  exit "$exit_code"
}

trap cleanup EXIT INT TERM

echo "==> Starting backend server on http://localhost:8080"
(
  cd "$ROOT_DIR"
  "$GO_BIN" run ./cmd/server
) &
BACKEND_PID=$!

echo "==> Starting frontend dev server on http://localhost:5173"
(
  cd "$ROOT_DIR"
  npm run dev
) &
FRONTEND_PID=$!

wait -n "$BACKEND_PID" "$FRONTEND_PID"
