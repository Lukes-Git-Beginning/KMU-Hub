#!/usr/bin/env bash
# Renders templated config files from environment variables.
#
# Currently handles:
#   - deploy/docker/livekit-secrets.yaml.tmpl -> livekit-secrets.yaml
#   - deploy/docker/egress.yaml.tmpl          -> egress-secrets.yaml
#   - deploy/docker/alertmanager.yml.tmpl    -> alertmanager.yml
#
# Required env vars (whitelist passed to envsubst):
#   LIVEKIT_API_KEY, LIVEKIT_API_SECRET, MINIO_ACCESS_KEY, MINIO_SECRET_KEY
#
# Optional env vars (allowed to be empty):
#   ALERT_WEBHOOK_URL — Slack-format webhook URL for Alertmanager.
#                      We use Discord's slack-compatible endpoint
#                      (https://discord.com/api/webhooks/<id>/<token>/slack).
#                      Empty means Alertmanager runs without notifications.
#
# Run before `docker compose up` so mounted overlay files exist.
# Idempotent — safe to run multiple times.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_DIR="$REPO_ROOT/deploy/docker"

log() { echo "[render-configs] $*"; }

# Verify envsubst is available
if ! command -v envsubst >/dev/null 2>&1; then
  echo "ERROR: envsubst not found. Install with: apt install gettext-base" >&2
  exit 1
fi

# Verify required env vars are set (and non-empty)
REQUIRED_VARS=(LIVEKIT_API_KEY LIVEKIT_API_SECRET MINIO_ACCESS_KEY MINIO_SECRET_KEY)
MISSING=()
for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var:-}" ]; then
    MISSING+=("$var")
  fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
  echo "ERROR: Required environment variables missing or empty: ${MISSING[*]}" >&2
  echo "       Source .env.production before running, or pass via env." >&2
  exit 1
fi

# Render livekit-secrets.yaml from template
TMPL="$DOCKER_DIR/livekit-secrets.yaml.tmpl"
OUT="$DOCKER_DIR/livekit-secrets.yaml"

if [ ! -f "$TMPL" ]; then
  echo "ERROR: Template not found: $TMPL" >&2
  exit 1
fi

log "Rendering $TMPL -> $OUT"
# Whitelist mode: only substitute the listed vars; other ${...} stay literal.
envsubst '$LIVEKIT_API_KEY $LIVEKIT_API_SECRET' < "$TMPL" > "$OUT"

# Quick sanity check: rendered file must not contain bare '${' anymore.
if grep -q '\${LIVEKIT' "$OUT"; then
  echo "ERROR: Unresolved placeholder in $OUT — env-var substitution failed" >&2
  exit 1
fi

# Render egress-secrets.yaml from template. The prod overlay mounts this over
# /etc/egress.yaml so the egress worker signs with the real LiveKit key and
# uploads recordings with the real MinIO credentials (dev keeps egress.yaml).
EGRESS_TMPL="$DOCKER_DIR/egress.yaml.tmpl"
EGRESS_OUT="$DOCKER_DIR/egress-secrets.yaml"

if [ ! -f "$EGRESS_TMPL" ]; then
  echo "ERROR: Template not found: $EGRESS_TMPL" >&2
  exit 1
fi

log "Rendering $EGRESS_TMPL -> $EGRESS_OUT"
envsubst '$LIVEKIT_API_KEY $LIVEKIT_API_SECRET $MINIO_ACCESS_KEY $MINIO_SECRET_KEY' < "$EGRESS_TMPL" > "$EGRESS_OUT"

if grep -qE '\$\{(LIVEKIT|MINIO)' "$EGRESS_OUT"; then
  echo "ERROR: Unresolved placeholder in $EGRESS_OUT — env-var substitution failed" >&2
  exit 1
fi

# Render alertmanager.yml from template (ALERT_WEBHOOK_URL is optional).
# Empty value yields `slack_api_url: ""` which Alertmanager accepts —
# alerts simply aren't sent until the webhook is provided.
ALERT_TMPL="$DOCKER_DIR/alertmanager.yml.tmpl"
ALERT_OUT="$DOCKER_DIR/alertmanager.yml"

if [ ! -f "$ALERT_TMPL" ]; then
  echo "ERROR: Template not found: $ALERT_TMPL" >&2
  exit 1
fi

log "Rendering $ALERT_TMPL -> $ALERT_OUT"
ALERT_WEBHOOK_URL="${ALERT_WEBHOOK_URL:-}" envsubst '$ALERT_WEBHOOK_URL' < "$ALERT_TMPL" > "$ALERT_OUT"

if grep -q '\${ALERT_WEBHOOK_URL' "$ALERT_OUT"; then
  echo "ERROR: Unresolved placeholder in $ALERT_OUT — env-var substitution failed" >&2
  exit 1
fi

if [ -z "${ALERT_WEBHOOK_URL:-}" ]; then
  log "WARNING: ALERT_WEBHOOK_URL empty — Alertmanager will run without notifications"
fi

log "Done."
