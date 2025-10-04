#!/bin/sh
set -eu
set -o pipefail 2>/dev/null || true

profile=${1:-coverage.out}
mode=${2:-atomic}

go test ./... -covermode="$mode" -coverprofile="$profile"
go tool cover -func="$profile"
