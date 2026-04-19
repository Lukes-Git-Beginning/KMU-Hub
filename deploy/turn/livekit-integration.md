# LiveKit ↔ coturn Integration

> After coturn is deployed, apply the changes below on the **app server** (`178.104.38.195`).
> Do NOT commit secrets. All `CHANGE_ME` / `<REDACTED>` placeholders must be filled in locally.

## 1. `.env.production` — add TURN keys

Add these two lines to `/opt/kmuhub/deploy/docker/.env.production`:

```
# TURN / coturn (Sprint 1 R2.1)
TURN_SECRET=REPLACE_ME            # the secret used when running deploy/turn/deploy.sh
COTURN_HOST=turn.zentria.tech
```

The `TURN_SECRET` value must be **identical** to the `--secret` argument passed to `deploy.sh`.
LiveKit reads this via its config overlay (see step 2), not directly as an env var —
the env file entries are for documentation and future services that may query TURN credentials.

## 2. `docker-compose.prod.yml` — uncomment livekit-turn.yaml volume

Current state (lines 64–82 in `deploy/docker/docker-compose.prod.yml`):

```yaml
  livekit:
    logging: *default-logging
    # volumes:
    #   - ./livekit.yaml:/etc/livekit.yaml
    #   - ./livekit-turn.yaml:/etc/livekit-turn.yaml
    # command: --config /etc/livekit.yaml --config /etc/livekit-turn.yaml
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: "0.5"
```

After change:

```yaml
  livekit:
    logging: *default-logging
    volumes:
      - ./livekit.yaml:/etc/livekit.yaml
      - ./livekit-turn.yaml:/etc/livekit-turn.yaml
    command: --config /etc/livekit.yaml --config /etc/livekit-turn.yaml
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: "0.5"
```

## 3. `livekit-turn.yaml` — verify domain (no edit needed for default setup)

`deploy/docker/livekit-turn.yaml` already contains `domain: turn.zentria.tech`.
If `COTURN_HOST` differs from `turn.zentria.tech`, update the `domain:` field before restarting.

Current content (already committed):

```yaml
use_external_ip: true

turn:
  enabled: true
  domain: turn.zentria.tech
  tls_port: 5349        # for future TLS — coturn will listen here post Let's Encrypt
  udp_port: 3478        # plain UDP — active in MVP
  external_tls: true    # set to false until Let's Encrypt cert is in place
```

> **MVP note:** `external_tls: true` tells LiveKit to use the `turns:` URI scheme (TLS).
> Since coturn MVP has no TLS cert yet, set `external_tls: false` and clients will use
> `turn:` (plain). Update to `true` after Let's Encrypt is configured.

Corrected livekit-turn.yaml for MVP (plain TURN only):

```yaml
use_external_ip: true

turn:
  enabled: true
  domain: turn.zentria.tech
  udp_port: 3478
  external_tls: false
```

## 4. Restart LiveKit

```bash
ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195
cd /opt/kmuhub/deploy/docker
sudo docker compose -f docker-compose.yml -f docker-compose.prod.yml restart livekit
sudo docker compose logs -f livekit --tail=50
```

Confirm in logs: `using TURN server turn:turn.zentria.tech:3478` (or similar LiveKit startup message).

## 5. Smoke-test

**Option A — Trickle ICE web tool (no credentials needed for STUN check):**
1. Open https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/
2. Add ICE server: `turn:turn.zentria.tech:3478`
3. Username / Credential: leave blank for STUN-only test; for TURN test use
   LiveKit-generated credentials from a room join or generate manually:
   ```
   # TURN REST API credential generation (Python one-liner)
   python3 -c "
   import hmac, hashlib, base64, time
   secret = 'YOUR_TURN_SECRET'
   username = str(int(time.time()) + 3600) + ':testuser'
   credential = base64.b64encode(hmac.new(secret.encode(), username.encode(), hashlib.sha1).digest()).decode()
   print('Username:', username)
   print('Credential:', credential)
   "
   ```

**Option B — turnutils_uclient (on the coturn server itself):**
```bash
ssh -i ~/.ssh/hetzner_kmuhub root@<coturn-ip>
turnutils_uclient -T -p 3478 turn.zentria.tech
```

## Key mapping summary

| Where | Key | Value |
|-------|-----|-------|
| `.env.production` (app server) | `TURN_SECRET` | `<hex-secret>` |
| `.env.production` (app server) | `COTURN_HOST` | `turn.zentria.tech` |
| `livekit-turn.yaml` | `turn.domain` | `turn.zentria.tech` |
| `livekit-turn.yaml` | `turn.udp_port` | `3478` |
| `livekit-turn.yaml` | `turn.external_tls` | `false` (MVP), `true` (post-TLS) |
| `coturn /etc/turnserver.conf` | `static-auth-secret` | same `<hex-secret>` |

The shared secret is the single coupling point between LiveKit and coturn.
All three locations (`.env.production`, `livekit-turn.yaml` indirectly via LiveKit runtime,
and `turnserver.conf`) must agree on this value.
