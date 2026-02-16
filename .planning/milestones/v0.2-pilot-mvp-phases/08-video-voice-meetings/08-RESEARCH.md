# Phase 8: Video, Voice & Meetings - Research

**Researched:** 2026-02-11
**Domain:** LiveKit video/voice, meeting lifecycle management, presence system, emoji reactions
**Confidence:** HIGH (LiveKit SDK well-documented, patterns match existing codebase)

## Summary

Phase 8 covers five major domains: (1) LiveKit-powered video/voice calls with room management, (2) meeting lifecycle with notes and action items, (3) DSGVO-compliant recording via LiveKit Egress, (4) presence/online-status via Redis + WebSocket, and (5) emoji reactions on chat messages. The existing codebase already has LiveKit token generation (`internal/work/livekit/service.go`), WebSocket hub (`internal/server/websocket.go`), and the `livekit/protocol` v1.44.0 dependency.

The standard approach uses `livekit/server-sdk-go/v2` for server-side room management and egress control, `@livekit/components-react` with `livekit-client` for the frontend, a custom egress template for DSGVO selective recording, Redis sorted sets for presence heartbeats, and a new `message_reactions` table for emoji reactions.

**Primary recommendation:** Extend the existing Work service (`:50055`) with a new `video.proto` for call/meeting RPCs, add `@livekit/components-react` to the desktop app, deploy LiveKit server + egress as Docker services, and use Redis for presence state with WebSocket broadcast.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Call Experience & Controls
- **Video layout:** Gallery + Speaker view, switchable. Default Gallery-Grid (all equal size), click on participant or active speaker switches to Speaker-View (1 large + rest as thumbnails)
- **Floating call bar:** Compact floating mini-bar (top or bottom) with Mute/Hangup/Camera-Toggle + duration timer when user navigates away. Call continues in background
- **Incoming call:** Fullscreen overlay over the entire screen (like phone app) with avatar, name, accept/decline buttons
- **Screen sharing:** Shared screen replaces the speaker area (main area), participant video thumbnails move to a sidebar

#### Meeting Lifecycle
- **Pre-meeting lobby:** Camera/microphone preview + check, shared meeting documents and agenda for reading before start
- **Meeting notes:** During the meeting there is a notes panel. After end, an automatic summary draft is created, organizer reviews and finalizes
- **Action items -> Tasks:** Batch conversion after the meeting: "Create all action items as tasks" button, creates tasks in one batch in the selected project
- **Recurring meetings:** When opening a recurring meeting, shows a "Last Notes" panel with the summary from the previous meeting

#### DSGVO Recording Consent
- **Consent flow:** All participants are asked (Accept/Decline popup). Participants who accept are fully recorded (video + audio). Participants who decline are blurred (video) and muted (audio) in the recording -- selective consent
- **Storage:** Recording appears in Meeting-Detail-Panel AND in central file manager (Phase 11) under a meeting folder. Only meeting participants have access
- **Retention:** 30-day retention, then automatic deletion. DSGVO-compliant as priority

#### Presence & Online-Status
- **Detection:** Automatic based on activity + manually overridable. Away-timeout is admin-configurable (not hardcoded) -- maximally flexible for the customer
- **Status levels:** 5 levels: Online (green), Away (yellow), Do Not Disturb (red), In Call (purple/blue, automatically set), Offline (gray)
- **"In Call" status:** Automatically set when user is in an active call
- **Visibility:** Presence dots only in chat participant lists/DMs and team overview -- not in CRM or calendar

### Claude's Discretion
- LiveKit SDK integration patterns (room management, token generation, track handling)
- Exact floating bar positioning and animation
- Summary auto-draft algorithm from meeting notes
- Presence heartbeat interval and Redis data structure
- Recording blur/mute technical approach (LiveKit Egress capabilities vs post-processing)
- Emoji reaction picker component choice and animation style

