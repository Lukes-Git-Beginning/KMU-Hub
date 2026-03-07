#!/usr/bin/env bash
set -euo pipefail

# KMU Hub Hetzner Cloud Firewall Setup
# Requires: hcloud CLI (https://github.com/hetznercloud/cli)
# Usage: ./firewall.sh YOUR_IP SERVER_NAME

if [[ $# -lt 2 ]]; then
    echo "Usage: $0 <your-ip> <server-name>"
    echo "Example: $0 203.0.113.42 kmuhub-prod"
    exit 1
fi

MY_IP="$1"
SERVER="$2"
FW_NAME="kmuhub-fw"

echo "Creating firewall: $FW_NAME"
hcloud firewall create --name "$FW_NAME" 2>/dev/null || echo "Firewall already exists"

echo "Adding rules..."
hcloud firewall add-rule "$FW_NAME" --direction in --protocol tcp --port 22 --source-ips "$MY_IP/32" --description "SSH" 2>/dev/null || true
hcloud firewall add-rule "$FW_NAME" --direction in --protocol tcp --port 80 --source-ips 0.0.0.0/0 --description "HTTP" 2>/dev/null || true
hcloud firewall add-rule "$FW_NAME" --direction in --protocol tcp --port 443 --source-ips 0.0.0.0/0 --description "HTTPS" 2>/dev/null || true
hcloud firewall add-rule "$FW_NAME" --direction in --protocol tcp --port 7881 --source-ips 0.0.0.0/0 --description "LiveKit WebRTC" 2>/dev/null || true
hcloud firewall add-rule "$FW_NAME" --direction in --protocol udp --port 50000-60000 --source-ips 0.0.0.0/0 --description "LiveKit ICE" 2>/dev/null || true

echo "Applying to server: $SERVER"
hcloud firewall apply-to-resource "$FW_NAME" --type server --server "$SERVER" 2>/dev/null || true

echo "Done. Current rules:"
hcloud firewall describe "$FW_NAME"
