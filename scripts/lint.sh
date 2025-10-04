#!/usr/bin/env bash
set -euo pipefail

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "golangci-lint is required but not installed. Install from https://golangci-lint.run or via package manager." >&2
  exit 127
fi

exec golangci-lint run --config "$(dirname "$0")/../.golangci.yml" "$@"
