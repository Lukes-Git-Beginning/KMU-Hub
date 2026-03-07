#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKIP_BACKUP=false
SERVICE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-backup) SKIP_BACKUP=true; shift ;;
        --service=*) SERVICE="${1#*=}"; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

COMPOSE="docker compose -f $COMPOSE_DIR/docker-compose.yml -f $COMPOSE_DIR/docker-compose.prod.yml"

log() { echo "[deploy] $(date '+%Y-%m-%d %H:%M:%S') $*"; }

log "=========================================="
log "  KMU Hub Production Deployment"
log "=========================================="

# Step 1: Backup
if [[ "$SKIP_BACKUP" == "false" ]]; then
    log "Step 1/6: Creating backup..."
    "$SCRIPT_DIR/backup.sh"
else
    log "Step 1/6: Backup skipped (--skip-backup)"
fi

# Step 2: Pull latest code
log "Step 2/6: Pulling latest code..."
cd "$COMPOSE_DIR"
git pull origin main

# Step 3: Build images
log "Step 3/6: Building images..."
if [[ -n "$SERVICE" ]]; then
    $COMPOSE build "$SERVICE"
else
    $COMPOSE build
fi

# Step 4: Run migrations
log "Step 4/6: Running migrations..."
$COMPOSE run --rm migrate

# Step 5: Restart services
log "Step 5/6: Restarting services..."
if [[ -n "$SERVICE" ]]; then
    $COMPOSE up -d "$SERVICE"
    log "Waiting for $SERVICE health check..."
    sleep 5
else
    # Rolling restart: infrastructure first, then services, then gateway
    $COMPOSE up -d postgres redis minio
    sleep 5

    for svc in auth crm chat notification work email document biz automation plugin; do
        log "  Starting $svc..."
        $COMPOSE up -d "$svc"
        sleep 3
    done

    $COMPOSE up -d gateway caddy
    sleep 5
fi

# Step 6: Health check
log "Step 6/6: Running health checks..."
if [[ -f "$SCRIPT_DIR/healthcheck.sh" ]]; then
    "$SCRIPT_DIR/healthcheck.sh"
else
    $COMPOSE ps
fi

log "=========================================="
log "  Deployment complete!"
log "=========================================="
log ""
log "Rollback: git checkout HEAD~1 && $0 --skip-backup"
