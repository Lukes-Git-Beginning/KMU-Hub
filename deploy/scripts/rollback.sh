#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEPLOY_LOCK="$COMPOSE_DIR/.deploy.lock"
DEPLOY_LOG="$COMPOSE_DIR/deploy-history.log"
ROLLBACK_START=$(date +%s)

TARGET_SHA=""
LIST_MODE=false
ALLOW_SCHEMA_AHEAD=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --to) TARGET_SHA="$2"; shift 2 ;;
        --to=*) TARGET_SHA="${1#*=}"; shift ;;
        --list) LIST_MODE=true; shift ;;
        --allow-schema-ahead) ALLOW_SCHEMA_AHEAD=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Mirror deploy.sh: the compose files live under deploy/docker and the env file
# must be passed explicitly, otherwise rolled-back containers start with empty
# secrets and crash-loop, and `-f $COMPOSE_DIR/docker-compose.yml` points at a
# path that does not exist.
COMPOSE_FILES_DIR="${COMPOSE_FILES_DIR:-$COMPOSE_DIR/deploy/docker}"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.production}"
COMPOSE="docker compose --env-file $ENV_FILE -f $COMPOSE_FILES_DIR/docker-compose.yml -f $COMPOSE_FILES_DIR/docker-compose.prod.yml"

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

# Migration guard.
#
# deploy.sh grew this after 2026-08-06, when an auto-rollback across an applied
# migration put the schema ahead of the code and cost 31 minutes of production
# 503 (see halt_without_rollback there). This script is the same operation run
# by hand and never got the guard — same failure mode, one command away.
#
# The mechanism: the migrate container of the older revision reads
# schema_migrations.version and finds no file of its own for it, so it refuses
# to start. Everything that waits on migrate stays down and the gateway fails
# closed. Going back across a migration is therefore not automatic — the down
# path can be destructive, so it stays a human decision.
SCHEMA_VERSION=$($COMPOSE exec -T postgres psql -U kmuhub -d kmuhub -tAc \
    "SELECT version FROM schema_migrations LIMIT 1" 2>/dev/null | tr -d '[:space:]' || true)

if [[ -z "$SCHEMA_VERSION" ]]; then
    log "ERROR: could not read schema_migrations.version from postgres."
    log "  Without it there is no way to tell whether $TARGET_SHA knows the current schema."
    log "  Fix the database connection first, or pass --allow-schema-ahead if you have"
    log "  checked by hand that no migration was applied since $TARGET_SHA."
    [[ "$ALLOW_SCHEMA_AHEAD" == true ]] || exit 1
else
    # Does the target revision carry a migration file for the version the DB is on?
    if git ls-tree --name-only "$TARGET_SHA" backend/migrations/ | grep -q "^backend/migrations/${SCHEMA_VERSION}_"; then
        log "Schema check: DB at $SCHEMA_VERSION, and $TARGET_SHA carries that migration. OK."
    else
        TARGET_HEAD=$(git ls-tree --name-only "$TARGET_SHA" backend/migrations/ \
            | grep -oE '[0-9]{6}' | sort | tail -1)
        log "REFUSING TO ROLL BACK: the schema is ahead of the target revision."
        log "  Database is at migration $SCHEMA_VERSION."
        log "  $TARGET_SHA has no file for it (its head is ${TARGET_HEAD:-unknown})."
        log "  Its migrate container would refuse to start and every service waiting on"
        log "  it would stay down — that is the 2026-08-06 outage, by hand."
        log ""
        log "  Roll the schema back deliberately first (migrate down to ${TARGET_HEAD:-<target head>}),"
        log "  then re-run. Or fix forward and deploy instead."
        log "  --allow-schema-ahead overrides this, and you own what happens next."
        if [[ "$ALLOW_SCHEMA_AHEAD" != true ]]; then
            log_deploy "rollback-refused-schema-ahead" "$CURRENT_SHA" "$TARGET_SHA"
            exit 1
        fi
        log "  --allow-schema-ahead given, continuing anyway."
    fi
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
#
# The service list used to be a second, hand-maintained copy that had already
# drifted: it carried livekit and livekit-egress, which deploy.sh does not
# build, and was missing `webapp` — the container that serves Cosmi in the
# browser since Etappe 2. A rollback would have left app.zentria.tech dark.
# It comes from deploy.sh now, the same way restore.sh takes it.
log "Step 4/5: Restarting services..."
BUILDABLE_LINE=$(grep -m1 '^BUILDABLE_SERVICES=' "$SCRIPT_DIR/deploy.sh") || {
    log "ERROR: could not find BUILDABLE_SERVICES in $SCRIPT_DIR/deploy.sh"
    exit 1
}
BUILDABLE_SERVICES=$(printf '%s' "$BUILDABLE_LINE" | cut -d'"' -f2)
if [[ -z "$BUILDABLE_SERVICES" ]]; then
    log "ERROR: BUILDABLE_SERVICES parsed empty from $SCRIPT_DIR/deploy.sh"
    exit 1
fi

$COMPOSE up -d postgres redis minio
sleep 5

for svc in $BUILDABLE_SERVICES; do
    # gateway and caddy come last, migrate is a run-to-completion job that the
    # dependent services already wait on.
    [[ "$svc" == "gateway" || "$svc" == "migrate" ]] && continue
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
