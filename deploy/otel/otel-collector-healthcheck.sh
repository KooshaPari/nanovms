#!/usr/bin/env bash
# ============================================================================
# otel-collector-healthcheck.sh — Health check for the NanoVMS OTel stack
# ============================================================================
#
# Verifies all services are healthy and the collector is receiving telemetry.
#
# Usage:
#   ./otel-collector-healthcheck.sh [OPTIONS]
#
# Options:
#   --env FILE        Path to .env file (default: .env.production)
#   --json            Output in JSON format
#   --timeout SEC     Timeout for each check (default: 10)
#   -h, --help        Show this help message
#
# Exit codes:
#   0  All checks passed
#   1  One or more checks failed
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

log_pass()    { echo -e "  ${GREEN}PASS${NC}  $*"; }
log_fail()    { echo -e "  ${RED}FAIL${NC}  $*"; }
log_warn()    { echo -e "  ${YELLOW}WARN${NC}  $*"; }
log_step()    { echo -e "${BLUE}[STEP]${NC}  $*"; }

# ---------------------------------------------------------------------------
# CLI arguments
# ---------------------------------------------------------------------------
ENV_FILE="${DEFAULT_ENV_FILE}"
JSON_OUTPUT=false
CHECK_TIMEOUT=10

usage() {
    sed -n '/^# Usage:/,/^# ====/p' "${BASH_SOURCE[0]}" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

while [ $# -gt 0 ]; do
    case "$1" in
        --env)      ENV_FILE="$2"; shift 2 ;;
        --json)     JSON_OUTPUT=true; shift ;;
        --timeout)  CHECK_TIMEOUT="$2"; shift 2 ;;
        -h|--help)  usage ;;
        *)          echo "Unknown option: $1"; usage ;;
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
# Results tracking
# ---------------------------------------------------------------------------
TOTAL=0
PASSED=0
FAILED=0
WARNED=0
RESULTS=()

record_result() {
    local status="$1" name="$2" detail="$3"
    TOTAL=$((TOTAL + 1))
    case "$status" in
        pass) PASSED=$((PASSED + 1)); log_pass "$name" ;;
        fail) FAILED=$((FAILED + 1)); log_fail "$name — $detail" ;;
        warn) WARNED=$((WARNED + 1)); log_warn "$name — $detail" ;;
    esac
    RESULTS+=("{\"status\":\"${status}\",\"check\":\"${name}\",\"detail\":\"${detail}\"}")
}

# ---------------------------------------------------------------------------
# Check: Docker Compose services
# ---------------------------------------------------------------------------
check_compose_services() {
    log_step "Checking Docker Compose services..."

    local services=("collector" "jaeger" "prometheus" "loki" "grafana")
    local compose_cmd="docker compose -f ${OTEL_DIR}/${COMPOSE_FILE}"

    for svc in "${services[@]}"; do
        local status
        status=$($compose_cmd ps --format json "$svc" 2>/dev/null \
            | python3 -c "
import sys, json
for line in sys.stdin:
    try:
        svc = json.loads(line)
        print(svc.get('State', 'unknown'))
    except: pass
" 2>/dev/null || echo "not found")

        case "$status" in
            running)
                # Check health if available
                local health
                health=$($compose_cmd ps --format json "$svc" 2>/dev/null \
                    | python3 -c "
import sys, json
for line in sys.stdin:
    try:
        svc = json.loads(line)
        print(svc.get('Health', 'none'))
    except: pass
" 2>/dev/null || echo "none")

                case "$health" in
                    healthy) record_result "pass" "$svc" "running and healthy" ;;
                    starting) record_result "warn" "$svc" "running but still starting" ;;
                    none) record_result "pass" "$svc" "running (no healthcheck)" ;;
                    *) record_result "warn" "$svc" "running but health: $health" ;;
                esac
                ;;
            exited|dead)
                record_result "fail" "$svc" "status: $status" ;;
            *)
                record_result "fail" "$svc" "not found or status: $status" ;;
        esac
    done
}

# ---------------------------------------------------------------------------
# Check: Collector health endpoint
# ---------------------------------------------------------------------------
check_collector_health() {
    log_step "Checking collector health endpoint..."

    local health_port="${OTEL_HEALTH_PORT:-13133}"
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout "$CHECK_TIMEOUT" \
        "http://localhost:${health_port}" 2>/dev/null || echo "000")

    if [ "$http_code" = "200" ]; then
        record_result "pass" "collector-health" "HTTP $http_code"
    else
        record_result "fail" "collector-health" "HTTP $http_code"
    fi
}

# ---------------------------------------------------------------------------
# Check: Collector metrics endpoint
# ---------------------------------------------------------------------------
check_collector_metrics() {
    log_step "Checking collector metrics endpoint..."

    local metrics_port="${OTEL_PROM_PORT:-8889}"
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout "$CHECK_TIMEOUT" \
        "http://localhost:${metrics_port}/metrics" 2>/dev/null || echo "000")

    if [ "$http_code" = "200" ]; then
        record_result "pass" "collector-metrics" "HTTP $http_code"
    else
        record_result "fail" "collector-metrics" "HTTP $http_code"
    fi
}

