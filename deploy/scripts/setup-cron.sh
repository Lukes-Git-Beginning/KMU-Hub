#!/usr/bin/env bash
set -euo pipefail

DEPLOY_USER="${DEPLOY_USER:-deploy}"
LOG_DIR="${LOG_DIR:-/opt/kmuhub/logs}"
BACKUP_SCRIPT="${BACKUP_SCRIPT:-/opt/kmuhub/deploy/scripts/backup.sh}"

# Warn if not root (crontab -u requires root)
if [[ $EUID -ne 0 ]]; then
  echo "[setup-cron] WARNING: not running as root; crontab -u may fail unless you are the deploy user"
fi

# Verify backup script exists
if [[ ! -f "$BACKUP_SCRIPT" ]]; then
  echo "[setup-cron] ERROR: Backup script not found at $BACKUP_SCRIPT" >&2
  exit 1
fi

# Create log directory
mkdir -p "$LOG_DIR"
if [[ $EUID -eq 0 ]]; then
  chown "$DEPLOY_USER:$DEPLOY_USER" "$LOG_DIR"
else
  echo "[setup-cron] NOTE: skipping chown (not root); ensure $DEPLOY_USER can write to $LOG_DIR"
fi

# Build cron entry
CRON_ENTRY="0 2 * * * $BACKUP_SCRIPT >> $LOG_DIR/backup.log 2>&1  # kmuhub backup"

# Install cron entry idempotently
if [[ "$(whoami)" == "$DEPLOY_USER" ]]; then
  # Running as deploy user — no -u flag needed
  CURRENT="$(crontab -l 2>/dev/null || true)"
  NEW="$(printf '%s\n' "$CURRENT" | grep -v '# kmuhub backup' || true)"
  printf '%s\n%s\n' "$NEW" "$CRON_ENTRY" | crontab -
else
  CURRENT="$(crontab -u "$DEPLOY_USER" -l 2>/dev/null || true)"
  NEW="$(printf '%s\n' "$CURRENT" | grep -v '# kmuhub backup' || true)"
  printf '%s\n%s\n' "$NEW" "$CRON_ENTRY" | crontab -u "$DEPLOY_USER" -
fi

echo "[setup-cron] backup cron installed for $DEPLOY_USER (logs: $LOG_DIR/backup.log)"
