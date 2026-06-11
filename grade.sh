#!/usr/bin/env bash
# grade.sh — Stack detection for nanovms
# Detects project stacks from manifest files and fixes "unknown stack" errors.
# Usage: ./grade.sh [--json] [--strict]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}"
JSON_OUTPUT=false
STRICT_MODE=false

usage() {
  echo "Usage: ${0##*/} [--json] [--strict]"
  echo ""
  echo "Options:"
  echo "  --json     Emit machine-readable JSON output"
  echo "  --strict   Exit with error if no stack is detected"
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --json)  JSON_OUTPUT=true; shift ;;
    --strict) STRICT_MODE=true; shift ;;
    -h|--help) usage ;;
    *) echo "Unknown option: $1"; usage ;;
  esac
done

# Detect stacks by looking for manifest files
detect_stacks() {
  local stacks=""

  if [[ -f "${PROJECT_ROOT}/go.mod" ]]; then
    stacks="${stacks}go "
  fi

  if [[ -f "${PROJECT_ROOT}/sdk/rust/Cargo.toml" ]] || [[ -f "${PROJECT_ROOT}/Cargo.toml" ]]; then
    stacks="${stacks}rust "
  fi

  if [[ -f "${PROJECT_ROOT}/package.json" ]]; then
    stacks="${stacks}node "
  fi

  if [[ -f "${PROJECT_ROOT}/package-lock.json" ]]; then
    stacks="${stacks}npm "
  fi

  if [[ -f "${PROJECT_ROOT}/pnpm-lock.yaml" ]]; then
    stacks="${stacks}pnpm "
  fi

  if [[ -f "${PROJECT_ROOT}/yarn.lock" ]]; then
    stacks="${stacks}yarn "
  fi

  if [[ -f "${PROJECT_ROOT}/bun.lockb" ]] || [[ -f "${PROJECT_ROOT}/bun.lock" ]]; then
    stacks="${stacks}bun "
  fi

  if [[ -n "${stacks}" ]]; then
    # trim trailing space
    echo "${stacks% }"
  else
    echo "unknown"
  fi
}

# Pretty-print detected stacks
print_stacks() {
  local stacks="$1"
  echo "=== grade.sh stack detection ==="
  echo "Project root: ${PROJECT_ROOT}"
  echo ""
  echo "Detected stacks: ${stacks}"
  echo ""

  if [[ "${stacks}" == "unknown" ]]; then
    echo "WARNING: No known stack manifest found."
    echo "  Looked for:"
    echo "    - go.mod         (Go)"
    echo "    - Cargo.toml     (Rust)"
    echo "    - package.json   (Node.js)"
    echo ""
    echo "Fix: Ensure one of the above manifests exists in the project root."
    return 1
  fi

  for stack in ${stacks}; do
    case "${stack}" in
      go)
        echo "  [go]     Found go.mod"
        echo "           Module: $(head -1 "${PROJECT_ROOT}/go.mod" | awk '{print $2}')"
        ;;
      rust)
        local cargo_path=""
        if [[ -f "${PROJECT_ROOT}/sdk/rust/Cargo.toml" ]]; then
          cargo_path="${PROJECT_ROOT}/sdk/rust/Cargo.toml"
        else
          cargo_path="${PROJECT_ROOT}/Cargo.toml"
        fi
        echo "  [rust]   Found Cargo.toml"
        echo "           Crate:  $(grep -m1 '^name' "${cargo_path}" | sed 's/.*= *"\(.*\)".*/\1/')"
        ;;
      node|npm|pnpm|yarn|bun)
        echo "  [node]   Found package.json"
        echo "           Package: $(node -p "require('${PROJECT_ROOT}/package.json').name" 2>/dev/null || echo 'n/a')"
        ;;
    esac
  done
  echo ""
  echo "Detection complete."
  return 0
}

# JSON output
print_json() {
  local stacks="$1"
  local go_present=false
  local rust_present=false
  local node_present=false

  if [[ "${stacks}" == "unknown" ]]; then
    cat <<EOF
{
  "project_root": "${PROJECT_ROOT}",
  "stacks": [],
  "status": "unknown",
  "error": "No known stack manifest found"
}
EOF
    return 1
  fi

  [[ "${stacks}" == *"go"* ]] && go_present=true
  [[ "${stacks}" == *"rust"* ]] && rust_present=true
  [[ "${stacks}" == *"node"* ]] && node_present=true

  cat <<EOF
{
  "project_root": "${PROJECT_ROOT}",
  "stacks": [$(echo "${stacks}" | tr ' ' '\n' | sed 's/^/"/' | sed 's/$/"/' | paste -sd, -)],
  "status": "ok",
  "manifests": {
    "go": ${go_present},
    "rust": ${rust_present},
    "node": ${node_present}
  }
}
EOF
  return 0
}

main() {
  local stacks
  stacks="$(detect_stacks)"

  if ${JSON_OUTPUT}; then
    print_json "${stacks}" || exit_code=$?
  else
    print_stacks "${stacks}" || exit_code=$?
  fi

  if ${STRICT_MODE} && [[ "${stacks}" == "unknown" ]]; then
    exit 1
  fi

  exit ${exit_code:-0}
}

main "$@"
