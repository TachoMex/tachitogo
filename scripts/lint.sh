#!/bin/sh
set -eu
set -o pipefail 2>/dev/null || true

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint is required but not installed. Install from https://golangci-lint.run or via package manager." >&2
  exit 127
fi

SCRIPT_DIR=$(dirname "$0")
exec golangci-lint run --config "$SCRIPT_DIR/../.golangci.yml" "$@"
