#!/usr/bin/env bash
# =============================================================================
# coturn-setup.sh — Self-hosted TURN server setup for Zentria / Cosmi
# =============================================================================
# WHEN: Run once after the Hetzner CPX11 is provisioned and DNS is live.
# WHO:  Luke, via SSH into the coturn CPX11 (not the main CPX42 app server).
# HOW:  ssh root@<coturn-server-ip>
#         curl -fsSL https://raw.githubusercontent.com/.../coturn-setup.sh | bash
#       OR: scp this file, then bash coturn-setup.sh
#
# PRE-REQUISITES:
#   1. DNS A-record: turn.zentria.tech -> <this server's public IP>
#      Wait for propagation (check: dig +short turn.zentria.tech)
#   2. Ubuntu 24.04 LTS (CPX11 default)
#   3. Port 80 + 443 open for Let's Encrypt HTTP-01 challenge
#
# IDEMPOTENT: Safe to run multiple times (apt install, certbot, systemctl are
# all idempotent; config is regenerated from env vars each run).
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration — override via environment variables before running
# ---------------------------------------------------------------------------
DOMAIN="${DOMAIN:-turn.zentria.tech}"
REALM="${REALM:-zentria.tech}"
EMAIL="${EMAIL:-smashboy90@gmail.com}"
RELAY_MIN_PORT="${RELAY_MIN_PORT:-49152}"
RELAY_MAX_PORT="${RELAY_MAX_PORT:-65535}"

# TURN_SECRET: if set, uses that value; otherwise generates a random secret.
# After setup, copy the printed secret into .env.production on the app server.
if [[ -z "${TURN_SECRET:-}" ]]; then
    TURN_SECRET="$(openssl rand -hex 32)"
    echo "[coturn-setup] Generated TURN_SECRET (save this!): ${TURN_SECRET}"
else
    echo "[coturn-setup] Using provided TURN_SECRET."
fi

# ---------------------------------------------------------------------------
# 1. System update & install
# ---------------------------------------------------------------------------
echo "[coturn-setup] Installing dependencies..."
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    coturn \
    certbot \
    ufw \
    openssl \
    curl

# ---------------------------------------------------------------------------
# 2. Firewall (ufw)
# ---------------------------------------------------------------------------
echo "[coturn-setup] Configuring ufw firewall..."

ufw --force enable

# SSH — keep open (important: before any other rules)
ufw allow 22/tcp comment "SSH"

# TURN plain (UDP + TCP)
ufw allow 3478/tcp comment "TURN plain TCP"
ufw allow 3478/udp comment "TURN plain UDP"

# TURN TLS
ufw allow 5349/tcp comment "TURN TLS TCP"
ufw allow 5349/udp comment "TURN TLS UDP"

# Let's Encrypt HTTP-01 challenge
ufw allow 80/tcp comment "Let's Encrypt HTTP challenge"

# ICE relay range
ufw allow "${RELAY_MIN_PORT}:${RELAY_MAX_PORT}/udp" comment "TURN ICE relay UDP"

ufw status verbose

# ---------------------------------------------------------------------------
# 3. TLS Certificate (Let's Encrypt)
# ---------------------------------------------------------------------------
echo "[coturn-setup] Obtaining TLS certificate for ${DOMAIN}..."

# Stop coturn temporarily in case it's already running and holding port 80
systemctl stop coturn 2>/dev/null || true

# Certbot standalone — listens on port 80 for the challenge
certbot certonly \
    --standalone \
    --non-interactive \
    --agree-tos \
    --email "${EMAIL}" \
    --domains "${DOMAIN}" \
    --keep-until-expiring

CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
echo "[coturn-setup] Certificate obtained at ${CERT_DIR}"

# Allow coturn to read Let's Encrypt certs
# (coturn runs as turnserver user on Ubuntu)
chown -R root:turnserver /etc/letsencrypt/{live,archive} 2>/dev/null || true
chmod -R g+rx /etc/letsencrypt/{live,archive} 2>/dev/null || true

# Auto-renew hook: restart coturn after renewal
RENEW_HOOK="/etc/letsencrypt/renewal-hooks/deploy/restart-coturn.sh"
cat > "${RENEW_HOOK}" << 'HOOK'
#!/usr/bin/env bash
set -euo pipefail
# Reload coturn to pick up renewed certificate
systemctl reload-or-restart coturn
HOOK
chmod +x "${RENEW_HOOK}"

