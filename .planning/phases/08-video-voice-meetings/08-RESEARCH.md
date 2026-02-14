# Phase 8: Video, Voice & Meetings - Research

**Researched:** 2026-02-10
**Domain:** LiveKit WebRTC video/voice, meeting management, presence system, emoji reactions
**Confidence:** HIGH (LiveKit official docs verified, codebase patterns confirmed, architecture patterns from prior phases applied)

## Summary

Phase 8 integrates LiveKit (self-hostable WebRTC SFU) into the KMU Hub for 1:1 and group video/audio calls, screen sharing, and DSGVO-compliant recording. Additionally, it builds meeting management (scheduling, lobby, notes, action items), adds emoji reactions to chat messages, and implements user presence/online status across the entire application.

The architecture approach is: (1) a new **Video microservice** (`cmd/video/main.go`) that wraps LiveKit server-sdk-go for room management, token generation, recording egress, and meeting CRUD; (2) extensions to the existing **Chat service** for emoji reactions; (3) a **presence system** built on Redis + WebSocket hub for real-time online/away/offline/in-a-call status; (4) a **LiveKit server** added to docker-compose as a self-hosted dependency alongside a **LiveKit Egress** service for recording; and (5) React frontend components using `@livekit/components-react` for the video UI with custom meeting management views.

The most complex aspects are: (1) LiveKit Egress configuration for recording to MinIO with DSGVO consent tracking, (2) Electron-specific screen sharing requiring `desktopCapturer` API instead of standard `getDisplayMedia`, (3) presence system design that scales and integrates with the existing WebSocket hub, and (4) meeting management linking to calendar events from Phase 7 and CRM entities.

**Primary recommendation:** Create a dedicated Video microservice (gRPC on :50056) that owns room lifecycle, token generation, recording management, and meeting records. Use Redis sorted sets with TTL for presence. Extend the Chat proto with emoji reaction RPCs. Use `@livekit/components-react` PreJoin + VideoConference as the base UI, customized with KMU Hub styling. Deploy livekit-server and livekit-egress as Docker containers in docker-compose.

## Standard Stack

### Core (Backend - New Video Service + Chat Extension)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/livekit/server-sdk-go/v2 | v2.x (latest) | Room management, token generation, egress control | Official LiveKit Go SDK |
| github.com/livekit/protocol | latest | LiveKit protobuf types, auth package for JWT token generation | Required companion to server-sdk-go |
| google.golang.org/grpc | 1.78.0 | gRPC service for video/meeting domain | Already in use |
| github.com/jackc/pgx/v5 | 5.8.0 | PostgreSQL for meeting records, reactions, recordings | Already in use |
| github.com/redis/go-redis/v9 | 9.17.3 | Presence system (sorted sets with heartbeat TTL) | Already in use |
| github.com/google/uuid | 1.6.0 | UUIDs for meetings, recordings, reactions | Already in use |
| log/slog | stdlib | Structured logging | Already in use |

### Core (Frontend - React/TypeScript in Electron)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @livekit/components-react | ^2.9.x | Pre-built video conference UI components (VideoConference, PreJoin, ControlBar, GridLayout) | Official LiveKit React components |
| @livekit/components-styles | ^2.x | Default styles for LiveKit components (customizable with CSS variables) | Required companion to components-react |
| livekit-client | ^2.17.x | Low-level LiveKit client SDK (Room, Track, Participant APIs) | Required peer dependency |
| react | ^19.0.0 | UI framework | Already in use |
| @tanstack/react-query | ^5.x | Server state for meetings, recordings, presence | Already in use |
| zustand | ^5.x | Client state for call UI, presence cache | Already in use |
| @radix-ui/react-popover | installed | Emoji picker popover, meeting action popovers | Already in use |
| @radix-ui/react-dialog | installed | Meeting form dialog, recording consent dialog | Already in use |
| lucide-react | ^0.470 | Icons (Video, Phone, PhoneOff, Monitor, Mic, MicOff, Smile) | Already in use |

### Infrastructure (New Docker Services)

| Service | Image | Purpose | Why Needed |
|---------|-------|---------|------------|
| livekit-server | livekit/livekit-server:v1.9 | WebRTC SFU server for video/audio routing | Core video infrastructure |
| livekit-egress | livekit/egress | Recording/export service (headless Chrome + GStreamer) | Call recording to MinIO |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| emoji-mart (or similar) | latest | Emoji picker component for reactions | When building the reaction UI |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| LiveKit (self-hosted) | Twilio, Vonage, Daily.co | LiveKit is open-source, EU-hostable, no per-minute fees; alternatives are SaaS-only with US data processing |
| New Video microservice | Extend Chat or Work service | Video has distinct lifecycle (rooms, recordings, egress), separate concerns warrant own service. Meeting records could live in Work but room management needs dedicated ownership |
| Redis for presence | PostgreSQL polling | Redis sorted sets give O(log N) updates + O(1) TTL expiry; PostgreSQL polling is too slow for real-time presence |
| @livekit/components-react | Custom WebRTC UI from scratch | Components provide grid layout, participant tiles, device selection, screen share controls out of the box. Customizable via CSS variables + composition |

