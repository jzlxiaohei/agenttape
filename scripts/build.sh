#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C
export LANG=C

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="$ROOT/frontend"
WEB_DIST="$ROOT/internal/web/dist"
STAMP_DIR="$ROOT/.cache/build"
FRONTEND_STAMP="$STAMP_DIR/frontend.sha256"
BINARY="${TRACELAB_BIN:-$ROOT/tracelab}"
FRONTEND_BUILD="${FRONTEND_BUILD:-auto}"
NPM="${NPM:-npm}"
GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOCACHE

sha256() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$@"
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    echo "neither shasum nor sha256sum was found" >&2
    exit 127
  fi
}

frontend_hash() {
  (
    cd "$ROOT"
    find frontend \
      -type d \( -name node_modules -o -name dist \) -prune \
      -o -type f -print |
      LC_ALL=C sort |
      while IFS= read -r file; do
        sha256 "$file"
      done |
      sha256 |
      awk '{print $1}'
  )
}

run_frontend_build=false
current_frontend_hash=""

case "$FRONTEND_BUILD" in
  auto)
    mkdir -p "$STAMP_DIR"
    current_frontend_hash="$(frontend_hash)"
    previous_frontend_hash=""
    if [[ -f "$FRONTEND_STAMP" ]]; then
      previous_frontend_hash="$(cat "$FRONTEND_STAMP")"
    fi

    if [[ "$current_frontend_hash" != "$previous_frontend_hash" || ! -s "$WEB_DIST/index.html" ]]; then
      run_frontend_build=true
    fi
    ;;
  always)
    mkdir -p "$STAMP_DIR"
    current_frontend_hash="$(frontend_hash)"
    run_frontend_build=true
    ;;
  skip)
    ;;
  *)
    echo "unknown FRONTEND_BUILD=$FRONTEND_BUILD; expected auto, always, or skip" >&2
    exit 2
    ;;
esac

if [[ "$run_frontend_build" == true ]]; then
  if ! command -v "$NPM" >/dev/null 2>&1; then
    echo "npm was not found; install Node.js/npm, set NPM=/path/to/npm, or use FRONTEND_BUILD=skip if the embedded viewer is already fresh" >&2
    exit 127
  fi

  if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
    echo "frontend/node_modules is missing; run: (cd frontend && npm ci)" >&2
  fi

  echo "==> Building frontend"
  (
    cd "$FRONTEND_DIR"
    "$NPM" run build
  )

  if [[ -n "$current_frontend_hash" ]]; then
    printf '%s\n' "$current_frontend_hash" > "$FRONTEND_STAMP"
  fi
else
  echo "==> Frontend unchanged; skipping npm run build"
fi

echo "==> Building Go binary: $BINARY"
(
  cd "$ROOT"
  go build "$@" -o "$BINARY" ./cmd/tracelab
)

echo "==> Done"