# ---------------------------------------------------------------------------
# 4. Detect external (public) IP
# ---------------------------------------------------------------------------
echo "[coturn-setup] Detecting external IP..."
EXTERNAL_IP="$(curl -fsSL --max-time 5 https://api.ipify.org || curl -fsSL --max-time 5 http://checkip.amazonaws.com || hostname -I | awk '{print $1}')"
echo "[coturn-setup] External IP: ${EXTERNAL_IP}"

# ---------------------------------------------------------------------------
# 5. Generate /etc/turnserver.conf
# ---------------------------------------------------------------------------
echo "[coturn-setup] Writing /etc/turnserver.conf..."

cat > /etc/turnserver.conf << CONF
# =============================================================================
# coturn configuration — managed by coturn-setup.sh
# Regenerated on: $(date -u '+%Y-%m-%dT%H:%M:%SZ')
# =============================================================================

# Network
listening-port=3478
tls-listening-port=5349
listening-ip=0.0.0.0
external-ip=${EXTERNAL_IP}
relay-ip=${EXTERNAL_IP}

# ICE relay port range — must match ufw rules and Hetzner firewall
min-port=${RELAY_MIN_PORT}
max-port=${RELAY_MAX_PORT}

# Identity
realm=${REALM}
server-name=${DOMAIN}

# RFC 5389 fingerprinting (required by WebRTC)
fingerprint

# REST API / shared-secret authentication mode (coturn HMAC-SHA1 time-limited credentials).
# LiveKit generates credentials automatically using this shared secret.
# Clients must NOT use static long-term credentials.
use-auth-secret
static-auth-secret=${TURN_SECRET}

# TLS — Let's Encrypt certificate
cert=${CERT_DIR}/fullchain.pem
pkey=${CERT_DIR}/privkey.pem

# Logging
log-file=/var/log/coturn.log
no-stdout-log
simple-log

# Security hardening
no-cli
no-loopback-peers
no-multicast-peers

# Block relay to RFC-1918 private address ranges (SSRF protection)
denied-peer-ip=0.0.0.0-0.255.255.255
denied-peer-ip=10.0.0.0-10.255.255.255
denied-peer-ip=100.64.0.0-100.127.255.255
denied-peer-ip=127.0.0.0-127.255.255.255
denied-peer-ip=169.254.0.0-169.254.255.255
denied-peer-ip=172.16.0.0-172.31.255.255
denied-peer-ip=192.0.0.0-192.0.0.255
denied-peer-ip=192.168.0.0-192.168.255.255
denied-peer-ip=198.18.0.0-198.19.255.255
denied-peer-ip=198.51.100.0-198.51.100.255
denied-peer-ip=203.0.113.0-203.0.113.255
denied-peer-ip=240.0.0.0-255.255.255.255
CONF

# ---------------------------------------------------------------------------
# 6. Enable & start coturn
# ---------------------------------------------------------------------------
echo "[coturn-setup] Enabling and starting coturn..."

# Ubuntu's coturn package requires TURNSERVER_ENABLED=1 in /etc/default/coturn
sed -i 's/^#\?TURNSERVER_ENABLED=.*/TURNSERVER_ENABLED=1/' /etc/default/coturn
grep -q '^TURNSERVER_ENABLED=1' /etc/default/coturn || echo 'TURNSERVER_ENABLED=1' >> /etc/default/coturn

systemctl daemon-reload
systemctl enable coturn
systemctl restart coturn

# ---------------------------------------------------------------------------
# 7. Verify
# ---------------------------------------------------------------------------
echo ""
echo "[coturn-setup] Waiting 3 seconds for coturn to start..."
sleep 3
systemctl is-active coturn && echo "[coturn-setup] coturn is RUNNING" || {
    echo "[coturn-setup] ERROR: coturn failed to start. Check: journalctl -u coturn -n 50"
    exit 1
}

# ---------------------------------------------------------------------------
# 8. Summary
# ---------------------------------------------------------------------------
echo ""
echo "============================================================"
echo "  coturn setup complete"
echo "============================================================"
echo "  TURN URL (TLS):  turns:${DOMAIN}:5349"
echo "  TURN URL (plain): turn:${DOMAIN}:3478"
echo ""
echo "  >>> TURN_SECRET: ${TURN_SECRET} <<<"
echo "  Copy this into .env.production on the app server:"
echo "    TURN_SECRET=${TURN_SECRET}"
echo "    COTURN_HOST=${DOMAIN}"
echo "============================================================"
echo ""
echo "Next steps:"
echo "  1. Copy TURN_SECRET + COTURN_HOST into /opt/kmuhub/deploy/docker/.env.production"
echo "  2. On app server: restart livekit and work services"
echo "     sudo docker compose restart livekit work"
echo "  3. Verify with: turnutils_uclient -T -p 5349 ${DOMAIN}"
echo "============================================================"
