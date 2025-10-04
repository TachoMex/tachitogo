#!/usr/bin/env bash
set -euo pipefail

profile=${1:-coverage.out}
mode=${2:-atomic}

go test ./... -covermode="$mode" -coverprofile="$profile"
go tool cover -func="$profile"