### Installation

```bash
# Backend (new Go dependencies)
cd backend
go get github.com/livekit/server-sdk-go/v2
go get github.com/livekit/protocol

# Frontend (new JS dependencies)
cd desktop
npm install @livekit/components-react @livekit/components-styles livekit-client
```

## Architecture Patterns

### Recommended Backend Structure (New Video Service)

```
backend/
  cmd/
    video/
      main.go                    # Video service entry point
  internal/
    video/
      room/
        errors.go
        repository.go            # Call/room log persistence
        postgres_repository.go
        service.go               # LiveKit room create/delete, token generation
        service_test.go
      recording/
        errors.go
        repository.go            # Recording metadata persistence
        postgres_repository.go
        service.go               # Egress start/stop, consent tracking, MinIO config
        service_test.go
      meeting/
        errors.go
        repository.go            # Meeting CRUD, notes, action items
        postgres_repository.go
        service.go               # Meeting lifecycle, calendar linking
        service_test.go
      presence/
        errors.go
        service.go               # Redis presence updates, heartbeat processing
        service_test.go
    chat/
      reaction/                  # NEW: extends Chat service
        errors.go
        repository.go            # Reaction CRUD
        postgres_repository.go
        service.go               # Add/remove reactions, aggregate counts
        service_test.go
  proto/
    video/
      v1/
        video.proto              # New proto for video/meeting RPCs
    chat/
      v1/
        chat.proto               # Extended with reaction RPCs
  internal/
    gateway/
      route_video.go             # HTTP routes for video/meeting endpoints
    models/
      call.go                    # Call, CallParticipant
      recording.go               # Recording, RecordingConsent
      meeting.go                 # Meeting, MeetingNote, MeetingActionItem
      reaction.go                # MessageReaction
      presence.go                # UserPresence
  Dockerfile.video               # Docker build for video service
```

### Recommended Frontend Structure

```
desktop/src/renderer/src/
  modules/
    video/
      VideoCallView.tsx           # Full-screen call UI (wraps LiveKit components)
      PreJoinScreen.tsx           # Camera/mic preview before joining
      CallControls.tsx            # Mute, camera, screen share, record, hang up
      ParticipantGrid.tsx         # Gallery view for group calls
      ScreenShareView.tsx         # Full-screen shared content + pip participants
      RecordingConsentDialog.tsx  # DSGVO consent before recording starts
      CallNotification.tsx        # Incoming call notification overlay
      FloatingCallBar.tsx         # Minimized call indicator when navigating away
    meetings/
      MeetingScheduleForm.tsx     # Create/edit meeting (links to calendar)
      MeetingLobby.tsx            # Pre-meeting view with agenda, attendees, docs
      MeetingNotesEditor.tsx      # Shared/private notes during meeting
      MeetingActionItems.tsx      # Action items list with assignees
      MeetingSummary.tsx          # Post-meeting summary card
      MeetingListPage.tsx         # All meetings list with filters
    chat/
      reactions/
        ReactionPicker.tsx        # Emoji picker popover (on message hover)
        ReactionBadge.tsx         # Individual reaction badge (+count)
        ReactionBar.tsx           # Row of reactions under a message
    presence/
      PresenceIndicator.tsx       # Dot indicator (green/yellow/grey/red)
      PresenceProvider.tsx        # Context provider managing presence state
      usePresence.ts              # Hook for reading user presence
  api/
    hooks/
      useCalls.ts                 # TanStack Query hooks for call operations
      useMeetings.ts              # Meeting CRUD hooks
      useRecordings.ts            # Recording management hooks
      useReactions.ts             # Reaction CRUD hooks
      usePresence.ts              # Presence subscription hooks
  stores/
    call.ts                       # Zustand: active call state, minimized state
    presence.ts                   # Zustand: presence cache for all users
```

### Pattern 1: LiveKit Room Creation and Token Generation (Backend)

**What:** Server creates LiveKit rooms and generates join tokens for participants.
**When to use:** Any call initiation (1:1, group, from chat, from meeting).

