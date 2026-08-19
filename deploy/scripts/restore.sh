#!/usr/bin/env bash
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/kmuhub}"
BACKUP_DIR="${BACKUP_DIR:-$COMPOSE_DIR/backups}"
ENV_FILE="${ENV_FILE:-$COMPOSE_DIR/.env.production}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ASSUME_YES=false
SKIP_ROLES=false

log() { echo "[restore] $(date '+%Y-%m-%d %H:%M:%S') $*"; }

usage() {
    echo "Usage: $0 [--yes|-y] [--skip-roles] <pg_backup.dump.gz> [minio_backup.tar.gz]"
    echo ""
    echo "  --yes, -y      Skip the interactive 'type yes' confirmation."
    echo "  --skip-roles   Do not restore roles. Only when they already exist on"
    echo "                 this cluster -- without them the app cannot read the data."
    echo ""
    echo "Restoring a backup shipped offsite (deploy/scripts/backup.sh with"
    echo "BACKUP_OFFSITE_TARGET set)? Decrypt it first, it does not arrive here:"
    echo "  age -d -i <your-identity-file> -o pg_TS.dump.gz pg_TS.dump.gz.age"
    echo ""
    echo "Available local backups:"
    ls -lh "$BACKUP_DIR"/pg_*.dump.gz 2>/dev/null || echo "  No PostgreSQL backups found in $BACKUP_DIR"
}

ARGS=()
for arg in "$@"; do
    case "$arg" in
        --yes|-y) ASSUME_YES=true ;;
        --skip-roles) SKIP_ROLES=true ;;
        -h|--help) usage; exit 0 ;;
        *) ARGS+=("$arg") ;;
    esac
done

