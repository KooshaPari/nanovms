#!/usr/bin/env bash
# ============================================================================
# otel-collector-teardown.sh — Stop and remove the NanoVMS OTel stack
# ============================================================================
#
# Stops all services, removes containers, and optionally prunes volumes.
#
# Usage:
#   ./otel-collector-teardown.sh [OPTIONS]
#
# Options:
#   --env FILE        Path to .env file (default: .env.production)
#   --volumes         Also remove named volumes (data loss!)
#   --prune           Remove images after stopping
#   --dry-run         Print commands without executing
#   -h, --help        Show this help message
# ============================================================================

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly OTEL_DIR="${PROJECT_ROOT}/otel"
readonly COMPOSE_FILE="docker-compose.production.yml"
readonly DEFAULT_ENV_FILE="${SCRIPT_DIR}/.env.production"

# Color helpers
if [ -t 1 ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    BLUE='\033[0;34m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

log_info()    { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step()    { echo -e "${BLUE}[STEP]${NC}  $*"; }

# ---------------------------------------------------------------------------
# CLI arguments
# ---------------------------------------------------------------------------
ENV_FILE="${DEFAULT_ENV_FILE}"
REMOVE_VOLUMES=false
PRUNE_IMAGES=false
DRY_RUN=false

usage() {
    sed -n '/^# Usage:/,/^# ====/p' "${BASH_SOURCE[0]}" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

while [ $# -gt 0 ]; do
    case "$1" in
        --env)       ENV_FILE="$2"; shift 2 ;;
        --volumes)   REMOVE_VOLUMES=true; shift ;;
        --prune)     PRUNE_IMAGES=true; shift ;;
        --dry-run)   DRY_RUN=true; shift ;;
        -h|--help)   usage ;;
        *)           log_error "Unknown option: $1"; usage ;;
    esac
done

# ---------------------------------------------------------------------------
# Load env
# ---------------------------------------------------------------------------
if [ -f "$ENV_FILE" ]; then
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
fi

# ---------------------------------------------------------------------------
# Confirmation
# ---------------------------------------------------------------------------
if [ "$REMOVE_VOLUMES" = true ]; then
    echo ""
    log_warn "WARNING: This will remove all persistent data volumes!"
    log_warn "  - Prometheus metrics history"
    log_warn "  - Loki log data"
    log_warn "  - Jaeger trace storage"
    log_warn "  - Grafana dashboards and settings"
    echo ""
    read -r -p "Type 'yes' to confirm volume removal: " confirm
    if [ "$confirm" != "yes" ]; then
        log_info "Aborted"
        exit 0
    fi
fi

# ---------------------------------------------------------------------------
# Stop stack
# ---------------------------------------------------------------------------
log_step "Stopping NanoVMS OTel Collector stack..."

STOP_CMD="docker compose -f ${OTEL_DIR}/${COMPOSE_FILE}"

if [ -n "${ENV_FILE:-}" ] && [ -f "$ENV_FILE" ]; then
    STOP_CMD="${STOP_CMD} --env-file ${ENV_FILE}"
fi

if [ "$DRY_RUN" = true ]; then
    log_info "[DRY RUN] Would run: ${STOP_CMD} down --remove-orphans"
else
    $STOP_CMD down --remove-orphans
    log_info "Services stopped"
fi

# ---------------------------------------------------------------------------
# Remove volumes
# ---------------------------------------------------------------------------
if [ "$REMOVE_VOLUMES" = true ]; then
    log_step "Removing named volumes..."
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would remove nanovms-* volumes"
    else
        $STOP_CMD down -v --remove-orphans 2>/dev/null || true

        # Also remove any leftover named volumes
        for vol in nanovms-prometheus-data nanovms-loki-data nanovms-grafana-data nanovms-jaeger-data nanovms-collector-logs; do
            docker volume rm "$vol" 2>/dev/null || true
        done
        log_info "Volumes removed"
    fi
fi

# ---------------------------------------------------------------------------
# Prune images
# ---------------------------------------------------------------------------
if [ "$PRUNE_IMAGES" = true ]; then
    log_step "Pruning NanoVMS OTel images..."
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would prune nanovms-otel images"
    else
        docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '(otel|jaeger|prometheus|loki|grafana)' | while read -r img; do
            docker rmi "$img" 2>/dev/null || true
        done
        log_info "Images pruned"
    fi
fi

# ---------------------------------------------------------------------------
# Clean up TLS certs (optional)
# ---------------------------------------------------------------------------
CERTS_DIR="${OTEL_DIR}/certs"
if [ -d "$CERTS_DIR" ]; then
    log_step "Cleaning up TLS certificates..."
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would remove ${CERTS_DIR}"
    else
        rm -rf "${CERTS_DIR}"
        log_info "TLS certificates removed"
    fi
fi

echo ""
log_info "NanoVMS OTel Collector stack has been torn down"
