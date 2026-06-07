#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
BACKEND_DIR="$ROOT_DIR"
EMBED_DIR="$BACKEND_DIR/web/dist"
OUTPUT_DIR="$ROOT_DIR/build"
OUTPUT_BIN="$OUTPUT_DIR/classical-chinese-quiz"

if [[ -x "$ROOT_DIR/.tools/go/bin/go" ]]; then
  GO_BIN="$ROOT_DIR/.tools/go/bin/go"
else
  GO_BIN="${GO_BIN:-go}"
fi

mkdir -p "$EMBED_DIR" "$OUTPUT_DIR"
find "$EMBED_DIR" -mindepth 1 ! -name 'placeholder.txt' -exec rm -rf {} +

if [[ "${SKIP_FRONTEND_BUILD:-0}" != "1" ]]; then
  echo "==> Building frontend bundle"
  (
    cd "$ROOT_DIR"
    npm ci
    npm run build
  )
fi

echo "==> Syncing frontend bundle into embedded asset directory"
cp -R "$FRONTEND_DIR/dist"/. "$EMBED_DIR"/

echo "==> Building embedded backend binary"
(
  cd "$BACKEND_DIR"
  "$GO_BIN" build -o "$OUTPUT_BIN" ./cmd/server
)

echo "Embedded binary written to $OUTPUT_BIN"
