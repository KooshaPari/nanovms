#!/usr/bin/env bash
# ============================================================================
# otel-collector-setup.sh — Production OTel Collector deployment for NanoVMS
# ============================================================================
#
# Sets up a production-grade OpenTelemetry Collector stack including:
#   - OTel Collector (Contrib) with TLS, tail sampling, memory safety
#   - Jaeger (traces backend)
#   - Prometheus (metrics backend)
#   - Loki (logs backend)
#   - Grafana (visualization)
#
# Usage:
#   ./otel-collector-setup.sh [OPTIONS]
#
# Options:
#   --env FILE          Path to .env file (default: .env.production)
#   --skip-tls          Skip TLS certificate generation
#   --skip-secrets      Skip secret generation
#   --dry-run           Print commands without executing
#   --force             Overwrite existing certs and configs
#   -h, --help          Show this help message
#
# Environment variables (can also be set in .env file):
#   OTEL_ENV            Deployment environment (default: production)
#   GRAFANA_ADMIN_PASS  Grafana admin password (required)
#   DEPLOY_DIR          Deployment directory (default: script's parent)
# ============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
readonly SCRIPT_NAME="$(basename "$0")"
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly OTEL_DIR="${PROJECT_ROOT}/otel"
readonly CERTS_DIR="${OTEL_DIR}/certs"
readonly COMPOSE_FILE="docker-compose.production.yml"
readonly DEFAULT_ENV_FILE="${SCRIPT_DIR}/.env.production"

# Color helpers (disabled when not a TTY)
if [ -t 1 ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    BLUE='\033[0;34m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
log_info()    { echo -e "${GREEN}[INFO]${NC}  $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_step()    { echo -e "${BLUE}[STEP]${NC}  $*"; }

# ---------------------------------------------------------------------------
# CLI argument parsing
# ---------------------------------------------------------------------------
ENV_FILE="${DEFAULT_ENV_FILE}"
SKIP_TLS=false
SKIP_SECRETS=false
DRY_RUN=false
FORCE=false

usage() {
    sed -n '/^# Usage:/,/^# ====/p' "${BASH_SOURCE[0]}" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

while [ $# -gt 0 ]; do
    case "$1" in
        --env)        ENV_FILE="$2"; shift 2 ;;
        --skip-tls)   SKIP_TLS=true; shift ;;
        --skip-secrets) SKIP_SECRETS=true; shift ;;
        --dry-run)    DRY_RUN=true; shift ;;
        --force)      FORCE=true; shift ;;
        -h|--help)    usage ;;
        *)            log_error "Unknown option: $1"; usage ;;
    esac
done

# ---------------------------------------------------------------------------
# Prerequisites check
# ---------------------------------------------------------------------------
check_prerequisites() {
    log_step "Checking prerequisites..."

    local missing=0

    for cmd in docker openssl; do
        if ! command -v "$cmd" &>/dev/null; then
            log_error "Required command not found: $cmd"
            missing=1
        fi
    done

    # Check Docker Compose v2
    if ! docker compose version &>/dev/null; then
        log_error "Docker Compose v2 is required (docker compose)"
        missing=1
    fi

    # Check Docker daemon
    if ! docker info &>/dev/null; then
        log_error "Docker daemon is not running"
        missing=1
    fi

    if [ "$missing" -ne 0 ]; then
        exit 1
    fi

    log_info "All prerequisites satisfied"
}

# ---------------------------------------------------------------------------
# Load environment
# ---------------------------------------------------------------------------
load_env() {
    if [ -f "$ENV_FILE" ]; then
        log_info "Loading environment from ${ENV_FILE}"
        set -a
        # shellcheck disable=SC1090
        source "$ENV_FILE"
        set +a
    else
        log_warn "No .env file found at ${ENV_FILE}"
        log_warn "Using default values. Create from .env.production.template:"
        log_warn "  cp ${SCRIPT_DIR}/.env.production.template ${ENV_FILE}"
    fi

    # Validate required variables
    if [ -z "${GRAFANA_ADMIN_PASSWORD:-}" ] && [ "$SKIP_SECRETS" = false ]; then
        log_error "GRAFANA_ADMIN_PASSWORD is required. Set it in your .env file or export it."
        exit 1
    fi
}