```go
// internal/video/room/service.go
// Source: https://github.com/livekit/server-sdk-go (official README)
import (
    lksdk "github.com/livekit/server-sdk-go/v2"
    "github.com/livekit/protocol/auth"
    "github.com/livekit/protocol/livekit"
)

type Service struct {
    roomClient *lksdk.RoomServiceClient
    apiKey     string
    apiSecret  string
    wsURL      string
    repo       Repository
}

func NewService(host, apiKey, apiSecret string, repo Repository) *Service {
    return &Service{
        roomClient: lksdk.NewRoomServiceClient(host, apiKey, apiSecret),
        apiKey:     apiKey,
        apiSecret:  apiSecret,
        wsURL:      host,
        repo:       repo,
    }
}

func (s *Service) CreateRoom(ctx context.Context, roomName string, maxParticipants uint32) (*livekit.Room, error) {
    room, err := s.roomClient.CreateRoom(ctx, &livekit.CreateRoomRequest{
        Name:            roomName,
        MaxParticipants: maxParticipants,
        EmptyTimeout:    300, // 5 min timeout when empty
    })
    if err != nil {
        return nil, fmt.Errorf("create livekit room: %w", err)
    }
    return room, nil
}

func (s *Service) GenerateJoinToken(roomName, userID, userName string) (string, error) {
    at := auth.NewAccessToken(s.apiKey, s.apiSecret)
    grant := &auth.VideoGrant{
        RoomJoin: true,
        Room:     roomName,
    }
    at.SetVideoGrant(grant).
        SetIdentity(userID).
        SetName(userName).
        SetValidFor(4 * time.Hour)
    return at.ToJWT()
}

// Room naming: "call-{initiatorID[:8]}-{timestamp}" for uniqueness
func CallRoomName(initiatorID uuid.UUID) string {
    return fmt.Sprintf("call-%s-%d", initiatorID.String()[:8], time.Now().Unix())
}
```

### Pattern 2: Recording with DSGVO Consent (Backend)

**What:** Start/stop recordings only after all participants consent, store to MinIO.
**When to use:** When any participant requests recording.

```go
// internal/video/recording/service.go
// Source: https://docs.livekit.io/home/egress/api/
import (
    lksdk "github.com/livekit/server-sdk-go/v2"
    "github.com/livekit/protocol/livekit"
)

type Service struct {
    egressClient *lksdk.EgressClient
    repo         Repository
    minioCfg     MinIOConfig
}

type MinIOConfig struct {
    Endpoint  string
    AccessKey string
    SecretKey string
    Bucket    string
}

func (s *Service) StartRecording(ctx context.Context, roomName string, callID uuid.UUID) (*livekit.EgressInfo, error) {
    req := &livekit.RoomCompositeEgressRequest{
        RoomName: roomName,
        Layout:   "grid",
        FileOutputs: []*livekit.EncodedFileOutput{
            {
                FileType: livekit.EncodedFileType_MP4,
                Filepath: fmt.Sprintf("recordings/%s/%s.mp4", callID, time.Now().Format("2006-01-02")),
                Output: &livekit.EncodedFileOutput_S3{
                    S3: &livekit.S3Upload{
                        AccessKey:      s.minioCfg.AccessKey,
                        Secret:         s.minioCfg.SecretKey,
                        Bucket:         s.minioCfg.Bucket,
                        Endpoint:       s.minioCfg.Endpoint,
                        ForcePathStyle: true, // Required for MinIO
                    },
                },
            },
        },
    }
    return s.egressClient.StartRoomCompositeEgress(ctx, req)
}

func (s *Service) StopRecording(ctx context.Context, egressID string) (*livekit.EgressInfo, error) {
    return s.egressClient.StopEgress(ctx, &livekit.StopEgressRequest{
        EgressId: egressID,
    })
}
```

### Pattern 3: Presence via Redis Sorted Sets + WebSocket

**What:** Track user online/away/offline/in-a-call status using Redis heartbeats.
**When to use:** Presence indicator on every user avatar across the app.

```go
// internal/video/presence/service.go
// Source: systemdesign.one/real-time-presence-platform-system-design (pattern), Redis docs (implementation)
type Status string

const (
    StatusOnline  Status = "online"
    StatusAway    Status = "away"
    StatusOffline Status = "offline"
    StatusInCall  Status = "in_call"
)

const (
    presenceKey     = "presence:heartbeats"
    presenceTimeout = 60 * time.Second  // Offline after 60s without heartbeat
    awayTimeout     = 300 * time.Second // Away after 5 min idle
)

type Service struct {
    redis *redis.Client
}

func (s *Service) Heartbeat(ctx context.Context, userID string) error {
    return s.redis.ZAdd(ctx, presenceKey, redis.Z{
        Score:  float64(time.Now().Unix()),
        Member: userID,
    }).Err()
}

func (s *Service) SetInCall(ctx context.Context, userID string, inCall bool) error {
    key := fmt.Sprintf("presence:incall:%s", userID)
    if inCall {
        return s.redis.Set(ctx, key, "1", 4*time.Hour).Err()
    }
    return s.redis.Del(ctx, key).Err()
}

func (s *Service) GetBulkStatus(ctx context.Context, userIDs []string) (map[string]Status, error) {
    result := make(map[string]Status, len(userIDs))
    pipe := s.redis.Pipeline()
    cmds := make(map[string]*redis.FloatCmd, len(userIDs))
    inCallCmds := make(map[string]*redis.IntCmd, len(userIDs))

    for _, uid := range userIDs {
        cmds[uid] = pipe.ZScore(ctx, presenceKey, uid)
        inCallCmds[uid] = pipe.Exists(ctx, fmt.Sprintf("presence:incall:%s", uid))
    }
    pipe.Exec(ctx)

    for _, uid := range userIDs {
        if inCallCmds[uid].Val() > 0 {
            result[uid] = StatusInCall
            continue
        }
        score, err := cmds[uid].Result()
        if err == redis.Nil {
            result[uid] = StatusOffline
            continue
        }
        lastSeen := time.Unix(int64(score), 0)
        elapsed := time.Since(lastSeen)
        switch {
        case elapsed > presenceTimeout:
            result[uid] = StatusOffline
        case elapsed > awayTimeout:
            result[uid] = StatusAway
        default:
            result[uid] = StatusOnline
        }
    }
    return result, nil
}
```

