#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
COMPOSE_FILES_DIR="${COMPOSE_FILES_DIR:-$COMPOSE_DIR/deploy/docker}"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.production}"
COMPOSE="docker compose --env-file $ENV_FILE -f $COMPOSE_FILES_DIR/docker-compose.yml -f $COMPOSE_FILES_DIR/docker-compose.prod.yml"

HEALTHY=0
DEGRADED=0
FAILED=0

check() {
    local name=$1 cmd=$2
    if eval "$cmd" > /dev/null 2>&1; then
        echo "  [OK]   $name"
        # `((var++))` returns exit code 1 when the pre-increment value is 0.
        # Under `set -e` that aborts the script on the very first OK check
        # (which is exactly what happened during the 2026-05-08 deploy).
        # Use arithmetic assignment instead — `=` is always exit 0.
        HEALTHY=$((HEALTHY + 1))
    else
        echo "  [FAIL] $name"
        FAILED=$((FAILED + 1))
    fi
}

echo "KMU Hub Health Check"
echo "===================="
echo ""

# Gateway (HTTP)
check "Gateway (HTTP)" "curl -sf http://localhost:8080/health"

# Service health endpoints — all 23 backend services (gateway is checked above).
# Each entry is "service_dns_name:health_port"; the DNS name matches the compose
# service (= cmd dir). Ports come from config.go (*_HEALTH_PORT defaults).
SERVICES=(
    "auth:9091" "crm:9092" "chat:9093" "notification:9094" "work:9095"
    "email:9096" "document:9097" "biz:9098" "automation:9099" "plugin:9100"
    "dialer:9101" "wiki:9102" "berichte:9103" "formulare:9104" "helpdesk:9105"
    "inventar:9110" "einkauf:9111" "produktion:9112" "vertraege:9113" "rapporte:9114"
    "schichten:9115" "fuhrpark:9116" "vermietung:9117"
)
for entry in "${SERVICES[@]}"; do
    svc="${entry%%:*}"
    port="${entry##*:}"
    check "$svc (port $port)" "$COMPOSE exec -T gateway wget --spider --quiet http://$svc:$port/health 2>/dev/null"
done

# Infrastructure
check "PostgreSQL" "$COMPOSE exec -T postgres pg_isready -U kmuhub"
check "Redis" "$COMPOSE exec -T redis redis-cli ping"
# Caddy serves TLS only for the configured domain (app.zentria.tech in prod);
# https://localhost has no matching cert and fails the TLS handshake.
# Override CADDY_HEALTHCHECK_HOST in non-prod environments as needed.
CADDY_HEALTHCHECK_HOST="${CADDY_HEALTHCHECK_HOST:-app.zentria.tech}"
check "Caddy (HTTPS)" "curl -ksf --resolve $CADDY_HEALTHCHECK_HOST:443:127.0.0.1 https://$CADDY_HEALTHCHECK_HOST/health"

echo ""
echo "===================="
TOTAL=$((HEALTHY + FAILED))
echo "Results: $HEALTHY/$TOTAL healthy, $FAILED failed"

if [[ $FAILED -gt 0 ]]; then
    echo ""
    echo "Unhealthy services detected!"
    echo "Run: docker compose ps"
    exit 2
fi

echo "All services healthy."
exit 0
