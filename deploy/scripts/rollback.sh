#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_LOCK="$COMPOSE_DIR/.deploy.lock"
DEPLOY_LOG="$COMPOSE_DIR/deploy-history.log"
ROLLBACK_START=$(date +%s)

TARGET_SHA=""
LIST_MODE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --to) TARGET_SHA="$2"; shift 2 ;;
        --to=*) TARGET_SHA="${1#*=}"; shift ;;
        --list) LIST_MODE=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

COMPOSE="docker compose -f $COMPOSE_DIR/docker-compose.yml -f $COMPOSE_DIR/docker-compose.prod.yml"

log() { echo "[rollback] $(date '+%Y-%m-%d %H:%M:%S') $*"; }

log_deploy() {
    local status=$1
    local prev_sha=$2
    local new_sha=$3
    local duration=$(( $(date +%s) - ROLLBACK_START ))
    echo "$(date '+%Y-%m-%d %H:%M:%S')	$prev_sha	$new_sha	$status	${duration}s" >> "$DEPLOY_LOG"
}

# --list mode: show recent deployments
if [[ "$LIST_MODE" == "true" ]]; then
    echo "Recent deployments (newest first):"
    echo "==========================================================="
    printf "%-20s %-10s %-10s %-16s %s\n" "TIMESTAMP" "FROM" "TO" "STATUS" "DURATION"
    echo "==========================================================="
    if [[ -f "$DEPLOY_LOG" ]]; then
        tail -10 "$DEPLOY_LOG" | tac | while IFS=$'\t' read -r ts prev new status dur; do
            printf "%-20s %-10s %-10s %-16s %s\n" "$ts" "${prev:0:8}" "${new:0:8}" "$status" "$dur"
        done
    else
        echo "No deployment history found."
    fi
    exit 0
fi

# Deployment lock
if [[ -f "$DEPLOY_LOCK" ]]; then
    lock_pid=$(cat "$DEPLOY_LOCK" 2>/dev/null || echo "unknown")
    log "ERROR: Another deployment is running (PID: $lock_pid)"
    exit 1
fi

echo $$ > "$DEPLOY_LOCK"
trap 'rm -f "$DEPLOY_LOCK"' EXIT

cd "$COMPOSE_DIR"
CURRENT_SHA=$(git rev-parse HEAD)

# Determine target SHA
if [[ -z "$TARGET_SHA" ]]; then
    # Default: rollback to previous deploy from history
    if [[ ! -f "$DEPLOY_LOG" ]]; then
        log "ERROR: No deploy history found. Use --to <sha> to specify target."
        exit 1
    fi
    # Find the last successful deploy's "from" SHA
    TARGET_SHA=$(grep -E "success|rollback-success" "$DEPLOY_LOG" | tail -1 | cut -f2)
    if [[ -z "$TARGET_SHA" ]]; then
        log "ERROR: No successful deploy found in history. Use --to <sha> to specify target."
        exit 1
    fi
fi

log "=========================================="
log "  KMU Hub Production Rollback"
log "=========================================="
log "Current: $CURRENT_SHA"
log "Target:  $TARGET_SHA"

# Verify target SHA exists
if ! git cat-file -e "$TARGET_SHA" 2>/dev/null; then
    log "ERROR: Target SHA $TARGET_SHA does not exist in repository"
    exit 1
fi

# Step 1: Backup
log "Step 1/5: Creating backup..."
"$SCRIPT_DIR/backup.sh"

# Step 2: Checkout target
log "Step 2/5: Checking out $TARGET_SHA..."
git checkout "$TARGET_SHA"

# Step 3: Rebuild
log "Step 3/5: Building images..."
BUILD_VERSION=$(git describe --tags --always 2>/dev/null || echo "rollback-$TARGET_SHA")
BUILD_COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)

$COMPOSE build \
    --build-arg BUILD_VERSION="$BUILD_VERSION" \
    --build-arg BUILD_COMMIT="$BUILD_COMMIT" \
    --build-arg BUILD_TIME="$BUILD_TIME"

# Step 4: Restart services
log "Step 4/5: Restarting services..."
$COMPOSE up -d postgres redis minio
sleep 5

for svc in auth crm chat notification work email document biz automation plugin; do
    log "  Starting $svc..."
    $COMPOSE up -d "$svc"
    sleep 3
done

$COMPOSE up -d gateway caddy
sleep 5

# Step 5: Health check
log "Step 5/5: Running health checks..."
if [[ -f "$SCRIPT_DIR/healthcheck.sh" ]]; then
    if "$SCRIPT_DIR/healthcheck.sh"; then
        log_deploy "rollback-success" "$CURRENT_SHA" "$TARGET_SHA"
        log "=========================================="
        log "  Rollback complete!"
        log "  Now at: $BUILD_VERSION ($BUILD_COMMIT)"
        log "=========================================="
    else
        log_deploy "rollback-failed" "$CURRENT_SHA" "$TARGET_SHA"
        log "ERROR: Health check failed after rollback — manual intervention required"
        exit 1
    fi
else
    $COMPOSE ps
    log_deploy "rollback-success" "$CURRENT_SHA" "$TARGET_SHA"
    log "Rollback complete (no healthcheck script found)."
fi
