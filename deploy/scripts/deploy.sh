#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
COMPOSE_FILES_DIR="${COMPOSE_FILES_DIR:-$COMPOSE_DIR/deploy/docker}"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.production}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SKIP_BACKUP=false
FORCE=false
SERVICE=""
DEPLOY_LOCK="$COMPOSE_DIR/.deploy.lock"
DEPLOY_LOG="$COMPOSE_DIR/deploy-history.log"
DEPLOY_START=$(date +%s)

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-backup) SKIP_BACKUP=true; shift ;;
        --skip-smoke) SKIP_SMOKE=true; shift ;;
        --force) FORCE=true; shift ;;
        --service=*) SERVICE="${1#*=}"; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

COMPOSE="docker compose --env-file $ENV_FILE -f $COMPOSE_FILES_DIR/docker-compose.yml -f $COMPOSE_FILES_DIR/docker-compose.prod.yml"

log() { echo "[deploy] $(date '+%Y-%m-%d %H:%M:%S') $*"; }

log_deploy() {
    local status=$1
    local prev_sha=$2
    local new_sha=$3
    local duration=$(( $(date +%s) - DEPLOY_START ))
    echo "$(date '+%Y-%m-%d %H:%M:%S')	$prev_sha	$new_sha	$status	${duration}s" >> "$DEPLOY_LOG"
}

cleanup() {
    rm -f "$DEPLOY_LOCK"
}

rollback() {
    local prev_sha=$1
    local current_sha
    current_sha=$(git -C "$COMPOSE_DIR" rev-parse HEAD)

    log "AUTO-ROLLBACK: Rolling back from $current_sha to $prev_sha"

    cd "$COMPOSE_DIR"
    git checkout "$prev_sha"

    # Calculate migration rollback count
    local new_migrations current_migrations diff
    new_migrations=$(ls -1 "$COMPOSE_DIR/backend/migrations/"*.up.sql 2>/dev/null | wc -l)
    git stash 2>/dev/null || true
    current_migrations=$(ls -1 "$COMPOSE_DIR/backend/migrations/"*.up.sql 2>/dev/null | wc -l)
    git stash pop 2>/dev/null || true

    diff=$(( new_migrations - current_migrations ))
    if [[ $diff -gt 0 ]]; then
        log "Rolling back $diff migration(s)..."
        $COMPOSE run --rm migrate -path /migrations -database "$DATABASE_URL" down "$diff" || true
    fi

    # Rebuild and restart
    local build_version build_commit
    build_version=$(git -C "$COMPOSE_DIR" describe --tags --always 2>/dev/null || echo "rollback")
    build_commit=$(git -C "$COMPOSE_DIR" rev-parse --short HEAD)

    # Rebuild serially — parallel `compose build` of all 24+ Go services has
    # hit OOM-kill on the 8 GB host (memory: feedback_swap_for_compose_builds.md
    # plus rollback's own learning during welle 3.4 on 2026-05-11). Matches the
    # serial pattern of Step 3 above so behaviour stays consistent.
    local build_time
    build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    local rollback_buildable="auth crm chat notification work email document biz automation plugin dialer wiki helpdesk berichte formulare inventar einkauf produktion vertraege rapporte schichten fuhrpark vermietung gateway migrate"
    for svc in $rollback_buildable; do
        log "AUTO-ROLLBACK: building $svc..."
        $COMPOSE build \
            --build-arg BUILD_VERSION="$build_version" \
            --build-arg BUILD_COMMIT="$build_commit" \
            --build-arg BUILD_TIME="$build_time" \
            "$svc"
    done

    $COMPOSE up -d postgres redis minio
    sleep 5
    for svc in auth crm chat notification work email document biz automation plugin dialer wiki helpdesk berichte formulare inventar einkauf produktion vertraege rapporte schichten fuhrpark vermietung livekit livekit-egress; do
        $COMPOSE up -d "$svc"
        sleep 3
    done
    $COMPOSE up -d gateway caddy
    sleep 5

    # Verify rollback health
    if [[ -f "$SCRIPT_DIR/healthcheck.sh" ]]; then
        if "$SCRIPT_DIR/healthcheck.sh"; then
            log "AUTO-ROLLBACK: Health check passed after rollback"
            log_deploy "rollback-success" "$current_sha" "$prev_sha"
        else
            log "AUTO-ROLLBACK: Health check FAILED after rollback — manual intervention required"
            log_deploy "rollback-failed" "$current_sha" "$prev_sha"
        fi
    fi
}

# ==========================================
# Deployment Lock
# ==========================================
if [[ -f "$DEPLOY_LOCK" ]]; then
    lock_pid=$(cat "$DEPLOY_LOCK" 2>/dev/null || echo "unknown")
    log "ERROR: Another deployment is running (PID: $lock_pid)"
    log "Lock file: $DEPLOY_LOCK"
    log "If stale, remove manually: rm $DEPLOY_LOCK"
    exit 1
