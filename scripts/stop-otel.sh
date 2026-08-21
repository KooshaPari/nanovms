#!/usr/bin/env bash
# stop-otel.sh — Stop the OpenTelemetry Collector stack (nanovms)
# Supports both the standard and production compose files.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OTEL_DIR="${SCRIPT_DIR}/../otel"

COMPOSE_FILE="${OTEL_DIR}/docker-compose.yml"
if [[ "${1:-}" == "--production" ]]; then
    COMPOSE_FILE="${OTEL_DIR}/docker-compose.production.yml"
fi

if [[ ! -f "${COMPOSE_FILE}" ]]; then
    echo "ERROR: compose file not found at ${COMPOSE_FILE}" >&2
    exit 1
fi

echo "[nanovms-otel] Stopping collector stack …"
docker compose -f "${COMPOSE_FILE}" down

echo "[nanovms-otel] Stack stopped."