if [[ ${#ARGS[@]} -lt 1 ]]; then
    usage
    exit 1
fi

PG_BACKUP="${ARGS[0]}"
MINIO_BACKUP="${ARGS[1]:-}"

if [[ ! -f "$PG_BACKUP" ]]; then
    echo "ERROR: Backup file not found: $PG_BACKUP"
    exit 1
fi
if [[ -n "$MINIO_BACKUP" && ! -f "$MINIO_BACKUP" ]]; then
    echo "ERROR: MinIO backup file not found: $MINIO_BACKUP"
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

if [[ "$ASSUME_YES" == "false" ]]; then
    read -p "Type 'yes' to continue: " CONFIRM
    if [[ "$CONFIRM" != "yes" ]]; then
        echo "Aborted."
        exit 0
    fi
else
    log "--yes given, skipping confirmation prompt."
fi

# Same compose invocation as backup.sh: files live under deploy/docker/, and
# --env-file is required for ${VAR} interpolation (POSTGRES_PASSWORD etc.) to
# resolve inside the compose files at all.
compose() {
    docker compose --env-file "$ENV_FILE" \
        -f "$COMPOSE_DIR/deploy/docker/docker-compose.yml" \
        -f "$COMPOSE_DIR/deploy/docker/docker-compose.prod.yml" "$@"
}

# Stop list derived from deploy.sh's BUILDABLE_SERVICES (deploy.sh:192) instead
# of a second hardcoded copy that silently drifts as services get added — this
# script used to stop only 11 of the (now) 24 backend/cmd/* services, leaving
# the rest writing to the database while it got dropped and recreated.
# `migrate` is excluded: it is a run-to-completion job, not a long-running
# service to stop, and it gets run again below.
BUILDABLE_LINE=$(grep -m1 '^BUILDABLE_SERVICES=' "$SCRIPT_DIR/deploy.sh") || {
    echo "ERROR: could not find BUILDABLE_SERVICES in $SCRIPT_DIR/deploy.sh"
    exit 1
}
BUILDABLE_SERVICES=$(printf '%s' "$BUILDABLE_LINE" | cut -d'"' -f2)
if [[ -z "$BUILDABLE_SERVICES" ]]; then
    echo "ERROR: BUILDABLE_SERVICES parsed empty from $SCRIPT_DIR/deploy.sh"
    exit 1
fi
STOP_SERVICES=""
for svc in $BUILDABLE_SERVICES; do
    [[ "$svc" == "migrate" ]] && continue
    STOP_SERVICES="$STOP_SERVICES $svc"
done

log "Stopping application services:$STOP_SERVICES"
# shellcheck disable=SC2086 # intentional word-splitting of a service-name list
compose stop $STOP_SERVICES

# Roles first: they are cluster-level, pg_dump does not carry them, and every
# GRANT in the data dump targets them. Restoring onto a fresh server without
# them was measured on 2026-08-19 -- 710 failed grants for kmuhub_app and
# cosmi_agent_readonly, and every app service connects as kmuhub_app. The data
# would have been there and unreadable.
PG_DIR="$(cd "$(dirname "$PG_BACKUP")" && pwd)"
ROLES_BACKUP="$PG_DIR/$(basename "$PG_BACKUP" | sed 's/^pg_/roles_/; s/\.dump\.gz$/.sql.gz/')"

if [[ "$SKIP_ROLES" == true ]]; then
    log "WARNING: --skip-roles given. Grants for kmuhub_app will fail and the app will not be able to read this database until the roles exist."
elif [[ ! -f "$ROLES_BACKUP" ]]; then
    echo "ERROR: no role dump next to the backup: $ROLES_BACKUP"
    echo ""
    echo "Backups taken before 2026-08-19 have none — backup.sh did not write one yet."
    echo "Restoring without it yields a database the application cannot read."
    echo ""
    echo "Either create the roles on the target first, or re-run with --skip-roles"
    echo "if you know they already exist on this cluster."
    exit 1
else
    log "Restoring roles from $ROLES_BACKUP..."
    # "already exists" is the expected case when restoring onto the same
    # cluster; anything else is a real failure and must not pass silently.
    ROLE_ERRORS=$(gunzip -c "$ROLES_BACKUP" \
        | compose exec -T postgres psql -U kmuhub -d kmuhub -v ON_ERROR_STOP=0 2>&1 \
        | grep -i "^ERROR" | grep -vi "already exists" || true)
    if [[ -n "$ROLE_ERRORS" ]]; then
        log "ERROR: restoring roles failed:"
        echo "$ROLE_ERRORS"
        exit 1
    fi
    log "Roles restore OK."
fi

log "Restoring PostgreSQL..."
# Real error handling: a failed pg_restore used to be swallowed by
# `|| { echo WARNING }`, which reported success even on actual corruption.
# This is intentionally strict — it will also flag the harmless ownership
# warnings pg_restore --if-exists still prints on a truly first restore into
# an empty database. Rerun with a scratch DB once to see if that is the case;
# do not silence this again to work around it.
if ! gunzip -c "$PG_BACKUP" | compose exec -T postgres pg_restore -U kmuhub -d kmuhub --clean --if-exists -Fc; then
    log "ERROR: pg_restore failed — database may be partially restored. Do NOT restart services yet; inspect manually first."
    exit 1
fi
log "PostgreSQL restore OK."

if [[ -n "$MINIO_BACKUP" ]]; then
    log "Restoring MinIO data..."
    # Mirrors backup.sh's sidecar approach: the minio image ships no tar, so
    # `compose exec -T minio tar xzf -` can never work. Extraction happens
    # through a throwaway alpine container mounting the same named volume.
    MINIO_CID=$(compose ps -q minio 2>/dev/null || true)
    if [[ -z "$MINIO_CID" ]]; then
        log "ERROR: MinIO container not running, cannot resolve its volume — file storage NOT restored"
        exit 1
    fi
    # Same volume-name resolution as backup.sh: read it from the running
    # container instead of hardcoding "docker_minio_data", which carries the
    # compose project prefix and would drift if the project dir is renamed.
    MINIO_VOLUME=$(docker inspect -f \
        '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}}' "$MINIO_CID" 2>/dev/null || true)
    if [[ -z "$MINIO_VOLUME" ]]; then
        log "ERROR: could not resolve the MinIO /data volume — file storage NOT restored"
        exit 1
    fi
    # backup.sh archived with `-C / data` (so members are prefixed "data/"),
    # extracting with `-C /` here reproduces that back onto the same mount.
    if ! gunzip -c "$MINIO_BACKUP" | docker run --rm -i -v "$MINIO_VOLUME":/data alpine tar xzf - -C /; then
        log "ERROR: MinIO restore failed — file storage NOT restored"
        exit 1
    fi
    log "MinIO restore OK."
fi

# The backup can be older than the checked-out code. Without this, the
# schema stays at whatever version the dump carried, and every service that
# expects a newer column/table fails at the first query.
log "Running migrations (schema must match the restored data)..."
if ! compose run --rm migrate; then
    log "ERROR: migrate failed after restore — schema may not match the code. Investigate before restarting services."
    exit 1
fi

log "Restarting services..."
compose up -d

log "Waiting for health checks..."
sleep 10
compose ps

log "Verifying restore..."
SCHEMA_VERSION=$(compose exec -T postgres psql -U kmuhub -d kmuhub -tAc \
    'SELECT version FROM schema_migrations' 2>/dev/null | tr -d '[:space:]' || echo "unknown")
CONTACT_COUNT=$(compose exec -T postgres psql -U kmuhub -d kmuhub -tAc \
    'SELECT count(*) FROM contacts' 2>/dev/null | tr -d '[:space:]' || echo "unknown")
log "Schema migration version: $SCHEMA_VERSION"
log "contacts row count: $CONTACT_COUNT"
if [[ "$SCHEMA_VERSION" == "unknown" || "$CONTACT_COUNT" == "unknown" ]]; then
    log "WARNING: could not read verification query results — check the database manually."
fi

log "Restore complete."
