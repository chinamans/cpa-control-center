#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PLATFORM="${1:-darwin/universal}"
WAILS_BIN="${WAILS_BIN:-$(go env GOPATH)/bin/wails}"

if [[ ! -x "$WAILS_BIN" ]]; then
  go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
fi

"$WAILS_BIN" build -clean -platform "$PLATFORM"
