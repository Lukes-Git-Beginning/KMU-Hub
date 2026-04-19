# LiveKit ↔ coturn Integration

> **Status 2026-04-19:** coturn is deployed on Hetzner CAX11 (`turn.zentria.tech`,
> `5.75.246.217`). The external TURN server is **not yet wired into LiveKit** —
> see "Pending work" below. Clients currently cannot use coturn for relay.

## What's done

- coturn 4.6.1 running on `turn.zentria.tech:3478` (plain UDP/TCP, no TLS yet)
- `static-auth-secret` in `/etc/turnserver.conf` matches `TURN_SECRET` in
  `/opt/kmuhub/.env.production` on the app server
- `rtc.use_external_ip: true` is set in `/opt/kmuhub/deploy/docker/livekit.yaml`
  on the production server (improves direct ICE candidates, not a TURN fix)

## What does NOT work (why the first integration attempt failed)

An earlier revision of this doc suggested putting a `turn:` block in
`livekit.yaml` pointing to `turn.zentria.tech`. **That is wrong.** LiveKit's
`turn:` block configures LiveKit's **own embedded TURN server**, not an
external one. Setting it while `turn.zentria.tech` already runs coturn causes
a port conflict and confusion.

## Pending work — wire the external coturn into LiveKit

There are two supported ways to advertise an external TURN server to LiveKit
clients. Pick ONE, **do not combine with a `turn:` block**.

### Option A — static `rtc.turn_servers` (simpler, static credentials)

Requires storing username/credential pairs somewhere (not great for
per-session auth). Skip unless a specific use case demands it.

```yaml
rtc:
  turn_servers:
    - host: turn.zentria.tech
      port: 3478
      protocol: udp
      username: <static-username>
      credential: <static-password>
```

### Option B — ephemeral credentials via `AccessToken` (recommended)

The LiveKit server generates short-lived TURN credentials per room-join using
the `static-auth-secret` shared with coturn. This is the standard pattern
documented by LiveKit.

**Backend change required** (`backend/internal/video/service.go`, when building
the `AccessToken`):

```go
// Generate REST-API-style TURN credential:
// username = "<unix-expiry-seconds>:<livekit-identity>"
// credential = base64( HMAC-SHA1( TURN_SECRET, username ) )

expiry := time.Now().Add(4 * time.Hour).Unix()
username := fmt.Sprintf("%d:%s", expiry, identity)
h := hmac.New(sha1.New, []byte(turnSecret))
h.Write([]byte(username))
credential := base64.StdEncoding.EncodeToString(h.Sum(nil))

at := auth.NewAccessToken(apiKey, apiSecret).
    AddGrant(&auth.VideoGrant{Room: roomName, RoomJoin: true}).
    SetIdentity(identity)

// Attach TURN server to the token (LiveKit ≥ 1.5)
at.SetTurnServer(&livekit.TURNServer{
    Host:       "turn.zentria.tech",
    Port:       3478,
    Protocol:   livekit.TURNServer_UDP,
    Username:   username,
    Credential: credential,
})
```

The app server reads `TURN_SECRET` from `.env.production` — already set during
the 2026-04-19 deploy.

## Key mapping summary

| Where                                    | Key                    | Value                                   |
|------------------------------------------|------------------------|-----------------------------------------|
| `coturn /etc/turnserver.conf` (TURN box) | `static-auth-secret`   | `<hex-secret>`                          |
| `/opt/kmuhub/.env.production` (app box)  | `TURN_SECRET`          | same `<hex-secret>`                     |
| `/opt/kmuhub/.env.production` (app box)  | `COTURN_HOST`          | `turn.zentria.tech`                     |
| `video` service (Go code, pending)       | read `TURN_SECRET` env | used to HMAC-sign per-token credentials |
| LiveKit client (browser)                 | receives TURN URI      | via access token (Option B)             |

## Obsolete file

`deploy/docker/livekit-turn.yaml` — **do not mount it**. It was created
alongside this doc under the wrong assumption about LiveKit's `turn:` block.
The file remains in the repo as a historical artifact for reference and will
be removed (or rewritten for Option A) once the backend-side TURN wiring lands.

## Smoke test (after Option B is implemented)

1. Deploy backend with TURN-credential generation in access tokens
2. Start a call — inspect the client-side `RTCPeerConnection.getStats()`;
   for clients behind symmetric NAT you should see `candidateType: relay`
   with `turnServer: turn.zentria.tech:3478`
3. Server-side coturn log should show `session ... allocated` entries
   (`sudo journalctl -u coturn -f` on `turn.zentria.tech`)

## Standalone test (no LiveKit required)

Verifies coturn alone is healthy:

```bash
# On any machine with the TURN_SECRET:
TURN_SECRET=<hex-secret>
EXPIRY=$(($(date +%s) + 3600))
USERNAME="$EXPIRY:test"
CREDENTIAL=$(echo -n "$USERNAME" | openssl dgst -sha1 -hmac "$TURN_SECRET" -binary | base64)

echo "Username:   $USERNAME"
echo "Credential: $CREDENTIAL"
```

Plug these into https://webrtc.github.io/samples/src/content/peerconnection/trickle-ice/
with `turn:turn.zentria.tech:3478`. A successful `relay` candidate confirms
coturn works end-to-end.