### Pattern 4: Emoji Reactions (Chat Service Extension)

**What:** Add/remove emoji reactions on messages, aggregate counts.
**When to use:** Any chat message.

```go
// internal/chat/reaction/service.go
type ReactionSummary struct {
    Emoji     string   `json:"emoji"`
    Count     int      `json:"count"`
    Users     []string `json:"users"`
    MeReacted bool     `json:"me_reacted"`
}

func (s *Service) ToggleReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (added bool, err error) {
    existing, err := s.repo.GetReaction(ctx, messageID, userID, emoji)
    if err != nil && !errors.Is(err, ErrReactionNotFound) {
        return false, err
    }
    if existing != nil {
        return false, s.repo.DeleteReaction(ctx, existing.ID)
    }
    _, err = s.repo.CreateReaction(ctx, messageID, userID, emoji)
    return true, err
}

func (s *Service) GetReactionSummaries(ctx context.Context, messageID, currentUserID uuid.UUID) ([]ReactionSummary, error) {
    reactions, err := s.repo.ListReactionsByMessage(ctx, messageID)
    if err != nil {
        return nil, err
    }
    // Group by emoji, count users, check if current user reacted
    summaries := make(map[string]*ReactionSummary)
    for _, r := range reactions {
        if s, ok := summaries[r.Emoji]; ok {
            s.Count++
            s.Users = append(s.Users, r.UserID.String())
            if r.UserID == currentUserID {
                s.MeReacted = true
            }
        } else {
            summaries[r.Emoji] = &ReactionSummary{
                Emoji:     r.Emoji,
                Count:     1,
                Users:     []string{r.UserID.String()},
                MeReacted: r.UserID == currentUserID,
            }
        }
    }
    result := make([]ReactionSummary, 0, len(summaries))
    for _, s := range summaries {
        result = append(result, *s)
    }
    return result, nil
}
```

### Pattern 5: Electron Screen Sharing with desktopCapturer

**What:** Electron does NOT support standard `getDisplayMedia`. Must use Electron's `desktopCapturer` API.
**When to use:** Screen sharing in the Electron desktop app.

```typescript
// In Electron main process (preload)
// Source: Electron documentation + LiveKit Electron integration guidance
const { desktopCapturer } = require('electron');

contextBridge.exposeInMainWorld('electronScreenShare', {
  getSources: () => desktopCapturer.getSources({
    types: ['window', 'screen'],
    thumbnailSize: { width: 320, height: 180 },
  }),
});

// In React renderer - custom screen share handler
async function startScreenShare(room: Room) {
  const sources = await window.electronScreenShare.getSources();
  const selectedSource = await showScreenPickerDialog(sources);
  if (!selectedSource) return;

  const stream = await navigator.mediaDevices.getUserMedia({
    audio: false,
    video: {
      // @ts-ignore - Electron-specific constraint
      mandatory: {
        chromeMediaSource: 'desktop',
        chromeMediaSourceId: selectedSource.id,
      },
    },
  });

  const track = stream.getVideoTracks()[0];
  await room.localParticipant.publishTrack(track, {
    name: 'screen_share',
    source: Track.Source.ScreenShare,
  });
}
```

### Anti-Patterns to Avoid

