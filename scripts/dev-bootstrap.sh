#!/usr/bin/env bash
# nanovms dev bootstrap: check Go toolchain and build.
set -euo pipefail
echo "[dev-bootstrap] checking go..."
command -v go >/dev/null 2>&1 || { echo "go not found"; exit 1; }
GO_VERSION=$(go version | awk '{print $3}' | tr -d 'go')
REQUIRED="1.21.0"
if ! printf '%s\n%s\n' "$REQUIRED" "$GO_VERSION" | sort -V -C; then
    echo "go $GO_VERSION < $REQUIRED"; exit 1
fi
echo "[dev-bootstrap] go $GO_VERSION OK"
go build ./...
echo "[dev-bootstrap] OK"
