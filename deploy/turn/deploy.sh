#!/usr/bin/env bash
# =============================================================================
# deploy.sh — Local wrapper to deploy coturn onto the Hetzner CAX11
# =============================================================================
# Run this on your LOCAL machine (Windows Git Bash / macOS / Linux).
# It uploads setup.sh + turnserver.conf.template via scp, then runs setup.sh
# over SSH.
#
# Usage:
#   bash deploy/turn/deploy.sh --host <public-ip> --secret <hex-secret> [--realm zentria.tech]
#
# Example:
#   bash deploy/turn/deploy.sh --host 1.2.3.4 --secret $(openssl rand -hex 32)
#
# PRE-REQUISITES:
#   - Hetzner CAX11 provisioned and reachable at <public-ip>
#   - SSH key ~/.ssh/hetzner_kmuhub has access (root@<ip>)
#   - DNS A-record turn.zentria.tech -> <public-ip> (propagation not required
#     for this script, but required for LiveKit to connect)
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
SSH_KEY="${SSH_KEY:-$HOME/.ssh/hetzner_kmuhub}"
REALM="zentria.tech"
HOST=""
SECRET=""

# Script directory (works in Git Bash on Windows)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
usage() {
    echo "Usage: $0 --host <public-ipv4> --secret <hex-secret> [--realm <realm>] [--key <ssh-key-path>]"
    echo ""
    echo "  --host    Public IPv4 of the CAX11 TURN server"
    echo "  --secret  Shared TURN secret (32-byte hex, e.g. from: openssl rand -hex 32)"
    echo "  --realm   TURN realm (default: zentria.tech)"
    echo "  --key     Path to SSH private key (default: ~/.ssh/hetzner_kmuhub)"
    echo ""
    echo "IMPORTANT: Store the --secret value in a password manager and set it as"
    echo "  TURN_SECRET in .env.production on the app server (178.104.38.195)."
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --host)   HOST="$2";   shift 2 ;;
        --secret) SECRET="$2"; shift 2 ;;
        --realm)  REALM="$2";  shift 2 ;;
        --key)    SSH_KEY="$2"; shift 2 ;;
        -h|--help) usage ;;
        *) echo "[deploy] Unknown argument: $1"; usage ;;
    esac
done

# ---------------------------------------------------------------------------
# Validation
# ---------------------------------------------------------------------------
if [[ -z "${HOST}" ]]; then
    echo "[deploy] ERROR: --host is required"
    usage
fi
if [[ -z "${SECRET}" ]]; then
    echo "[deploy] ERROR: --secret is required"
    echo "[deploy] Generate one with: openssl rand -hex 32"
    usage
fi
if [[ ! -f "${SSH_KEY}" ]]; then
    echo "[deploy] ERROR: SSH key not found: ${SSH_KEY}"
    echo "[deploy] Use --key to specify a different path."
    exit 1
fi

echo "[deploy] Target host : ${HOST}"
echo "[deploy] SSH key     : ${SSH_KEY}"
echo "[deploy] Realm       : ${REALM}"
echo "[deploy] Secret      : <REDACTED>"

# ---------------------------------------------------------------------------
# Upload artifacts
# ---------------------------------------------------------------------------
echo ""
echo "[deploy] Uploading setup.sh and turnserver.conf.template to /tmp/..."

scp -i "${SSH_KEY}" \
    -o StrictHostKeyChecking=accept-new \
    "${SCRIPT_DIR}/setup.sh" \
    "${SCRIPT_DIR}/turnserver.conf.template" \
    "root@${HOST}:/tmp/"

echo "[deploy] Upload complete."

# ---------------------------------------------------------------------------
# Run setup on remote
# ---------------------------------------------------------------------------
echo ""
echo "[deploy] Running setup.sh on ${HOST}..."
echo "--------------------------------------------------------------"

ssh -i "${SSH_KEY}" \
    -o StrictHostKeyChecking=accept-new \
    "root@${HOST}" \
    "bash /tmp/setup.sh --ip ${HOST} --secret ${SECRET} --realm ${REALM}"

echo "--------------------------------------------------------------"
echo ""
echo "[deploy] Setup complete on ${HOST}."
echo ""
echo "========================================================================"
echo "  Next steps (manual)"
echo "========================================================================"
echo "  1. Set on app server (178.104.38.195):"
echo "       TURN_SECRET=<your-secret>     # add to .env.production"
echo "       COTURN_HOST=turn.zentria.tech"
echo ""
echo "  2. Uncomment livekit-turn.yaml volume in docker-compose.prod.yml"
echo "     (see deploy/turn/livekit-integration.md for exact diff)"
echo ""
echo "  3. Restart LiveKit:"
echo "       ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195"
echo "       cd /opt/kmuhub/deploy/docker"
echo "       sudo docker compose -f docker-compose.yml -f docker-compose.prod.yml restart livekit"
echo ""
echo "  4. Smoke-test TURN:"
echo "       https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/"
echo "       Server: turn:turn.zentria.tech:3478"
echo "       Username/Credential: LiveKit-generated (or use turnutils_uclient)"
echo "========================================================================"
