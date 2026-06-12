#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HEADER="$ROOT/include/nvms_syscalls.h"
OUTPUT="$ROOT/src/generated/nvms_syscalls_bindings.rs"

bindgen "$HEADER" \
  --allowlist-function 'nvms_(focus|exit|exec)' \
  --allowlist-type 'nvms_exec_args' \
  --output "$OUTPUT"
