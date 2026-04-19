#!/usr/bin/env bash
# =============================================================================
# setup.sh — coturn setup for Zentria TURN server (CAX11, Ubuntu 24.04)
# =============================================================================
# PURPOSE: Install and configure coturn on the dedicated TURN server.
#          Run via deploy/turn/deploy.sh from the local machine, or manually:
#            ssh root@<ip> 'bash /tmp/setup.sh --ip <ip> --secret <secret>'
#
# IDEMPOTENT: Safe to run multiple times.
#   - apt install is idempotent
#   - /etc/turnserver.conf is regenerated from template on every run
#   - coturn service is re-enabled and restarted
#   - UFW rules are added if not already present
#
# TLS NOTE: This MVP setup uses plain TURN (port 3478, no TLS).
#   Let's Encrypt will be added in a follow-up (Sprint 1 post-MVP).
#   When TLS is added: uncomment the tls-listening-port and cert/pkey lines
#   in the generated config, open port 5349 in UFW and Hetzner firewall.
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
EXTERNAL_IP=""
TURN_SECRET=""
REALM="zentria.tech"

usage() {
    echo "Usage: $0 --ip <public-ipv4> --secret <hex-secret> [--realm <realm>]"
    echo ""
    echo "  --ip      Public IPv4 of this server (used for relay-ip / external-ip)"
    echo "  --secret  Shared TURN secret (must match TURN_SECRET in .env.production)"
    echo "  --realm   TURN realm (default: zentria.tech)"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --ip)     EXTERNAL_IP="$2"; shift 2 ;;
        --secret) TURN_SECRET="$2"; shift 2 ;;
        --realm)  REALM="$2"; shift 2 ;;
        -h|--help) usage ;;
        *) echo "[setup] Unknown argument: $1"; usage ;;
    esac
done

if [[ -z "${EXTERNAL_IP}" ]]; then
    echo "[setup] ERROR: --ip is required"
    usage
fi
if [[ -z "${TURN_SECRET}" ]]; then
    echo "[setup] ERROR: --secret is required"
    usage
fi

echo "[setup] Configuration:"
echo "  External IP : ${EXTERNAL_IP}"
echo "  Realm       : ${REALM}"
echo "  Secret      : <REDACTED>"

# ---------------------------------------------------------------------------
# 1. System update & install coturn
# ---------------------------------------------------------------------------
echo "[setup] Updating package index and installing coturn..."
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    coturn \
    ufw \
    openssl \
    curl

# ---------------------------------------------------------------------------
# 2. Firewall (UFW)
# ---------------------------------------------------------------------------
echo "[setup] Configuring UFW firewall..."

ufw --force enable

# SSH — keep open first (prevent lockout)
ufw allow 22/tcp comment "SSH"

# STUN/TURN plain (UDP + TCP) — MVP
ufw allow 3478/tcp comment "TURN plain TCP"
ufw allow 3478/udp comment "TURN plain UDP"

# ICE relay range
ufw allow 49152:65535/udp comment "TURN ICE relay UDP"

echo "[setup] UFW status:"
ufw status verbose

# ---------------------------------------------------------------------------
# 3. Generate /etc/turnserver.conf from template
# ---------------------------------------------------------------------------
echo "[setup] Writing /etc/turnserver.conf..."

# Template is uploaded to /tmp/turnserver.conf.template by deploy.sh
TEMPLATE_PATH="/tmp/turnserver.conf.template"

if [[ ! -f "${TEMPLATE_PATH}" ]]; then
    echo "[setup] ERROR: Template not found at ${TEMPLATE_PATH}"
    echo "  Make sure deploy/turn/deploy.sh uploaded it via scp."
    exit 1
fi

# Substitute placeholders — sed delimiters use | to avoid issues with IP dots
sed \
    -e "s|{{EXTERNAL_IP}}|${EXTERNAL_IP}|g" \
    -e "s|{{REALM}}|${REALM}|g" \
    -e "s|{{STATIC_AUTH_SECRET}}|${TURN_SECRET}|g" \
    "${TEMPLATE_PATH}" > /etc/turnserver.conf

chmod 640 /etc/turnserver.conf
chown root:turnserver /etc/turnserver.conf 2>/dev/null || chown root:root /etc/turnserver.conf

echo "[setup] /etc/turnserver.conf written."

# ---------------------------------------------------------------------------
# 4. Enable coturn (Ubuntu package requires TURNSERVER_ENABLED=1)
# ---------------------------------------------------------------------------
echo "[setup] Enabling coturn service..."

# Ubuntu's coturn init script checks /etc/default/coturn
if grep -q '^TURNSERVER_ENABLED=' /etc/default/coturn 2>/dev/null; then
    sed -i 's/^TURNSERVER_ENABLED=.*/TURNSERVER_ENABLED=1/' /etc/default/coturn
else
    echo 'TURNSERVER_ENABLED=1' >> /etc/default/coturn
fi

systemctl daemon-reload
systemctl enable coturn
systemctl restart coturn

# ---------------------------------------------------------------------------
# 5. Health check
# ---------------------------------------------------------------------------
echo "[setup] Waiting for coturn to start..."
sleep 3

if systemctl is-active --quiet coturn; then
    echo "[setup] coturn is RUNNING"
else
    echo "[setup] ERROR: coturn failed to start."
    echo "  Check logs: journalctl -u coturn -n 50"
    exit 1
fi

echo "[setup] Verifying port 3478 is listening..."
if ss -tulpen 2>/dev/null | grep ':3478' | grep -q 'coturn\|turnserver\|udp\|tcp'; then
    echo "[setup] Port 3478 is open."
else
    # ss output varies — check without process filter as fallback
    ss -tulpen | grep ':3478' || echo "[setup] WARNING: port 3478 not visible in ss output yet (may take a moment)."
fi

# ---------------------------------------------------------------------------
# 6. Summary
# ---------------------------------------------------------------------------
echo ""
echo "============================================================"
echo "  coturn setup complete (MVP — plain TURN, no TLS)"
echo "============================================================"
echo "  STUN/TURN URL : turn:turn.zentria.tech:3478"
echo "  Relay range   : 49152–65535 UDP"
echo ""
echo "Next steps:"
echo "  1. Set in .env.production on app server (178.104.38.195):"
echo "       TURN_SECRET=<the secret you provided>"
echo "       COTURN_HOST=turn.zentria.tech"
echo "  2. On app server: uncomment livekit-turn.yaml in docker-compose.prod.yml"
echo "  3. Restart LiveKit: sudo docker compose restart livekit"
echo "  4. Smoke-test: https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/"
echo "============================================================"
