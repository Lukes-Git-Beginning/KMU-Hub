#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
BACKUP_DIR="${BACKUP_DIR:-/opt/kmuhub/backups}"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.production}"
RETENTION_DAYS=30
MIN_BACKUPS=7

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_PREFIX="[backup]"

log() { echo "$LOG_PREFIX $(date '+%Y-%m-%d %H:%M:%S') $*"; }

# Reads a single KEY=value out of .env.production without sourcing the whole
# file (which would export every production secret — DB/JWT/SMTP passwords —
# into this process and everything it shells out to, for the sake of three
# optional settings). Strips one layer of surrounding quotes to match the
# `KEY="multi word value"` convention used there.
read_env_var() {
    local key="$1" val
    [[ -f "$ENV_FILE" ]] || return 0
    val=$(grep -m1 "^${key}=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2-)
    val="${val%\"}"
    val="${val#\"}"
    printf '%s' "$val"
}

# Optional failure alert. Reuses the same Discord/Slack-compatible webhook
# Alertmanager already sends to (ALERT_WEBHOOK_URL, see alertmanager.yml.tmpl
# and PRODUCTION_TEMPLATE) unless a dedicated BACKUP_ALERT_WEBHOOK is set.
# Empty in both places (the default) -> no alert is ever attempted.
ALERT_WEBHOOK="${BACKUP_ALERT_WEBHOOK:-$(read_env_var BACKUP_ALERT_WEBHOOK)}"
if [[ -z "$ALERT_WEBHOOK" ]]; then
    ALERT_WEBHOOK="$(read_env_var ALERT_WEBHOOK_URL)"
fi

# Optional offsite shipment. Both must be set or neither is used — see below.
OFFSITE_TARGET="${BACKUP_OFFSITE_TARGET:-$(read_env_var BACKUP_OFFSITE_TARGET)}"
AGE_RECIPIENT="${BACKUP_AGE_RECIPIENT:-$(read_env_var BACKUP_AGE_RECIPIENT)}"

send_alert() {
    local message="$1" escaped
    [[ -z "$ALERT_WEBHOOK" ]] && return 0
    command -v curl &> /dev/null || { log "WARNING: curl not found, cannot send failure alert"; return 0; }
    escaped=$(printf '%s' "$message" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')
    curl -sf -m 10 -X POST -H 'Content-Type: application/json' \
        -d "{\"text\":\"[kmuhub backup] ${escaped}\"}" \
        "$ALERT_WEBHOOK" > /dev/null 2>&1 || log "WARNING: failed to deliver backup alert webhook"
}

# Catches hard failures (pg_dump/gzip erroring under `set -e`) that would
# otherwise exit this script silently before reaching the status summary below.
on_exit() {
    local exit_code=$?
    if [[ "$exit_code" -ne 0 ]]; then
        log "Backup script exiting with code $exit_code"
        send_alert "backup.sh failed with exit code $exit_code on $(hostname 2>/dev/null || echo unknown-host)"
    fi
}
trap on_exit EXIT

log "Starting backup..."
mkdir -p "$BACKUP_DIR"

compose() {
    docker compose --env-file "$ENV_FILE" \
        -f "$COMPOSE_DIR/deploy/docker/docker-compose.yml" \
        -f "$COMPOSE_DIR/deploy/docker/docker-compose.prod.yml" "$@"
}

# PostgreSQL backup
PG_FILE="$BACKUP_DIR/pg_${TIMESTAMP}.dump.gz"
log "Dumping PostgreSQL..."
compose exec -T postgres pg_dump -U kmuhub -Fc kmuhub | gzip > "$PG_FILE"
log "PostgreSQL backup: $PG_FILE ($(du -h "$PG_FILE" | cut -f1))"

# Roles are cluster-level and pg_dump does not carry them. Without this file a
# restore onto a fresh server replays every GRANT against roles that do not
# exist -- verified on 2026-08-19 against a scratch instance: the data came
# back, and 710 grants for kmuhub_app and cosmi_agent_readonly failed. Every
# app service connects as kmuhub_app, so that restore would have produced a
# database nothing could read.
#
# Contains role password hashes: treat this file exactly like the data dump.
ROLES_FILE="$BACKUP_DIR/roles_${TIMESTAMP}.sql.gz"
compose exec -T postgres pg_dumpall -U kmuhub --roles-only | gzip > "$ROLES_FILE"
log "Role backup: $ROLES_FILE ($(du -h "$ROLES_FILE" | cut -f1))"

# MinIO backup (volume snapshot via a sidecar).
#
# This used to run `compose exec -T minio tar czf - /data`, which could never
# work: the official minio image ships no tar binary, so every run failed,
# logged "non-critical" and deleted the empty file. The result was that chat
# attachments, documents/WOPI files, mail attachments, report photos and
# recordings had no backup at all for as long as the script existed — only the
# database rows pointing at them.
#
# Mounting the volume into a throwaway alpine reads the same bytes without
# depending on what the minio image contains.
MINIO_FILE="$BACKUP_DIR/minio_${TIMESTAMP}.tar.gz"
MINIO_OK=false
log "Backing up MinIO data..."

MINIO_CID=$(compose ps -q minio 2>/dev/null || true)
if [[ -z "$MINIO_CID" ]]; then
    log "ERROR: MinIO container not running — FILE STORAGE IS NOT BACKED UP"
else
    # Resolve the volume from the running container rather than hardcoding
    # "docker_minio_data": the name carries the compose project prefix and
    # would silently drift if the project directory is ever renamed.
    MINIO_VOLUME=$(docker inspect -f \
        '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}}' "$MINIO_CID" 2>/dev/null || true)
    if [[ -z "$MINIO_VOLUME" ]]; then
        log "ERROR: could not resolve the MinIO /data volume — FILE STORAGE IS NOT BACKED UP"
    elif docker run --rm -v "$MINIO_VOLUME":/data:ro alpine tar czf - -C / data > "$MINIO_FILE" 2>/dev/null; then
        # An empty gzip stream is ~20 bytes, so a "successful" run that wrote
        # nothing is still a failure — that is exactly how the old bug stayed
        # invisible.
        MINIO_SIZE=$(stat -c%s "$MINIO_FILE" 2>/dev/null || echo 0)
        if [[ "$MINIO_SIZE" -lt 100 ]]; then
            log "ERROR: MinIO archive is empty (${MINIO_SIZE} bytes) — FILE STORAGE IS NOT BACKED UP"
            rm -f "$MINIO_FILE"
        else
            MINIO_OK=true
            log "MinIO backup: $MINIO_FILE ($(du -h "$MINIO_FILE" | cut -f1))"
        fi
    else
        log "ERROR: MinIO backup failed — FILE STORAGE IS NOT BACKED UP"
        rm -f "$MINIO_FILE"
    fi
fi

# Offsite backup (optional, encrypted).
#
# Local backups sit in $BACKUP_DIR on the same host they protect against —
# useless if that host's disk dies or the provider has an outage. This ships
# an encrypted copy elsewhere. It only activates when BOTH BACKUP_OFFSITE_TARGET
# (an rsync destination: a local/mounted path, or user@host:/path for a remote
# host) and BACKUP_AGE_RECIPIENT (one or more space-separated age public keys,
# `age-keygen` output) are configured. Neither set (the default) -> this block
# is a no-op and the script behaves exactly as before.
OFFSITE_OK=true
if [[ -n "$OFFSITE_TARGET" || -n "$AGE_RECIPIENT" ]]; then
    if [[ -z "$OFFSITE_TARGET" || -z "$AGE_RECIPIENT" ]]; then
        log "ERROR: offsite backup needs BOTH BACKUP_OFFSITE_TARGET and BACKUP_AGE_RECIPIENT — only one is set. Skipping offsite (local backup unaffected)."
        OFFSITE_OK=false
    elif ! command -v age &> /dev/null; then
        log "ERROR: age not installed — cannot encrypt for offsite transfer. Install age (https://github.com/FiloSottile/age); refusing to ship backups unencrypted."
        OFFSITE_OK=false
    elif ! command -v rsync &> /dev/null; then
        log "ERROR: rsync not installed — cannot ship to BACKUP_OFFSITE_TARGET."
        OFFSITE_OK=false
    else
        log "Shipping encrypted backup to $OFFSITE_TARGET..."
        AGE_RECIPIENT_ARGS=()
        for r in $AGE_RECIPIENT; do
            AGE_RECIPIENT_ARGS+=(-r "$r")
        done

        OFFSITE_FILES=("$PG_FILE")
        [[ "$MINIO_OK" == "true" ]] && OFFSITE_FILES+=("$MINIO_FILE")

        for f in "${OFFSITE_FILES[@]}"; do
            if age "${AGE_RECIPIENT_ARGS[@]}" -o "${f}.age" "$f" && rsync -a "${f}.age" "$OFFSITE_TARGET/"; then
                log "Offsite: $(basename "${f}.age") shipped to $OFFSITE_TARGET"
            else
                log "ERROR: offsite transfer failed for $(basename "$f")"
                OFFSITE_OK=false
            fi
            rm -f "${f}.age"
        done
    fi
fi

# Rotation: delete old backups but keep minimum count
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "pg_*.dump.gz" -type f | wc -l)
if [[ "$BACKUP_COUNT" -gt "$MIN_BACKUPS" ]]; then
    find "$BACKUP_DIR" -name "pg_*.dump.gz" -type f -mtime +"$RETENTION_DAYS" -delete
    find "$BACKUP_DIR" -name "roles_*.sql.gz" -type f -mtime +"$RETENTION_DAYS" -delete
    find "$BACKUP_DIR" -name "minio_*.tar.gz" -type f -mtime +"$RETENTION_DAYS" -delete
    log "Cleaned backups older than $RETENTION_DAYS days"
fi

if [[ "$MINIO_OK" == "true" && "$OFFSITE_OK" == "true" ]]; then
    log "Backup complete."
else
    # lean: exits 0 even on a soft failure (MinIO or offsite), because
    # deploy.sh runs this as step 1/7 under `set -e` — a hard failure here
    # would abort every deploy over a backup problem, and that script already
    # has a rollback history. The messages above are the signal; send_alert
    # is now the one that actually surfaces it instead of relying on someone
    # reading cron logs.
    log "Backup complete WITH ERRORS — see above."
    send_alert "kmuhub backup on $(hostname 2>/dev/null || echo unknown-host) completed with errors — see backup.log for detail (MinIO ok=$MINIO_OK, offsite ok=$OFFSITE_OK)"
fi