- **Running LiveKit in the same process as the Go backend**: LiveKit server is a separate binary. Deploy as a Docker container alongside your services.
- **Storing presence in PostgreSQL**: Do NOT poll PostgreSQL for presence status. Use Redis with sorted sets.
- **Starting recordings without consent tracking**: Do NOT start egress without verifying all participants have consented. Store consent records with timestamps for DSGVO audit trail.
- **Using getDisplayMedia in Electron**: Electron does NOT support the standard browser `getDisplayMedia()` API. Use Electron's `desktopCapturer` API instead.
- **Bundling call state in the Chat service**: Video calls have distinct lifecycle, recording, and participant management. Keep them in a separate Video service.
- **Ignoring LiveKit room cleanup**: LiveKit rooms persist until empty_timeout. Track room lifecycle in your database and handle cleanup.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WebRTC SFU (media routing) | Custom WebRTC server | LiveKit server (livekit/livekit-server Docker image) | WebRTC SFU is extremely complex (ICE, DTLS, SRTP, simulcast, SVC). LiveKit handles all of this. |
| Video conference UI | Custom video grid, device selector, controls | @livekit/components-react (VideoConference, PreJoin, ControlBar, GridLayout) | Layout algorithms, track subscription, reconnection handling are built-in |
| Call recording/export | Custom ffmpeg pipeline | LiveKit Egress service (livekit/egress Docker image) | Headless Chrome compositor + GStreamer encoding with S3/MinIO output |
| Access token generation | Custom JWT for video | github.com/livekit/protocol/auth | Token format must match LiveKit server expectations (VideoGrant structure) |
| Presence heartbeat cleanup | Custom cron job | Redis sorted set TTL + ZRANGEBYSCORE cleanup | Redis handles expiry natively; sorted sets enable range queries |
| Emoji picker UI | Custom emoji grid | emoji-mart or similar battle-tested picker | Unicode emoji rendering, search, categories, skin tone variants are complex |

**Key insight:** Video/voice infrastructure is the most "don't hand-roll" domain in this entire project. LiveKit provides the SFU, recording, and UI components. The application code should focus on business logic: meeting management, consent tracking, presence, and integration with existing Chat/Calendar services.

## Common Pitfalls

### Pitfall 1: Electron Screen Sharing Incompatibility

**What goes wrong:** Screen sharing silently fails or shows a blank stream in the Electron app.
**Why it happens:** Electron's Chromium does not implement the standard `getDisplayMedia()` API. LiveKit's default screen share uses this API.
**How to avoid:** Use Electron's `desktopCapturer` API to enumerate screens/windows, present a custom picker dialog, then create a MediaStream with `chromeMediaSource: 'desktop'` constraint. Publish the resulting track to LiveKit manually instead of using `setScreenShareEnabled(true)`.
**Warning signs:** Black screen share, browser permission dialog not appearing, "NotAllowedError" in console.

### Pitfall 2: LiveKit Egress Chrome Sandboxing in Docker

**What goes wrong:** Recording fails to start, egress container crashes.
**Why it happens:** Since egress v1.7.6, Chrome sandboxing requires `SYS_ADMIN` capability in Docker.
**How to avoid:** Add `cap_add: [SYS_ADMIN]` to the egress service in docker-compose.yml. Allocate at least 4 CPUs and 4 GB RAM per egress instance.
**Warning signs:** Egress container exits immediately, "Failed to move to new namespace" errors in logs.

### Pitfall 3: DSGVO Recording Consent Race Condition

**What goes wrong:** Recording starts before all participants have consented, violating GDPR/DSGVO.
**Why it happens:** Consent is collected asynchronously via WebSocket. A participant might join after recording request but before consent check completes.
**How to avoid:** Implement a consent state machine: (1) Recording requested -> (2) All current participants notified -> (3) Each participant responds (consent/deny) -> (4) If all consent, start egress -> (5) If any deny, cancel. New participants joining during an active recording must consent before their tracks are included. Store consent records with timestamps in `recording_consents` table.
**Warning signs:** Recording starts with only partial consent, no audit trail for consent decisions.

### Pitfall 4: Presence Heartbeat Flood

**What goes wrong:** Hundreds of users sending heartbeats every 5 seconds overwhelm the gateway and Redis.
**Why it happens:** Too-frequent heartbeats or heartbeats sent as HTTP requests instead of WebSocket messages.
**How to avoid:** Send heartbeats via the existing WebSocket connection (not separate HTTP calls). Use 30-second intervals (not 5 seconds). Use Redis pipeline for bulk operations. Batch presence queries (get all user statuses for a channel in one call, not per-user).
**Warning signs:** Redis CPU spikes, WebSocket message backlog, gateway memory growth.

### Pitfall 5: LiveKit Room Persistence Mismatch

**What goes wrong:** Application thinks a call is active but the LiveKit room was auto-deleted, or vice versa.
**Why it happens:** LiveKit rooms have `empty_timeout` (auto-delete when empty). If the application DB says "call active" but LiveKit deleted the room, participants get errors.
**How to avoid:** Use LiveKit webhooks or periodic `ListRooms` polling to sync room state. Set a reasonable `empty_timeout` (5 min). When a room is destroyed by LiveKit, update the call record in PostgreSQL. Treat LiveKit as the source of truth for room existence.
**Warning signs:** "Room not found" errors when trying to join, stale "active call" indicators.

### Pitfall 6: N+1 Presence Queries