fi

echo $$ > "$DEPLOY_LOCK"
trap cleanup EXIT

log "=========================================="
log "  KMU Hub Production Deployment"
log "=========================================="

# Pre-Deploy Snapshot
cd "$COMPOSE_DIR"
PREV_SHA=$(git rev-parse HEAD)
log "Previous commit: $PREV_SHA"

# Step 1: Backup
if [[ "$SKIP_BACKUP" == "false" ]]; then
    log "Step 1/7: Creating backup..."
    "$SCRIPT_DIR/backup.sh"
else
    log "Step 1/7: Backup skipped (--skip-backup)"
fi

# Step 2: Pull latest code
log "Step 2/7: Pulling latest code..."
git pull origin main
NEW_SHA=$(git rev-parse HEAD)
log "New commit: $NEW_SHA"

if [[ "$PREV_SHA" == "$NEW_SHA" && -z "$SERVICE" && "$FORCE" == "false" ]]; then
    log "No changes to deploy (already at $NEW_SHA)"
    log_deploy "no-change" "$PREV_SHA" "$NEW_SHA"
    exit 0
fi

if [[ "$PREV_SHA" == "$NEW_SHA" && "$FORCE" == "true" ]]; then
    log "Force-rebuild requested — Code unchanged ($NEW_SHA), but rebuilding anyway"
fi

# Step 2.5: Render templated config files (livekit-secrets.yaml from .env.production)
log "Step 2.5/7: Rendering config templates..."
set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a
"$SCRIPT_DIR/render-configs.sh"

# Step 3: Build images with version info
# Built sequentially per service to keep peak RAM under 16GB on CPX42 hosts.
# 24 parallel Go builds via buildx-bake exceed memory and trigger OOM-killer.
log "Step 3/7: Building images..."
BUILD_VERSION=$(git describe --tags --always 2>/dev/null || echo "$NEW_SHA")
BUILD_COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

BUILDABLE_SERVICES="auth crm chat notification work email document biz automation plugin dialer wiki helpdesk berichte formulare inventar einkauf produktion vertraege rapporte schichten fuhrpark vermietung gateway migrate"

if [[ -n "$SERVICE" ]]; then
    $COMPOSE build \
        --build-arg BUILD_VERSION="$BUILD_VERSION" \
        --build-arg BUILD_COMMIT="$BUILD_COMMIT" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        "$SERVICE"
else
    for svc in $BUILDABLE_SERVICES; do
        log "  Building $svc..."
        $COMPOSE build \
            --build-arg BUILD_VERSION="$BUILD_VERSION" \
            --build-arg BUILD_COMMIT="$BUILD_COMMIT" \
            --build-arg BUILD_TIME="$BUILD_TIME" \
            "$svc"
    done
fi

# Step 4: Run migrations
log "Step 4/7: Running migrations..."
$COMPOSE run --rm migrate

# Step 5: Restart services
log "Step 5/7: Restarting services..."
if [[ -n "$SERVICE" ]]; then
    $COMPOSE up -d "$SERVICE"
    log "Waiting for $SERVICE health check..."
    sleep 5
else
    # Rolling restart: infrastructure first, then services, then gateway
    $COMPOSE up -d postgres redis minio
    sleep 5

    for svc in auth crm chat notification work email document biz automation plugin dialer wiki helpdesk berichte formulare inventar einkauf produktion vertraege rapporte schichten fuhrpark vermietung livekit livekit-egress; do
        log "  Starting $svc..."
        $COMPOSE up -d "$svc"
        sleep 3
    done

    $COMPOSE up -d gateway caddy
    sleep 5
fi

# Step 6: Health check
log "Step 6/7: Running health checks..."
if [[ -f "$SCRIPT_DIR/healthcheck.sh" ]]; then
    if ! "$SCRIPT_DIR/healthcheck.sh"; then
        log "Health check FAILED — initiating auto-rollback..."
        rollback "$PREV_SHA"
        exit 1
    fi
else
    $COMPOSE ps
fi

# Step 7: Smoke tests
log "Step 7/7: Running smoke tests..."
if [[ "${SKIP_SMOKE:-false}" == "true" ]]; then
    log "Smoke tests skipped (--skip-smoke). Run manually: deploy/scripts/smoke.sh --base-url https://app.zentria.tech"
elif [[ -f "$SCRIPT_DIR/smoke.sh" ]]; then
    if ! "$SCRIPT_DIR/smoke.sh" --base-url https://app.zentria.tech --expect-version "$BUILD_COMMIT"; then
        log "Smoke tests FAILED — initiating auto-rollback..."
        rollback "$PREV_SHA"
        exit 1
    fi
else
    log "Smoke script not found, skipping..."
fi

# Log success
log_deploy "success" "$PREV_SHA" "$NEW_SHA"

log "=========================================="
log "  Deployment complete!"
log "  Version: $BUILD_VERSION ($BUILD_COMMIT)"
log "=========================================="
