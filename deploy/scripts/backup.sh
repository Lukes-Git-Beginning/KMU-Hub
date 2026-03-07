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

# PostgreSQL backup
PG_FILE="$BACKUP_DIR/pg_${TIMESTAMP}.dump.gz"
log "Dumping PostgreSQL..."
docker compose -f "$COMPOSE_DIR/docker-compose.yml" -f "$COMPOSE_DIR/docker-compose.prod.yml" \
    exec -T postgres pg_dump -U kmuhub -Fc kmuhub | gzip > "$PG_FILE"
log "PostgreSQL backup: $PG_FILE ($(du -h "$PG_FILE" | cut -f1))"

# MinIO backup (volume snapshot via tar)
MINIO_FILE="$BACKUP_DIR/minio_${TIMESTAMP}.tar.gz"
log "Backing up MinIO data..."
docker compose -f "$COMPOSE_DIR/docker-compose.yml" -f "$COMPOSE_DIR/docker-compose.prod.yml" \
    exec -T minio tar czf - /data 2>/dev/null > "$MINIO_FILE" || {
    log "WARNING: MinIO backup failed (non-critical)"
    rm -f "$MINIO_FILE"
}
if [[ -f "$MINIO_FILE" ]]; then
    log "MinIO backup: $MINIO_FILE ($(du -h "$MINIO_FILE" | cut -f1))"
fi

# Rotation: delete old backups but keep minimum count
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "pg_*.dump.gz" -type f | wc -l)
if [[ "$BACKUP_COUNT" -gt "$MIN_BACKUPS" ]]; then
    find "$BACKUP_DIR" -name "pg_*.dump.gz" -type f -mtime +"$RETENTION_DAYS" -delete
    find "$BACKUP_DIR" -name "minio_*.tar.gz" -type f -mtime +"$RETENTION_DAYS" -delete
    log "Cleaned backups older than $RETENTION_DAYS days"
fi

log "Backup complete."