**What goes wrong:** Rendering a channel member list with presence indicators makes one Redis query per user.
**Why it happens:** Presence is checked per-user instead of in bulk.
**How to avoid:** Use Redis PIPELINE to fetch all member presence statuses in a single round-trip. Expose a bulk endpoint `GetBulkPresence(userIDs)`. Cache presence results on the frontend (zustand store, refresh every 30s).
**Warning signs:** Channel view taking > 1 second to show presence, Redis connection pool exhaustion.

### Pitfall 7: Missing Meeting-Calendar Link Consistency

**What goes wrong:** A meeting is scheduled but no calendar event is created, or vice versa.
**Why it happens:** Meeting and calendar are in different services (Video vs Work). No transactional guarantee across services.
**How to avoid:** Use eventual consistency: meeting creation emits a pg_notify event, which the Work/Calendar service picks up to create the calendar event. Store the calendar_event_id on the meeting record. If calendar event creation fails, retry with backoff.
**Warning signs:** Meetings without calendar events, orphaned calendar events after meeting deletion.

## Code Examples

### Database Migration: Calls and Recordings

```sql
-- Migration: 000032_create_calls.up.sql

CREATE TABLE calls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    room_name VARCHAR(100) NOT NULL UNIQUE,
    call_type VARCHAR(20) NOT NULL DEFAULT 'direct',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    initiated_by UUID NOT NULL REFERENCES users(id),
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    meeting_id UUID,
    max_participants INTEGER NOT NULL DEFAULT 25,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calls_status ON calls (status) WHERE status = 'active';
CREATE INDEX idx_calls_channel ON calls (channel_id) WHERE channel_id IS NOT NULL;
CREATE INDEX idx_calls_initiated_by ON calls (initiated_by);

CREATE TABLE call_participants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at TIMESTAMPTZ,
    had_video BOOLEAN NOT NULL DEFAULT false,
    had_audio BOOLEAN NOT NULL DEFAULT true,
    had_screen_share BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (call_id, user_id, joined_at)
);

CREATE INDEX idx_call_participants_call ON call_participants (call_id);
CREATE INDEX idx_call_participants_user ON call_participants (user_id);

CREATE TABLE call_recordings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    egress_id VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'recording',
    file_path VARCHAR(500),
    file_size BIGINT,
    duration_seconds INTEGER,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_call_recordings_call ON call_recordings (call_id);

CREATE TABLE recording_consents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    consented BOOLEAN NOT NULL,
    responded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (call_id, user_id)
);

CREATE INDEX idx_recording_consents_call ON recording_consents (call_id);
```

### Database Migration: Meetings

```sql
-- Migration: 000033_create_meetings.up.sql

CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    agenda TEXT,
    meeting_type VARCHAR(20) NOT NULL DEFAULT 'standard',
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled',
    scheduled_start TIMESTAMPTZ NOT NULL,
    scheduled_end TIMESTAMPTZ NOT NULL,
    actual_start TIMESTAMPTZ,
    actual_end TIMESTAMPTZ,
    calendar_event_id UUID,
    call_id UUID REFERENCES calls(id) ON DELETE SET NULL,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    linked_entity_type VARCHAR(20),
    linked_entity_id UUID,
    livekit_room_name VARCHAR(100),
    organized_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meetings_status ON meetings (status);
CREATE INDEX idx_meetings_scheduled ON meetings (scheduled_start, scheduled_end);
CREATE INDEX idx_meetings_organized_by ON meetings (organized_by);
CREATE INDEX idx_meetings_calendar_event ON meetings (calendar_event_id)
    WHERE calendar_event_id IS NOT NULL;
CREATE INDEX idx_meetings_linked_entity ON meetings (linked_entity_type, linked_entity_id)
    WHERE linked_entity_type IS NOT NULL;

CREATE TABLE meeting_attendees (
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rsvp_status VARCHAR(10) NOT NULL DEFAULT 'pending',
    role VARCHAR(20) NOT NULL DEFAULT 'attendee',
    responded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (meeting_id, user_id)
);

CREATE INDEX idx_meeting_attendees_user ON meeting_attendees (user_id);

CREATE TABLE meeting_notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_private BOOLEAN NOT NULL DEFAULT false,
    author_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meeting_notes_meeting ON meeting_notes (meeting_id);

CREATE TABLE meeting_action_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    assigned_to UUID REFERENCES users(id),
    due_date DATE,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meeting_action_items_meeting ON meeting_action_items (meeting_id);
CREATE INDEX idx_meeting_action_items_assigned ON meeting_action_items (assigned_to)
    WHERE assigned_to IS NOT NULL;
```

### Database Migration: Emoji Reactions

```sql
-- Migration: 000034_create_message_reactions.up.sql

CREATE TABLE message_reactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (message_id, user_id, emoji)
);

CREATE INDEX idx_message_reactions_message ON message_reactions (message_id);
CREATE INDEX idx_message_reactions_user ON message_reactions (user_id);
```