# ---------------------------------------------------------------------------
# Check: Prometheus
# ---------------------------------------------------------------------------
check_prometheus() {
    log_step "Checking Prometheus..."

    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout "$CHECK_TIMEOUT" \
        "http://localhost:${PROM_PORT:-9090}/-/healthy" 2>/dev/null || echo "000")

    if [ "$http_code" = "200" ]; then
        record_result "pass" "prometheus" "healthy"
    else
        record_result "fail" "prometheus" "HTTP $http_code"
    fi
}

# ---------------------------------------------------------------------------
# Check: Grafana
# ---------------------------------------------------------------------------
check_grafana() {
    log_step "Checking Grafana..."

    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout "$CHECK_TIMEOUT" \
        "http://localhost:${GRAFANA_PORT:-3000}/api/health" 2>/dev/null || echo "000")

    if [ "$http_code" = "200" ]; then
        record_result "pass" "grafana" "healthy"
    else
        record_result "fail" "grafana" "HTTP $http_code"
    fi
}

# ---------------------------------------------------------------------------
# Check: Jaeger
# ---------------------------------------------------------------------------
check_jaeger() {
    log_step "Checking Jaeger..."

    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout "$CHECK_TIMEOUT" \
        "http://localhost:${JAEGER_UI_PORT:-16686}/" 2>/dev/null || echo "000")

    if [ "$http_code" = "200" ]; then
        record_result "pass" "jaeger" "UI reachable"
    else
        record_result "fail" "jaeger" "HTTP $http_code"
    fi
}

# ---------------------------------------------------------------------------
# Check: Loki
# ---------------------------------------------------------------------------
check_loki() {
    log_step "Checking Loki..."

    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout "$CHECK_TIMEOUT" \
        "http://localhost:${LOKI_PORT:-3100}/ready" 2>/dev/null || echo "000")

    if [ "$http_code" = "200" ]; then
        record_result "pass" "loki" "ready"
    else
        record_result "fail" "loki" "HTTP $http_code"
    fi
}

# ---------------------------------------------------------------------------
# Check: OTLP receiver (send a test span)
# ---------------------------------------------------------------------------
check_otlp_receiver() {
    log_step "Checking OTLP receiver..."

    local http_port="${OTEL_HTTP_PORT:-4318}"
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout "$CHECK_TIMEOUT" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"resourceSpans":[]}' \
        "http://localhost:${http_port}/v1/traces" 2>/dev/null || echo "000")

    # 200 = accepted, 400 = bad request (but receiver is up), 415 = wrong content type
    if [ "$http_code" = "200" ] || [ "$http_code" = "400" ] || [ "$http_code" = "415" ]; then
        record_result "pass" "otlp-receiver" "HTTP $http_code (receiver is up)"
    else
        record_result "fail" "otlp-receiver" "HTTP $http_code"
    fi
}

# ---------------------------------------------------------------------------
# Check: Prometheus targets
# ---------------------------------------------------------------------------
check_prometheus_targets() {
    log_step "Checking Prometheus scrape targets..."

    local targets
    targets=$(curl -s --connect-timeout "$CHECK_TIMEOUT" \
        "http://localhost:${PROM_PORT:-9090}/api/v1/targets" 2>/dev/null || echo "{}")

    local up_count
    up_count=$(echo "$targets" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    active = data.get('data', {}).get('activeTargets', [])
    up = sum(1 for t in active if t.get('health') == 'up')
    print(up)
except: print(0)
" 2>/dev/null || echo "0")

    if [ "$up_count" -gt 0 ]; then
        record_result "pass" "prometheus-targets" "${up_count} target(s) up"
    else
        record_result "warn" "prometheus-targets" "no targets confirmed up"
    fi
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
print_summary() {
    echo ""

    if [ "$JSON_OUTPUT" = true ]; then
        echo "{"
        echo "  \"total\": ${TOTAL},"
        echo "  \"passed\": ${PASSED},"
        echo "  \"failed\": ${FAILED},"
        echo "  \"warned\": ${WARNED},"
        echo "  \"checks\": ["
        local first=true
        for r in "${RESULTS[@]}"; do
            if [ "$first" = true ]; then first=false; else echo ","; fi
            echo -n "    ${r}"
        done
        echo ""
        echo "  ]"
        echo "}"
        return
    fi

    echo "========================================"
    echo -n "  Results: "
    echo -e "${GREEN}${PASSED} passed${NC}, ${RED}${FAILED} failed${NC}, ${YELLOW}${WARNED} warned${NC} (${TOTAL} total)"
    echo "========================================"
    echo ""

    if [ "$FAILED" -gt 0 ]; then
        echo -e "  ${RED}Some health checks failed.${NC}"
        echo "  Run the following for diagnostics:"
        echo "    docker compose -f ${OTEL_DIR}/${COMPOSE_FILE} ps"
        echo "    docker compose -f ${OTEL_DIR}/${COMPOSE_FILE} logs --tail=50"
        echo ""
        return 1
    else
        echo -e "  ${GREEN}All health checks passed.${NC}"
        echo ""
        return 0
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    echo ""
    echo -e "${BLUE}NanoVMS OTel Collector Health Check${NC}"
    echo "==================================="
    echo ""

    check_compose_services
    check_collector_health
    check_collector_metrics
    check_prometheus
    check_grafana
    check_jaeger
    check_loki
    check_otlp_receiver
    check_prometheus_targets

    print_summary
}

main "$@"
