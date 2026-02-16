---
phase: 08-video-voice-meetings
plan: 05
subsystem: api
tags: [grpc, gateway, websocket, livekit, docker, openapi, chi, protobuf]

# Dependency graph
requires:
  - phase: 08-02
    provides: Emoji reaction service (reaction package)
  - phase: 08-03
    provides: Call and recording services with LiveKit/Egress integration
  - phase: 08-04
    provides: Meeting lifecycle, notes, action items, presence services
provides:
  - VideoGRPCServer with ~30 RPCs bridging domain services to transport
  - Gateway HTTP routes (~35 endpoints) for video, meetings, recordings, presence, reactions
  - WebSocket hub extensions with presence/call/reaction real-time events
  - LiveKit + Egress Docker services with dev config
  - OpenAPI spec covering all new endpoints
affects: [08-07, 08-08, 08-09, 09-security, desktop-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "VideoRoutes shares gRPC connection with WorkRoutes via ServiceName 'work'"
    - "WSPresenceService/WSVideoService interfaces for dependency injection without circular imports"
    - "Presence subscriber tracking in WebSocket hub with cleanup on disconnect"

key-files:
  created:
    - backend/internal/server/video_grpc.go
    - backend/internal/gateway/route_video.go
    - deploy/docker/livekit.yaml
    - deploy/docker/egress.yaml
  modified:
    - backend/cmd/work/main.go
    - backend/cmd/gateway/main.go
    - backend/internal/config/config.go
    - backend/internal/server/websocket.go
    - backend/api/openapi.yaml
    - deploy/docker/docker-compose.yml

key-decisions:
  - "VideoRoutes ServiceName returns 'work' to share gRPC connection (same binary)"
  - "Reaction HTTP endpoints return 501 (reactions handled via WebSocket events, not in video.proto)"
  - "LiveKit webhook handler does JSON parsing and logging; JWT signature validation deferred to library integration"
  - "WSPresenceService/WSVideoService interfaces injected after construction to avoid circular imports"
  - "LiveKit SDK types live in github.com/livekit/protocol/livekit, not in the SDK package itself"

patterns-established:
  - "Optional WebSocket service injection via SetPresenceService/SetVideoService pattern"
  - "Presence subscriber tracking: users subscribe to other users' presence updates"
  - "cleanupPresenceSubscriptions called on disconnect to prevent stale subscriptions"

# Metrics
duration: ~15min
completed: 2026-02-11
---

# Phase 8 Plan 5: Video/Meeting gRPC + Gateway + WebSocket + Docker Wiring Summary

**VideoGRPCServer with ~30 RPCs, gateway with 35 HTTP routes, WebSocket hub with 9 new event types, LiveKit/Egress Docker services, and full OpenAPI spec**

## Performance

- **Duration:** ~15 min (across context resets)
- **Tasks:** 3
- **Files created:** 4
- **Files modified:** 6

## Accomplishments
- VideoGRPCServer implements all ~30 RPCs from video.proto covering calls, recordings, meetings, notes, action items, and presence
- Gateway route_video.go registers 35 HTTP endpoints under /api/v1/video/*, /api/v1/meetings/*, and /api/v1/webhooks/livekit
- WebSocket hub extended with 9 new message types and presence subscriber tracking
- Docker Compose updated with LiveKit server, LiveKit Egress, and full work service environment variables
- OpenAPI spec updated with 26 new path entries and comprehensive schemas

## Task Commits

Each task was committed atomically:

1. **Task 1: gRPC server + Work binary registration** - `169f92e` (feat)
2. **Task 2: Gateway routes + LiveKit webhook + Docker Compose + config** - `648d7d9` (feat)
3. **Task 3: WebSocket extensions + OpenAPI spec** - `8819191` (feat)

## Files Created/Modified
- `backend/internal/server/video_grpc.go` - VideoGRPCServer implementing all VideoService RPCs with proto converters
- `backend/internal/gateway/route_video.go` - 35 HTTP routes following RouteRegistrar pattern
- `backend/cmd/work/main.go` - Redis client, video domain services, VideoGRPCServer registration
- `backend/cmd/gateway/main.go` - VideoRoutes added to registrars list
- `backend/internal/config/config.go` - LiveKitWebhookSecret field added
- `backend/internal/server/websocket.go` - 9 new message types, presence/call/reaction handlers and broadcasts
- `backend/api/openapi.yaml` - 26 new paths, 30+ new schemas for video/meeting/recording/presence/reaction
- `deploy/docker/docker-compose.yml` - LiveKit, Egress services; work service Redis/LiveKit/MinIO env vars
- `deploy/docker/livekit.yaml` - LiveKit server dev config (ports, Redis, keys, webhooks)
- `deploy/docker/egress.yaml` - Egress dev config (API keys, Redis, S3/MinIO)

## Decisions Made
- VideoRoutes returns ServiceName "work" to share the existing gRPC connection with WorkRoutes/CalendarRoutes
- Reaction HTTP handlers return 501 Not Implemented because reactions are not in video.proto -- they are event-driven via WebSocket
- LiveKit webhook handler parses JSON and logs events; full JWT signature validation with livekit/protocol/webhook deferred
- WSPresenceService and WSVideoService interfaces use dependency injection to avoid circular imports between server and domain packages
- Fixed LiveKit SDK type imports: RoomCompositeEgressRequest etc. live in github.com/livekit/protocol/livekit, not in the SDK package

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed LiveKit SDK type imports in room_manager.go and egress_manager.go**
- **Found during:** Task 1 (build verification)
- **Issue:** Code used `lksdk.RoomCompositeEgressRequest` but these types are in `lkproto "github.com/livekit/protocol/livekit"`, not in the SDK package
- **Fix:** Rewrote both files to import `lkproto` and use correct type references
- **Files modified:** backend/internal/work/livekit/room_manager.go, backend/internal/work/livekit/egress_manager.go
- **Verification:** `go build ./internal/server/...` and `go build ./cmd/work/...` both pass
- **Committed in:** 169f92e (Task 1 commit)

**2. [Rule 3 - Blocking] Removed duplicate parseTimestamp function**
- **Found during:** Task 2 (gateway routes)
- **Issue:** route_video.go defined its own parseTimestamp but the function already exists in route_work.go (same package)
- **Fix:** Removed duplicate definition, added comment noting shared function location
- **Files modified:** backend/internal/gateway/route_video.go
- **Verification:** `go build ./internal/gateway/...` passes
- **Committed in:** 648d7d9 (Task 2 commit)

**3. [Rule 2 - Missing Critical] Added VideoRoutes to gateway registrars list**
- **Found during:** Task 2 (gateway integration)
- **Issue:** Plan did not explicitly mention adding VideoRoutes to cmd/gateway/main.go registrars array
- **Fix:** Added `gateway.NewVideoRoutes(registry)` to the registrars slice in gateway main.go
- **Files modified:** backend/cmd/gateway/main.go
- **Verification:** `go build ./cmd/gateway/...` passes, routes registered on startup
- **Committed in:** 648d7d9 (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 bug, 1 blocking, 1 missing critical)
**Impact on plan:** All auto-fixes necessary for correct compilation and gateway wiring. No scope creep.

## Issues Encountered
- Context window reset during Task 1 execution required resumption from continuation summary
- LiveKit SDK type discovery required `go doc` investigation to find correct import paths

## User Setup Required
None - no external service configuration required. Docker Compose handles all LiveKit/Egress setup.

## Next Phase Readiness
- All video/meeting backend services are now accessible via HTTP and WebSocket
- Frontend can integrate with the 35 HTTP endpoints and 9 new WebSocket event types
- LiveKit + Egress Docker services ready for local development
- Plans 08-07 through 08-09 can build UI components against this API layer

## Self-Check: PASSED

---
*Phase: 08-video-voice-meetings*
*Completed: 2026-02-11*