### Docker Compose: LiveKit Server and Egress

```yaml
# Add to deploy/docker/docker-compose.yml

  livekit:
    image: livekit/livekit-server:v1.9
    command: --dev --bind 0.0.0.0
    ports:
      - "7880:7880"
      - "7881:7881"
      - "50100-50200:50100-50200/udp"
    environment:
      LIVEKIT_KEYS: "devkey: secret"
    healthcheck:
      test: ["CMD-SHELL", "wget --no-verbose --spider --tries=1 http://localhost:7880 || exit 1"]
      interval: 5s
      timeout: 5s
      retries: 5

  livekit-egress:
    image: livekit/egress:latest
    cap_add:
      - SYS_ADMIN
    environment:
      EGRESS_CONFIG_BODY: |
        api_key: devkey
        api_secret: secret
        ws_url: ws://livekit:7880
        insecure: true
        s3:
          access_key: minioadmin
          secret: minioadmin
          endpoint: http://minio:9000
          bucket: kmuhub-files
          force_path_style: true
        redis:
          address: redis:6379
    depends_on:
      livekit:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy
```

### Config Extension for LiveKit

```go
// Add to backend/internal/config/config.go

// LiveKit
LiveKitHost      string `env:"LIVEKIT_HOST,default=http://localhost:7880"`
LiveKitWSURL     string `env:"LIVEKIT_WS_URL,default=ws://localhost:7880"`
LiveKitAPIKey    string `env:"LIVEKIT_API_KEY,default=devkey"`
LiveKitAPISecret string `env:"LIVEKIT_API_SECRET,default=secret"`

// Video Service
VideoGRPCPort    string `env:"VIDEO_GRPC_PORT,default=:50056"`
VideoGRPCAddress string `env:"VIDEO_GRPC_ADDRESS,default=localhost:50056"`
VideoHealthPort  string `env:"VIDEO_HEALTH_PORT,default=:9096"`
```

### WebSocket Extensions for Presence and Calls

```go
// Add to WebSocket message types in websocket.go

// Presence (Client -> Server)
WSPresenceHeartbeat = "presence.heartbeat"

// Presence (Server -> Client)
WSPresenceUpdate = "presence.update"

// Calls (Server -> Client)
WSCallIncoming              = "call.incoming"
WSCallAccepted              = "call.accepted"
WSCallRejected              = "call.rejected"
WSCallEnded                 = "call.ended"
WSCallRecordingStarted      = "call.recording.started"
WSCallRecordingConsentReq   = "call.recording.consent_required"

// Reactions (Server -> Client)
WSReactionAdded   = "reaction.added"
WSReactionRemoved = "reaction.removed"
```

### Frontend: LiveKit Video Call View

```typescript
// modules/video/VideoCallView.tsx
// Source: https://docs.livekit.io/home/quickstarts/react/
import {
  LiveKitRoom,
  VideoConference,
  RoomAudioRenderer,
} from '@livekit/components-react';
import '@livekit/components-styles';

interface VideoCallViewProps {
  serverUrl: string;
  token: string;
  onDisconnected: () => void;
}

function VideoCallView({ serverUrl, token, onDisconnected }: VideoCallViewProps) {
  return (
    <LiveKitRoom
      serverUrl={serverUrl}
      token={token}
      onDisconnected={onDisconnected}
      data-lk-theme="default"
      style={{ height: '100vh' }}
    >
      <VideoConference />
      <RoomAudioRenderer />
    </LiveKitRoom>
  );
}
```

### Frontend: PreJoin Screen

