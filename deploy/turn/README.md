# coturn Deploy — Sprint 1 R2.1

Self-hosted TURN server on a dedicated Hetzner CAX11 (ARM, 4 GB RAM, 20 TB traffic, ~€3.80/M).
Decouples TURN from the main CPX42 app server to protect relay bandwidth and keep the
LiveKit port range off the production application host.

---

## Step 1 — Hetzner Cloud Console: create CAX11

1. Log in to [console.hetzner.cloud](https://console.hetzner.cloud)
2. Select your project → **Servers** → **Add Server**
3. Configure:
   | Setting | Value |
   |---------|-------|
   | Location | Nuremberg (nbg1) |
   | Image | Ubuntu 24.04 |
   | Type | **CAX11** (Ampere, 2 vCPU, 4 GB RAM) |
   | SSH keys | `hetzner_kmuhub` (already added to your project) |
   | Name | `zentria-turn` |
4. Click **Create & Buy**
5. Note the **public IPv4** — you need it for every step below.

**Hetzner Cloud Firewall** (create or edit `kmuhub-fw` / create a separate `turn-fw`):

| Direction | Protocol | Port(s) | Source |
|-----------|----------|---------|--------|
| Inbound | TCP | 22 | `0.0.0.0/0` (restrict to your IP for hardening) |
| Inbound | UDP | 3478 | `0.0.0.0/0` |
| Inbound | TCP | 3478 | `0.0.0.0/0` |
| Inbound | UDP | 49152-65535 | `0.0.0.0/0` |

Attach the firewall to `zentria-turn`.

> TLS (port 5349) is not in scope for this MVP wave. Add it after Let's Encrypt is configured.

---

## Step 2 — DNS

In your DNS provider (Hetzner DNS or wherever `zentria.tech` is managed):

1. Add an **A record**: `turn.zentria.tech` → `<CAX11 public IPv4>`
2. In **Hetzner Cloud Console → Server → zentria-turn → Networking**:
   set **Reverse DNS (PTR)** to `turn.zentria.tech`

DNS propagation is not required for the deploy script, but LiveKit needs it resolved
before TURN will work for clients.

---

## Step 3 — Generate the shared secret

Run on your local machine:

```bash
openssl rand -hex 32
```

**Save this value** in your password manager (Bitwarden / 1Password / etc.) before proceeding.
It must be entered identically in two places: `deploy.sh --secret` and `.env.production` on the app server.

---

## Step 4 — Run deploy.sh

From the repo root on your local machine (Git Bash):

```bash
bash deploy/turn/deploy.sh \
  --host <CAX11-public-ip> \
  --secret <your-hex-secret>
```

This will:
- Upload `setup.sh` and `turnserver.conf.template` to `/tmp/` on the server
- Install coturn via apt
- Configure UFW (SSH, UDP/TCP 3478, UDP 49152-65535)
- Write `/etc/turnserver.conf` from the template
- Enable + start the coturn systemd service
- Print a health-check summary

If the script fails, check coturn logs on the server:
```bash
ssh -i ~/.ssh/hetzner_kmuhub root@<ip>
journalctl -u coturn -n 50
```

---

## Step 5 — LiveKit integration on app server

> **The earlier revision of this section (uncomment `livekit-turn.yaml`, add a
> `turn:` block to `livekit.yaml`) was wrong.** LiveKit's `turn:` block
> configures LiveKit's *own* embedded TURN server, not an external one — putting
> `turn.zentria.tech` there causes a port conflict and a half-broken setup.
>
> The correct integration is the **Metadata-JSON** strategy: per-session TURN
> credentials are embedded in every LiveKit AccessToken by the Go backend (HMAC-SHA1
> over `<expiry>:<identity>`). LiveKit itself stays unaware of coturn — it just
> hands the metadata to the client, which uses it for ICE.
>
> See `deploy/turn/livekit-integration.md` for the implementation details and
> Before/After diff.

What you actually need to do on `178.104.38.195`:

1. Add to `/opt/kmuhub/.env.production`:
   ```
   TURN_SECRET=REPLACE_ME      # must match static-auth-secret in /etc/turnserver.conf on the TURN host
   COTURN_HOST=turn.zentria.tech
   ```

2. Restart the affected app services so they pick up the new env vars:
   ```bash
   ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195
   cd /opt/kmuhub
   sudo docker compose --env-file .env.production \
     -f deploy/docker/docker-compose.yml -f deploy/docker/docker-compose.prod.yml \
     restart work gateway
   ```

3. Verify a freshly issued AccessToken contains TURN ICE servers in the Metadata
   field — see Step 6 below for the smoke-test.

**Do not** edit `livekit.yaml` to add a `turn:` block, and **do not** mount
`livekit-turn.yaml`. Those files exist only as historical reference and are
superseded by the AccessToken-Metadata approach.

---

## Step 6 — Smoke-test

**Trickle ICE web tool** (quickest, no install needed):
1. Open https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/
2. Remove default STUN server entries
3. Add: `turn:turn.zentria.tech:3478`
4. For credentials, generate a time-limited pair from the shared secret:
   ```bash
   python3 -c "
   import hmac, hashlib, base64, time
   secret = 'YOUR_TURN_SECRET'
   user = str(int(time.time()) + 3600) + ':test'
   cred = base64.b64encode(hmac.new(secret.encode(), user.encode(), hashlib.sha1).digest()).decode()
   print('Username:', user); print('Credential:', cred)
   "
   ```
5. Click **Gather candidates** — you should see `relay` candidates appear.

**turnutils_uclient** (on the coturn server):
```bash
ssh -i ~/.ssh/hetzner_kmuhub root@<coturn-ip>
turnutils_uclient -T -p 3478 127.0.0.1
```

---

## File overview

| File | Purpose |
|------|---------|
| `deploy.sh` | Local wrapper — scp + ssh orchestration |
| `setup.sh` | Remote setup script — runs on the CAX11 |
| `turnserver.conf.template` | coturn config with `{{PLACEHOLDER}}` variables |
| `livekit-integration.md` | Before/After diff for LiveKit config on app server |
| `README.md` | This file |

---

## Future: TLS with Let's Encrypt

When adding TLS (post-MVP):

1. Open port 80 (HTTP-01 challenge) and 5349 (TURNS) in Hetzner firewall + UFW
2. Install certbot: `apt install certbot`
3. Obtain cert: `certbot certonly --standalone -d turn.zentria.tech`
4. Add to `/etc/turnserver.conf`:
   ```
   tls-listening-port=5349
   cert=/etc/letsencrypt/live/turn.zentria.tech/fullchain.pem
   pkey=/etc/letsencrypt/live/turn.zentria.tech/privkey.pem
   ```
5. Set up auto-renewal hook (see `deploy/hetzner/coturn-setup.sh` for reference)
6. Update `livekit-turn.yaml`: set `tls_port: 5349`, `external_tls: true`
7. Restart coturn + LiveKit

The `deploy/hetzner/coturn-setup.sh` file contains a full TLS-enabled version for reference.
