#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <pg_backup.dump.gz> [minio_backup.tar.gz]"
    echo ""
    echo "Available backups:"
    ls -lh /opt/kmuhub/backups/pg_*.dump.gz 2>/dev/null || echo "  No PostgreSQL backups found"
    exit 1
fi

PG_BACKUP="$1"
MINIO_BACKUP="${2:-}"

if [[ ! -f "$PG_BACKUP" ]]; then
    echo "ERROR: Backup file not found: $PG_BACKUP"
    exit 1
fi

echo "============================================"
echo "  KMU Hub Database Restore"
echo "============================================"
echo ""
echo "PostgreSQL backup: $PG_BACKUP"
[[ -n "$MINIO_BACKUP" ]] && echo "MinIO backup: $MINIO_BACKUP"
echo ""
echo "WARNING: This will DROP and recreate the database!"
echo ""
read -p "Type 'yes' to continue: " CONFIRM

if [[ "$CONFIRM" != "yes" ]]; then
    echo "Aborted."
    exit 0
fi

COMPOSE="docker compose -f $COMPOSE_DIR/docker-compose.yml -f $COMPOSE_DIR/docker-compose.prod.yml"

echo "Stopping application services..."
$COMPOSE stop gateway auth crm chat notification work email document biz automation plugin

echo "Restoring PostgreSQL..."
gunzip -c "$PG_BACKUP" | $COMPOSE exec -T postgres pg_restore -U kmuhub -d kmuhub --clean --if-exists -Fc 2>&1 || {
    echo "WARNING: pg_restore completed with warnings (normal for --clean on first restore)"
}

if [[ -n "$MINIO_BACKUP" && -f "$MINIO_BACKUP" ]]; then
    echo "Restoring MinIO data..."
    $COMPOSE exec -T minio tar xzf - -C / < "$MINIO_BACKUP"
fi

echo "Restarting services..."
$COMPOSE up -d

echo "Waiting for health checks..."
sleep 10
$COMPOSE ps

echo "Restore complete."