```typescript
// modules/video/PreJoinScreen.tsx
// Source: https://docs.livekit.io/reference/components/react/component/prejoin/
import { PreJoin } from '@livekit/components-react';

interface PreJoinScreenProps {
  onSubmit: (values: { audioEnabled: boolean; videoEnabled: boolean }) => void;
}

function PreJoinScreen({ onSubmit }: PreJoinScreenProps) {
  return (
    <div className="flex items-center justify-center h-full">
      <PreJoin
        onSubmit={(values) => {
          onSubmit({
            audioEnabled: values.audioEnabled,
            videoEnabled: values.videoEnabled,
          });
        }}
        defaults={{
          username: '', // Set from auth context
          videoEnabled: true,
          audioEnabled: true,
        }}
      />
    </div>
  );
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Peer-to-peer WebRTC (no server) | SFU (Selective Forwarding Unit) via LiveKit | 2021+ | Scales beyond 4 participants, lower client bandwidth |
| Custom recording with ffmpeg | LiveKit Egress (headless Chrome + GStreamer) | 2022+ | Handles composition, encoding, storage automatically |
| Polling-based presence (HTTP) | WebSocket heartbeat + Redis sorted sets | Standard practice | Real-time presence with sub-second updates |
| Per-message reaction column (JSON) | Separate reactions table with unique constraint | Standard practice | Efficient aggregation, proper indexing, toggle semantics |
| getDisplayMedia for screen share | Electron desktopCapturer API | Electron requirement | Browser API not available in Electron context |
| @livekit/react-components (old) | @livekit/components-react (current) | 2023+ | Rewritten component library with better composition |

**Deprecated/outdated:**
- `@livekit/react-components` (old package): Replaced by `@livekit/components-react`. Do not install the old package.
- `livekit-react` (old package): Deprecated. Use `@livekit/components-react` instead.
- LiveKit Egress without SYS_ADMIN: As of v1.7.6, Chrome sandboxing requires SYS_ADMIN Docker capability.

## Open Questions

1. **LiveKit server version pinning for production**
   - What we know: LiveKit server is at v1.9.11 (January 2026). The `--dev` flag provides default keys for local development.
   - What's unclear: Production deployment requires SSL, TURN server, and proper key management.
   - Recommendation: Use `--dev` mode for local development. Document production deployment requirements separately.

2. **Egress service scaling for concurrent recordings**
   - What we know: Each egress instance records one room at a time. 4 CPUs + 4 GB RAM per instance.
   - What's unclear: How many simultaneous recordings needed?
   - Recommendation: Start with 1 egress instance in docker-compose. Production needs autoscaling.

3. **Emoji picker library choice**
   - What we know: emoji-mart is the most popular. Lighter alternatives exist (emoji-picker-react).
   - What's unclear: Whether emoji-mart works with React 19 and Electron.
   - Recommendation: Test emoji-mart first. Fall back to emoji-picker-react. The picker is swappable.

4. **Presence system and WebSocket hub scaling**
   - What we know: Current WebSocket hub is in-memory (single gateway).
   - What's unclear: When multi-gateway will be needed.
   - Recommendation: Implement via existing single-gateway WebSocket hub. Design Redis store to be gateway-agnostic for future multi-gateway support.

5. **Call signaling: WebSocket vs LiveKit native**
   - What we know: LiveKit handles WebRTC signaling. Call initiation (ring, accept, reject) is application-level.
   - What's unclear: Best channel for application-level signaling.
   - Recommendation: Use our WebSocket hub for call signaling (incoming, accept, reject, end). LiveKit handles WebRTC signaling separately.

6. **@livekit/components-react React 19 compatibility**
   - What we know: Package v2.9.19 was published January 2026. React 19 has been out since late 2024.
   - What's unclear: Whether the package explicitly supports React 19 in peer dependencies.
   - Recommendation: Test during installation. If peer dependency conflicts, use `--legacy-peer-deps` flag or check for newer version. LiveKit actively maintains this package.

## Sources

### Primary (HIGH confidence)
- LiveKit official documentation (docs.livekit.io) - Egress API, room management, React components, screen sharing, self-hosting
- LiveKit GitHub server-sdk-go README (github.com/livekit/server-sdk-go) - Go SDK API, token generation, room client
- LiveKit components-react npm (v2.9.19) - React component library
- livekit-client npm (v2.17.0) - Client SDK
- LiveKit server GitHub releases (v1.9.11, January 2026) - Server version
- Codebase analysis - All existing patterns (WebSocket hub, config, docker-compose, proto, models, gateway routes)

### Secondary (MEDIUM confidence)
- Presence system design articles (systemdesign.one) - Redis sorted set pattern for heartbeats
- LiveKit Egress Docker configuration (docs.livekit.io/transport/self-hosting/egress/) - SYS_ADMIN requirement, config.yaml format
- Electron desktopCapturer documentation - Screen sharing in Electron context

### Tertiary (LOW confidence)
- emoji-mart React 19 compatibility - Not verified, needs testing
- LiveKit + Electron integration specifics - Limited official documentation; pattern based on general Electron WebRTC guidance
- Exact server-sdk-go/v2 version number - pkg.go.dev was not accessible; using "latest" with go get

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - LiveKit is explicitly chosen in CLAUDE.md tech stack; all SDK versions verified via npm/GitHub
- Architecture: HIGH - New microservice follows exact same pattern as auth/crm/chat/notification/work services
- LiveKit integration: HIGH - Official docs verified for Go SDK, React components, Egress, self-hosting
- Presence system: MEDIUM - Redis sorted set pattern is well-established but timeout values need tuning
- Electron screen sharing: MEDIUM - Electron limitation documented, but LiveKit-specific Electron integration has limited official docs
- Meeting management: HIGH - Follows established CRM/Calendar patterns
- Emoji reactions: HIGH - Simple extension to existing Chat service; standard database pattern
- DSGVO recording consent: MEDIUM - Technical approach clear, legal specifics need validation

**Research date:** 2026-02-10
**Valid until:** 2026-03-10 (LiveKit has frequent releases; verify SDK versions before implementation)
