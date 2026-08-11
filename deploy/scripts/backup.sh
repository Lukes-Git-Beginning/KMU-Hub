#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
BACKUP_DIR="${BACKUP_DIR:-/opt/kmuhub/backups}"
RETENTION_DAYS=30
MIN_BACKUPS=7

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_PREFIX="[backup]"

log() { echo "$LOG_PREFIX $(date '+%Y-%m-%d %H:%M:%S') $*"; }

log "Starting backup..."
mkdir -p "$BACKUP_DIR"

compose() {
    docker compose --env-file "$COMPOSE_DIR/.env.production" \
        -f "$COMPOSE_DIR/deploy/docker/docker-compose.yml" \
        -f "$COMPOSE_DIR/deploy/docker/docker-compose.prod.yml" "$@"
}

# PostgreSQL backup
PG_FILE="$BACKUP_DIR/pg_${TIMESTAMP}.dump.gz"
log "Dumping PostgreSQL..."
compose exec -T postgres pg_dump -U kmuhub -Fc kmuhub | gzip > "$PG_FILE"
log "PostgreSQL backup: $PG_FILE ($(du -h "$PG_FILE" | cut -f1))"

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

# Rotation: delete old backups but keep minimum count
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "pg_*.dump.gz" -type f | wc -l)
if [[ "$BACKUP_COUNT" -gt "$MIN_BACKUPS" ]]; then
    find "$BACKUP_DIR" -name "pg_*.dump.gz" -type f -mtime +"$RETENTION_DAYS" -delete
    find "$BACKUP_DIR" -name "minio_*.tar.gz" -type f -mtime +"$RETENTION_DAYS" -delete
    log "Cleaned backups older than $RETENTION_DAYS days"
fi

if [[ "$MINIO_OK" == "true" ]]; then
    log "Backup complete."
else
    # lean: exits 0 even though MinIO failed, because deploy.sh runs this as
    # step 1/7 under `set -e` — a hard failure here would abort every deploy
    # over a backup problem, and that script already has a rollback history.
    # The message above is the signal. Upgrade to a non-zero exit (and an
    # explicit handler in deploy.sh) once backups are monitored.
    log "Backup complete WITH ERRORS — see above. Database is backed up, file storage is NOT."
fi