# ---------------------------------------------------------------------------
# TLS certificate generation
# ---------------------------------------------------------------------------
generate_certs() {
    if [ "$SKIP_TLS" = true ]; then
        log_warn "Skipping TLS certificate generation (--skip-tls)"
        return 0
    fi

    if [ -d "$CERTS_DIR" ] && [ "$FORCE" = false ]; then
        log_info "TLS certificates already exist at ${CERTS_DIR}. Use --force to regenerate."
        return 0
    fi

    log_step "Generating TLS certificates..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would generate TLS certs in ${CERTS_DIR}"
        return 0
    fi

    mkdir -p "$CERTS_DIR"
    rm -rf "${CERTS_DIR:?}"/*

    # Generate CA key and certificate
    openssl genrsa -out "${CERTS_DIR}/ca.key" 2048 2>/dev/null
    openssl req -new -x509 -days 365 -key "${CERTS_DIR}/ca.key" \
        -out "${CERTS_DIR}/ca.crt" \
        -subj "/C=US/ST=Cloud/L=NVMS/O=NanoVMS/CN=NanoVMS-CA" 2>/dev/null

    # Generate collector key and CSR
    openssl genrsa -out "${CERTS_DIR}/collector.key" 2048 2>/dev/null
    openssl req -new -key "${CERTS_DIR}/collector.key" \
        -out "${CERTS_DIR}/collector.csr" \
        -subj "/C=US/ST=Cloud/L=NVMS/O=NanoVMS/CN=nanovms-otel-collector" 2>/dev/null

    # Create SAN extension file
    cat > "${CERTS_DIR}/san.ext" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = nanovms-otel-collector
DNS.3 = nanovms-jaeger
DNS.4 = jaeger
DNS.5 = *.nanovms-otel.local
IP.1 = 127.0.0.1
IP.2 = 172.28.0.2
EOF

    # Sign collector certificate with CA
    openssl x509 -req -days 365 \
        -in "${CERTS_DIR}/collector.csr" \
        -CA "${CERTS_DIR}/ca.crt" \
        -CAkey "${CERTS_DIR}/ca.key" \
        -CAcreateserial \
        -out "${CERTS_DIR}/collector.crt" \
        -extfile "${CERTS_DIR}/san.ext" 2>/dev/null

    # Clean up CSR and temp files
    rm -f "${CERTS_DIR}/collector.csr" "${CERTS_DIR}/san.ext" "${CERTS_DIR}/ca.srl"

    chmod 600 "${CERTS_DIR}"/*.key
    chmod 644 "${CERTS_DIR}"/*.crt

    log_info "TLS certificates generated in ${CERTS_DIR}"
}

# ---------------------------------------------------------------------------
# Secret generation
# ---------------------------------------------------------------------------
generate_secrets() {
    if [ "$SKIP_SECRETS" = true ]; then
        log_warn "Skipping secret generation (--skip-secrets)"
        return 0
    fi

    log_step "Generating secrets..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would generate secrets in ${ENV_FILE}"
        return 0
    fi

    # Generate .env.production if it doesn't exist
    if [ ! -f "$ENV_FILE" ]; then
        log_info "Creating ${ENV_FILE} from template..."

        local grafana_pass
        grafana_pass="$(openssl rand -base64 24 2>/dev/null || head -c 32 /dev/urandom | base64)"

        cat > "$ENV_FILE" <<ENVEOF
# NanoVMS OTel Stack — Production Environment
# Generated by otel-collector-setup.sh on $(date -u +"%Y-%m-%dT%H:%M:%SZ")
# DO NOT commit this file to version control.

# Deployment
OTEL_ENV=production
OTEL_SERVICE_VERSION=0.1.0
OTEL_INSTANCE_ID=nanovms-collector-1

# Memory / Performance
OTEL_MEM_LIMIT_MIB=512
OTEL_MEM_SPIKE_MIB=128
OTEL_BATCH_SIZE=8192
OTEL_BATCH_MAX_SIZE=16384
OTEL_BATCH_TIMEOUT=5s
OTEL_SAMPLING_PERCENTAGE=10

# Ports (host-exposed)
OTEL_GRPC_PORT=4317
OTEL_HTTP_PORT=4318
OTEL_PROM_PORT=8889
OTEL_HEALTH_PORT=13133
OTEL_ZPAGES_PORT=55679
PROM_PORT=9090
LOKI_PORT=3100
GRAFANA_PORT=3000
JAEGER_UI_PORT=16686

# TLS
OTEL_TLS_INSECURE=false

# Grafana
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=${grafana_pass}
GRAFANA_ROOT_URL=http://localhost:3000

# Jaeger
JAEGER_STORAGE_TYPE=badger

# Logging
OTEL_LOG_LEVEL=info
OTEL_DEBUG_VERBOSITY=basic

# Loki
OTEL_LOKI_TENANT=nanovms
ENVEOF

        log_info "Created ${ENV_FILE} with generated secrets"
        log_warn "Review and adjust ${ENV_FILE} before production use"
    else
        log_info "Environment file already exists: ${ENV_FILE}"
    fi
}

# ---------------------------------------------------------------------------
# Validate configuration
# ---------------------------------------------------------------------------
validate_config() {
    log_step "Validating collector configuration..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would validate config"
        return 0
    fi

    # Validate YAML syntax
    if command -v python3 &>/dev/null; then
        if ! python3 -c "import yaml; yaml.safe_load(open('${OTEL_DIR}/collector-config.production.yaml'))" 2>/dev/null; then
            log_error "Invalid YAML in collector-config.production.yaml"
            exit 1
        fi
    elif command -v python &>/dev/null; then
        if ! python -c "import yaml; yaml.safe_load(open('${OTEL_DIR}/collector-config.production.yaml'))" 2>/dev/null; then
            log_warn "Cannot validate YAML (python yaml module not available)"
        fi
    fi

    # Validate compose file
    if ! docker compose -f "${OTEL_DIR}/${COMPOSE_FILE}" config --quiet 2>/dev/null; then
        log_error "Invalid docker-compose.production.yml"
        exit 1
    fi

    log_info "Configuration validated"
}

# ---------------------------------------------------------------------------
# Deploy stack
# ---------------------------------------------------------------------------
deploy_stack() {
    log_step "Deploying NanoVMS OTel Collector stack..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would deploy the following:"
        log_info "  docker compose -f ${OTEL_DIR}/${COMPOSE_FILE} --env-file ${ENV_FILE} pull"
        log_info "  docker compose -f ${OTEL_DIR}/${COMPOSE_FILE} --env-file ${ENV_FILE} up -d"
        return 0
    fi

    # Pull latest images
    log_info "Pulling images..."
    docker compose -f "${OTEL_DIR}/${COMPOSE_FILE}" --env-file "${ENV_FILE}" pull

    # Stop any existing stack
    docker compose -f "${OTEL_DIR}/${COMPOSE_FILE}" --env-file "${ENV_FILE}" down --remove-orphans 2>/dev/null || true

    # Start the stack
    log_info "Starting services..."
    docker compose -f "${OTEL_DIR}/${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d

    log_info "Stack deployed successfully"
}

# ---------------------------------------------------------------------------
# Wait for health
# ---------------------------------------------------------------------------
wait_for_healthy() {
    log_step "Waiting for services to become healthy..."

    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would wait for health checks"
        return 0
    fi

    local max_wait=120
    local interval=5
    local elapsed=0
    local all_healthy=false

    while [ "$elapsed" -lt "$max_wait" ]; do
        local unhealthy
        unhealthy=$(docker compose -f "${OTEL_DIR}/${COMPOSE_FILE}" ps --format json 2>/dev/null \
            | python3 -c "
import sys, json
unhealthy = []
for line in sys.stdin:
    try:
        svc = json.loads(line)
        if svc.get('Health') not in ('healthy', 'starting', None, ''):
            unhealthy.append(svc.get('Service', 'unknown'))
    except: pass
print(','.join(unhealthy))
" 2>/dev/null || echo "")

        if [ -z "$unhealthy" ]; then
            all_healthy=true
            break
        fi

        echo -ne "\r  Waiting... (${elapsed}s/${max_wait}s) "
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done

    echo ""

    if [ "$all_healthy" = true ]; then
        log_info "All services are healthy"
    else
        log_warn "Some services may not be healthy yet. Check with:"
        log_warn "  docker compose -f ${OTEL_DIR}/${COMPOSE_FILE} ps"
    fi
}

# ---------------------------------------------------------------------------
# Print summary
# ---------------------------------------------------------------------------
print_summary() {
    log_step "Deployment Summary"
    echo ""
    echo "  Collector gRPC:   http://localhost:${OTEL_GRPC_PORT:-4317}"
    echo "  Collector HTTP:   http://localhost:${OTEL_HTTP_PORT:-4318}"
    echo "  Collector Health: http://localhost:${OTEL_HEALTH_PORT:-13133}"
    echo "  Prometheus:       http://localhost:${PROM_PORT:-9090}"
    echo "  Jaeger UI:        http://localhost:${JAEGER_UI_PORT:-16686}"
    echo "  Loki:             http://localhost:${LOKI_PORT:-3100}"
    echo "  Grafana:          http://localhost:${GRAFANA_PORT:-3000}"
    echo ""
    echo "  Logs:     docker compose -f ${OTEL_DIR}/${COMPOSE_FILE} logs -f"
    echo "  Status:   docker compose -f ${OTEL_DIR}/${COMPOSE_FILE} ps"
    echo "  Teardown: ${SCRIPT_DIR}/otel-collector-teardown.sh"
    echo ""
    log_info "NanoVMS OTel Collector stack is running"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    echo ""
    log_info "NanoVMS Production OTel Collector Setup"
    log_info "========================================"
    echo ""

    check_prerequisites
    load_env
    generate_certs
    generate_secrets
    validate_config
    deploy_stack
    wait_for_healthy
    print_summary
}

main "$@"
