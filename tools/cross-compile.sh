#!/usr/bin/env bash
# nanovms cross-compile (L128).
# Runs cargo check against multiple targets and reports status.

set -uo pipefail
TARGETS="${NVMS_TARGETS:-aarch64-apple-darwin x86_64-unknown-linux-gnu wasm32-unknown-unknown}"
RESULTS=0
for tgt in $TARGETS; do
  echo "[cross-compile] checking target: $tgt"
  if cargo check --target "$tgt" --quiet 2>/dev/null; then
    echo "[cross-compile] $tgt: OK"
  else
    echo "[cross-compile] $tgt: FAIL"
    RESULTS=$((RESULTS + 1))
  fi
done
exit $RESULTS