### Deferred Ideas (OUT OF SCOPE)
- Whiteboard during calls (D8 in Darien's design roadmap, large scope) -- future phase
- Meeting transcription (AI-powered) -- future phase or v2
- Virtual background / background blur for user's own camera -- future enhancement
- Meeting room booking from meeting form (already exists in Calendar Phase 7, just needs linking)
</user_constraints>

## Standard Stack

### Core - Backend (Go)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `livekit/server-sdk-go/v2` | latest | Room management, egress control, webhook validation | Official LiveKit Go server SDK, required for RoomServiceClient and EgressClient |
| `livekit/protocol` | v1.44.0 (already installed) | Protocol types, auth token generation | Already in go.mod, provides `livekit.CreateRoomRequest`, `auth.VideoGrant`, webhook types |
| `livekit/protocol/webhook` | (part of protocol) | Webhook event receiver and validation | Built-in webhook handler with JWT signature verification |
| `jackc/pgx/v5` | v5.8.0 (already installed) | Database access for meetings, reactions, recordings | Standard PostgreSQL driver, already in use |
| `redis/go-redis/v9` | v9.17.3 (already installed) | Presence heartbeat storage and pub/sub | Already in use for rate limiting, extend to presence |

### Core - Frontend (Electron/React)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `livekit-client` | latest | Core LiveKit client SDK | Required peer dependency, handles WebRTC connections |
| `@livekit/components-react` | ~2.9.x | Pre-built React components for video conferencing | Official React SDK: VideoConference, GridLayout, FocusLayout, ControlBar, PreJoin |
| `@livekit/components-styles` | latest | Default styles for LiveKit components | Peer dependency of components-react |
| `frimousse` | latest | Emoji picker | Lightweight (~0 deps), unstyled, composable, works with Radix UI + Tailwind (already in stack) |

### Core - Infrastructure

| Service | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `livekit/livekit-server` | latest | WebRTC SFU server | Self-hostable, EU-deployable, the core of the video platform |
| `livekit/egress` | latest | Recording service | Handles room composite recording with custom templates, outputs to S3/MinIO |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `coder/websocket` | v1.8.14 (already installed) | WebSocket for presence updates | Already used by chat WebSocket hub, extend for presence |
| `minio/minio-go/v7` | v7.0.98 (already installed) | S3-compatible file storage | Already used for chat files, reuse for recordings |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `frimousse` | `emoji-picker-react` (4.18.x) | emoji-picker-react is 2.59MB vs frimousse ~0 deps; frimousse is unstyled and matches Radix UI + Tailwind stack better |
| `@livekit/components-react` prefabs | Custom components with `livekit-client` hooks | Prefabs (VideoConference, PreJoin) save massive time; customize via composition not from scratch |
| Redis for presence | PostgreSQL for presence | Redis is orders of magnitude faster for heartbeat writes (SET with TTL), presence is ephemeral data not suited for PostgreSQL |

**Installation (Backend):**
```bash
cd backend
go get github.com/livekit/server-sdk-go/v2
```

**Installation (Frontend):**
```bash
cd desktop
npm install livekit-client @livekit/components-react @livekit/components-styles frimousse
```

## Architecture Patterns

### Recommended Project Structure

```
backend/
  proto/video/v1/video.proto              # New proto for calls, meetings, recordings
  internal/work/video/
    service.go                            # Call session management
    repository.go                         # Repository interface
    postgres_repository.go                # PostgreSQL implementation
  internal/work/meeting/
    service.go                            # Meeting lifecycle (notes, action items, summaries)
    repository.go
    postgres_repository.go
  internal/work/recording/
    service.go                            # Egress management, consent tracking, retention
    repository.go
    postgres_repository.go
  internal/work/presence/
    service.go                            # Presence heartbeat, status management
    redis_store.go                        # Redis-backed presence storage
  internal/work/reaction/
    service.go                            # Emoji reactions on messages
    repository.go
    postgres_repository.go
  internal/gateway/
    route_video.go                        # HTTP routes for video/meeting/recording
  internal/server/
    video_grpc.go                         # gRPC handler for VideoService
  cmd/work/main.go                        # Extended to register VideoService

desktop/
  src/renderer/src/features/meetings/     # Meeting pages (Darien's components)
  src/renderer/src/features/video/        # Video call components (LiveKit integration)
  src/renderer/src/features/presence/     # Presence indicators
  src/renderer/src/hooks/useLiveKit.ts    # LiveKit connection hooks
  src/renderer/src/hooks/usePresence.ts   # Presence WebSocket hooks
```

### Pattern 1: VideoService in Work Binary (Service Consolidation)

**What:** Register a new `VideoService` gRPC server alongside `WorkService` and `CalendarService` in the same `cmd/work/main.go` binary (port `:50055`).
**When to use:** This follows the existing consolidation decision -- Work service already hosts Calendar, adding Video/Meeting maintains the pattern.
**Example:**
```go
// Source: Existing pattern from cmd/work/main.go
videoGRPC := server.NewVideoGRPCServer(videoService, meetingService, recordingService, presenceService, reactionService)
videov1.RegisterVideoServiceServer(grpcServer, videoGRPC)
```

### Pattern 2: LiveKit Room Management via Server SDK

**What:** Use `livekit/server-sdk-go/v2` RoomServiceClient to create/manage rooms server-side, while clients connect via tokens.
**When to use:** For all call initiation, room creation, and participant management.
**Example:**
```go
// Source: https://docs.livekit.io/home/server/managing-rooms/
import lksdk "github.com/livekit/server-sdk-go/v2"
import "github.com/livekit/protocol/livekit"

roomClient := lksdk.NewRoomServiceClient(s.wsURL, s.apiKey, s.apiSecret)

room, err := roomClient.CreateRoom(ctx, &livekit.CreateRoomRequest{
    Name:            roomName,
    EmptyTimeout:    10 * 60, // 10 minutes
    MaxParticipants: 25,      // matches VID-02 requirement
})
```

### Pattern 3: LiveKit Webhook Handler for Room Events

**What:** Receive LiveKit server webhooks for room lifecycle events (room_started, room_finished, participant_joined, participant_left) to update presence status and meeting records.
**When to use:** For automatic "In Call" presence detection and meeting duration tracking.
**Example:**
```go
// Source: https://docs.livekit.io/home/server/webhooks/
import (
    "github.com/livekit/protocol/auth"
    "github.com/livekit/protocol/webhook"
)

func (h *WebhookHandler) HandleLiveKitWebhook(w http.ResponseWriter, r *http.Request) {
    authProvider := auth.NewSimpleKeyProvider(h.apiKey, h.apiSecret)
    event, err := webhook.ReceiveWebhookEvent(r, authProvider)
    if err != nil {
        respondError(w, http.StatusUnauthorized, err)
        return
    }
    switch event.GetEvent() {
    case "participant_joined":
        h.presenceService.SetInCall(ctx, event.GetParticipant().GetIdentity())
    case "participant_left":
        h.presenceService.ClearInCall(ctx, event.GetParticipant().GetIdentity())
    case "room_finished":
        h.meetingService.EndMeeting(ctx, event.GetRoom().GetName())
    }
}
```

### Pattern 4: Redis Presence with Heartbeat TTL

**What:** Store presence as Redis keys with TTL. Client sends heartbeat via WebSocket every 30 seconds. Server sets `presence:{userId}` with 90-second TTL. If heartbeats stop, key expires and user is offline.
**When to use:** For all presence tracking (online/away/offline/DND/in-call).
**Recommended data structure:**
```
# Redis keys
presence:{userId}          -> JSON: {"status":"online","manual":false,"updated_at":"..."}
presence:away_timeout      -> int (seconds), admin-configurable, default 300
presence:subscribers:{userId} -> SET of userIDs subscribed to this user's presence changes
```
**Heartbeat interval:** 30 seconds (client-side), 90-second TTL (server-side, 3x heartbeat).
**Away detection:** Server tracks last activity timestamp. If `now - last_activity > away_timeout` and no manual override, status becomes "away".

### Pattern 5: Egress with Custom Template for DSGVO Selective Recording

**What:** Use LiveKit RoomCompositeEgress with a custom recording template that reads participant consent metadata and applies CSS blur filter + audio muting for non-consenting participants.
**When to use:** For all meeting recordings (VID-07).
**How it works:**
1. Before recording starts, each participant sets metadata on their LiveKit participant object: `{"recording_consent": true|false}`
2. Custom egress template (a React app served by the egress service) reads participant metadata
3. For `recording_consent: false`, the template applies `filter: blur(20px)` on the video tile and mutes the audio element
4. Egress renders the composite with consenting participants clear and non-consenting participants blurred/muted
5. Recording is saved to MinIO via S3-compatible upload

### Pattern 6: Emoji Reactions as Database + WebSocket Broadcast

**What:** Store reactions in `message_reactions` table, broadcast via WebSocket for real-time updates.
**When to use:** For CHAT-01 emoji reactions.
**Schema pattern:**
```sql
CREATE TABLE message_reactions (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    emoji TEXT NOT NULL,          -- Unicode emoji character
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);
CREATE INDEX idx_message_reactions_message ON message_reactions(message_id);
```

### Anti-Patterns to Avoid

- **Polling for presence:** Never poll the server for presence status. Use WebSocket push with Redis pub/sub for real-time updates.
- **Storing recordings in PostgreSQL:** Recording files go to MinIO. Only metadata (URL, duration, consent records, retention date) goes to PostgreSQL.
- **Running LiveKit in the same process:** LiveKit server is a separate Docker container. Do NOT embed WebRTC in the Go backend.
- **Global emoji picker state:** Emoji picker should be per-message, not a global singleton. Use Radix Popover per message.
- **Hardcoded away timeout:** Away timeout MUST be admin-configurable via system settings table, not a constant.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Video conferencing UI | Custom WebRTC components | `@livekit/components-react` VideoConference + GridLayout + FocusLayout | WebRTC layout, track management, reconnection are extremely complex |
| WebRTC SFU server | Custom Go WebRTC server | `livekit/livekit-server` Docker image | SFU topology, TURN/STUN, bandwidth estimation = years of work |
| Recording compositor | Custom ffmpeg pipeline | LiveKit Egress service with custom template | Chrome-based compositing handles layout, encoding, S3 upload automatically |
| Emoji data/rendering | Custom emoji database | `frimousse` (picker) + native Unicode emoji rendering | Emoji data changes per Unicode version, cross-platform rendering is a minefield |
| Screen sharing capture | Custom Electron screen capture | `livekit-client` built-in screen share via `localParticipant.setScreenShareEnabled(true)` | Electron's desktopCapturer integration is handled by livekit-client |
| Token generation/validation | Custom JWT for LiveKit | `livekit/protocol/auth.NewAccessToken()` | Already using this in `internal/work/livekit/service.go` |

**Key insight:** LiveKit provides the entire video infrastructure stack (SFU, client SDK, recording, webhooks). The backend's job is orchestration (room lifecycle, consent, meeting records), not media handling.

## Common Pitfalls

### Pitfall 1: LiveKit Egress Resource Requirements
**What goes wrong:** Egress service crashes or produces low-quality recordings.
**Why it happens:** RoomCompositeEgress runs headless Chrome and needs 4 CPUs + 4GB RAM minimum. Under-provisioning causes Chrome to struggle.
**How to avoid:** Allocate dedicated CPU/memory in docker-compose. In development, limit to 1 concurrent egress. In production, use Kubernetes with egress autoscaling.
**Warning signs:** Egress jobs timing out, choppy recordings, high CPU on egress container.

### Pitfall 2: WebSocket Presence Thundering Herd
**What goes wrong:** When many users come online simultaneously (e.g., morning login), a flood of presence updates overwhelms the WebSocket hub.
**Why it happens:** Each user's status change broadcasts to all subscribers.
**How to avoid:** Batch presence updates (aggregate changes over 2-3 second windows before broadcasting). Use Redis pub/sub for cross-instance propagation.
**Warning signs:** WebSocket message queue backing up, gateway memory spike during peak hours.

### Pitfall 3: DSGVO Recording Consent Race Condition
**What goes wrong:** Recording starts before all participants have responded to consent popup.
**Why it happens:** Egress is started immediately when organizer clicks "Record" without waiting for all consent responses.
**How to avoid:** Recording can ONLY start after all current participants have responded. New joiners get the consent popup; recording of their tracks starts only after they respond.
**Warning signs:** Participants recorded without consent (legal liability).

### Pitfall 4: Stale Presence After Crash
**What goes wrong:** User shows as "Online" even though their app crashed.
**Why it happens:** No heartbeat = no TTL renewal, but TTL may be long.
**How to avoid:** 30-second heartbeat with 90-second TTL means maximum 90 seconds of stale presence. WebSocket disconnect handler immediately removes presence. LiveKit webhook `participant_left` clears "In Call" status.
**Warning signs:** Ghost users showing as online in chat member lists.

### Pitfall 5: Meeting Notes Lost on Disconnect
**What goes wrong:** Meeting notes typed during a call are lost when the WebSocket disconnects.
**Why it happens:** Notes stored only in client state, not persisted until "save".
**How to avoid:** Auto-save meeting notes every 30 seconds via API call. Store as draft in meeting_notes table. Optimistic UI with conflict resolution on reconnect.
**Warning signs:** Users reporting lost notes after network issues.

### Pitfall 6: LiveKit Server Configuration for Self-Hosting
**What goes wrong:** Video calls fail to connect or have poor quality.
**Why it happens:** LiveKit needs proper TURN/STUN configuration for NAT traversal, especially in corporate networks.
**How to avoid:** Configure LiveKit with embedded TURN server for development. For production, use LiveKit's built-in TURN with proper TLS certificates. Test behind NAT/firewall.
**Warning signs:** Calls failing to connect, one-way audio, high packet loss.

### Pitfall 7: Emoji Reaction Count Consistency
**What goes wrong:** Reaction counts become inconsistent between database and UI.
**Why it happens:** Optimistic UI updates without proper conflict resolution.
**How to avoid:** Use the composite primary key `(message_id, user_id, emoji)` to prevent duplicates. Toggle semantics: if reaction exists, remove it; if not, add it. Return the full reaction list after each toggle for consistency.
**Warning signs:** Duplicate reactions from same user, counts not matching.

## Code Examples

### LiveKit Room Creation (Go Backend)
```go
// Source: https://docs.livekit.io/home/server/managing-rooms/
import lksdk "github.com/livekit/server-sdk-go/v2"
import "github.com/livekit/protocol/livekit"

type VideoService struct {
    roomClient  *lksdk.RoomServiceClient
    egressClient *lksdk.EgressClient
    // ...
}

func NewVideoService(apiKey, apiSecret, wsURL string) *VideoService {
    return &VideoService{
        roomClient:   lksdk.NewRoomServiceClient(wsURL, apiKey, apiSecret),
        egressClient: lksdk.NewEgressClient(wsURL, apiKey, apiSecret),
    }
}

func (s *VideoService) CreateCallRoom(ctx context.Context, callID string, maxParticipants int) (*livekit.Room, error) {
    return s.roomClient.CreateRoom(ctx, &livekit.CreateRoomRequest{
        Name:            "call-" + callID[:8],
        EmptyTimeout:    10 * 60,
        MaxParticipants: uint32(maxParticipants),
    })
}
```

### LiveKit Egress to MinIO (Go Backend)
```go
// Source: https://docs.livekit.io/home/egress/examples/
func (s *RecordingService) StartRecording(ctx context.Context, roomName string) (*livekit.EgressInfo, error) {
    req := &livekit.RoomCompositeEgressRequest{
        RoomName:      roomName,
        Layout:        "speaker",
        CustomBaseUrl: s.customTemplateURL, // Custom DSGVO template
        FileOutputs: []*livekit.EncodedFileOutput{
            {
                FileType: livekit.EncodedFileType_MP4,
                Filepath: fmt.Sprintf("recordings/%s/{room_name}-{time}.mp4", roomName),
                Output: &livekit.EncodedFileOutput_S3{
                    S3: &livekit.S3Upload{
                        AccessKey:      s.minioAccessKey,
                        Secret:         s.minioSecret,
                        Endpoint:       s.minioEndpoint,
                        Bucket:         s.minioBucket,
                        ForcePathStyle: true, // Required for MinIO
                    },
                },
            },
        },
    }
    return s.egressClient.StartRoomCompositeEgress(ctx, req)
}
```

### Presence Redis Store (Go Backend)
```go
// Recommended Redis data structure for presence
func (s *PresenceStore) SetPresence(ctx context.Context, userID, status string, manual bool) error {
    data, _ := json.Marshal(map[string]interface{}{
        "status":  status,
        "manual":  manual,
        "updated": time.Now().Unix(),
    })
    // 90-second TTL, renewed every 30 seconds by heartbeat
    return s.redis.Set(ctx, "presence:"+userID, data, 90*time.Second).Err()
}

func (s *PresenceStore) GetPresence(ctx context.Context, userIDs []string) (map[string]string, error) {
    pipe := s.redis.Pipeline()
    cmds := make(map[string]*redis.StringCmd, len(userIDs))
    for _, id := range userIDs {
        cmds[id] = pipe.Get(ctx, "presence:"+id)
    }
    _, _ = pipe.Exec(ctx)

    result := make(map[string]string, len(userIDs))
    for id, cmd := range cmds {
        val, err := cmd.Result()
        if err == redis.Nil {
            result[id] = "offline"
        } else {
            var p struct{ Status string `json:"status"` }
            json.Unmarshal([]byte(val), &p)
            result[id] = p.Status
        }
    }
    return result, nil
}
```

### LiveKit React Frontend Connection
```tsx
// Source: https://docs.livekit.io/reference/components/react/
import { LiveKitRoom, VideoConference, GridLayout, FocusLayout, ControlBar, PreJoin } from '@livekit/components-react';
import '@livekit/components-styles';

function CallRoom({ token, wsUrl }: { token: string; wsUrl: string }) {
  return (
    <LiveKitRoom
      token={token}
      serverUrl={wsUrl}
      connect={true}
      audio={true}
      video={true}
    >
      <VideoConference />
    </LiveKitRoom>
  );
}

// Pre-join lobby with camera/mic preview
function MeetingLobby({ onJoin }: { onJoin: (audio: boolean, video: boolean) => void }) {
  return (
    <PreJoin
      onSubmit={(values) => onJoin(values.audioEnabled, values.videoEnabled)}
    />
  );
}
```

### Emoji Reaction Toggle (Go Backend)
```go
func (s *ReactionService) ToggleReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*ReactionResult, error) {
    exists, err := s.repo.ReactionExists(ctx, messageID, userID, emoji)
    if err != nil {
        return nil, err
    }
    if exists {
        if err := s.repo.RemoveReaction(ctx, messageID, userID, emoji); err != nil {
            return nil, err
        }
    } else {
        if err := s.repo.AddReaction(ctx, messageID, userID, emoji); err != nil {
            return nil, err
        }
    }
    // Return full reaction list for this message for consistency
    reactions, err := s.repo.ListReactions(ctx, messageID)
    return &ReactionResult{Reactions: reactions, Added: !exists}, err
}
```

### WebSocket Presence Events
```go
// New WebSocket message types for presence
const (
    WSPresenceUpdate     = "presence.update"      // Server -> Client
    WSPresenceHeartbeat  = "presence.heartbeat"    // Client -> Server
    WSPresenceSubscribe  = "presence.subscribe"    // Client -> Server
    WSPresenceStatus     = "presence.set_status"   // Client -> Server (manual override)

    // Call events
    WSCallIncoming       = "call.incoming"         // Server -> Client
    WSCallAccepted       = "call.accepted"         // Client -> Server
    WSCallDeclined       = "call.declined"         // Client -> Server
    WSCallEnded          = "call.ended"            // Server -> Client

    // Reaction events
    WSReactionToggled    = "reaction.toggled"      // Server -> Client
)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `livekit/server-sdk-go` v1 | `livekit/server-sdk-go/v2` | 2024 | New import path: `github.com/livekit/server-sdk-go/v2`, uses Twirp HTTP instead of direct gRPC for server APIs |
| `@livekit/react-components` (deprecated) | `@livekit/components-react` | 2023 | Complete rewrite with composable architecture, prefab components |
| Manual egress configuration | Auto Egress (in CreateRoom) | 2024 | Can automatically start recording when room is created, but we need manual control for consent flow |
| Track-level egress for selective recording | Room composite with custom template | Current | Custom templates give full control over which participants appear in recording |

**Deprecated/outdated:**
- `@livekit/react-components`: Replaced by `@livekit/components-react` (completely different package)
- `livekit/server-sdk-go` v1 (non-v2 import): Still works but v2 is recommended
- Polling-based presence: WebSocket push is the standard approach

## Open Questions

1. **LiveKit server version for docker-compose**
   - What we know: Latest image is `livekit/livekit-server:latest`, needs Redis for multi-instance
   - What's unclear: Exact TURN configuration needed for Electron app behind corporate NATs
   - Recommendation: Start with embedded TURN in development, document production TURN setup separately

2. **Custom Egress Template Hosting**
   - What we know: Egress loads a web app via URL for composite recording; the template can read participant metadata
   - What's unclear: Whether the custom DSGVO template should be served from the gateway or a separate static file server
   - Recommendation: Serve from gateway as a static route (`/recording-template/`) for simplicity; the egress service fetches it via HTTP

3. **Meeting Summary Auto-Draft Algorithm**
   - What we know: User wants auto-summary from meeting notes after meeting ends
   - What's unclear: Whether this should be a simple structural extraction (headings -> sections, bullet points -> action items) or something more sophisticated
   - Recommendation: V1 = structural: extract lines starting with `- [ ]` or `TODO:` as action items, everything else as notes. V2 could add AI summarization.

4. **livekit/server-sdk-go/v2 vs existing livekit/protocol**
   - What we know: go.mod has `livekit/protocol v1.44.0`; server-sdk-go/v2 also depends on protocol
   - What's unclear: Whether adding server-sdk-go/v2 will cause version conflicts with existing protocol dependency
   - Recommendation: Run `go get` and check for conflicts. The protocol package should be compatible since server-sdk-go/v2 depends on it.

5. **Electron desktopCapturer for Screen Sharing**
   - What we know: livekit-client handles screen sharing, but Electron needs `desktopCapturer` for screen/window selection
   - What's unclear: Whether `@livekit/components-react` automatically uses Electron's desktopCapturer or needs manual integration
   - Recommendation: Research during implementation. May need to pass `getDisplayMedia` override to LiveKitRoom or use Electron's IPC to provide screen sources.

## Sources

### Primary (HIGH confidence)
- [LiveKit Room Management Docs](https://docs.livekit.io/home/server/managing-rooms/) - Room creation, listing, deletion Go examples
- [LiveKit Egress Docs](https://docs.livekit.io/home/egress/overview/) - Egress types, architecture, self-hosting
- [LiveKit Egress Examples](https://docs.livekit.io/home/egress/examples/) - Go SDK code for StartRoomCompositeEgress with S3
- [LiveKit Custom Templates](https://docs.livekit.io/home/egress/custom-template/) - Custom recording layout architecture
- [LiveKit Recording Consent Recipe](https://docs.livekit.io/recipes/recording-consent/) - Consent collection patterns
- [LiveKit Webhooks](https://docs.livekit.io/home/server/webhooks/) - Event types, Go handler, configuration
- [LiveKit React Components](https://docs.livekit.io/reference/components/react/) - VideoConference, GridLayout, FocusLayout, hooks
- [LiveKit Self-Hosting Egress](https://docs.livekit.io/home/self-hosting/egress/) - Docker deployment, config YAML, resource requirements
- [LiveKit Go Server SDK](https://github.com/livekit/server-sdk-go) - v2 import path, RoomServiceClient, EgressClient
- Existing codebase: `backend/internal/work/livekit/service.go` - Token generation pattern already established
- Existing codebase: `backend/internal/server/websocket.go` - WebSocket hub pattern for extending presence

### Secondary (MEDIUM confidence)
- [Frimousse Emoji Picker](https://frimousse.liveblocks.io) - Lightweight, unstyled, composable React emoji picker
- [Redis Presence Pattern](https://medium.com/tilt-engineering/redis-powered-user-session-tracking-with-heartbeat-based-expiration-c7308420489f) - Heartbeat TTL approach
- [System Design Presence](https://systemdesign.one/real-time-presence-platform-system-design/) - Architecture patterns for presence systems

### Tertiary (LOW confidence)
- npm version info for `@livekit/components-react` v2.9.19 (from search results, not verified via npm directly)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - LiveKit SDKs are well-documented, existing codebase already uses livekit/protocol
- Architecture: HIGH - Follows existing patterns (Work service consolidation, WebSocket hub, gateway routes)
- Pitfalls: MEDIUM - Based on documentation warnings and common WebRTC deployment issues
- Egress DSGVO template: MEDIUM - Custom template approach is documented but selective blur/mute via metadata is our custom implementation
- Presence system: HIGH - Redis heartbeat is a well-established pattern, fits existing Redis infrastructure
- Emoji reactions: HIGH - Simple CRUD with WebSocket broadcast, follows existing chat patterns

**Research date:** 2026-02-11
**Valid until:** 2026-03-11 (30 days - LiveKit SDKs are stable, patterns well-established)
