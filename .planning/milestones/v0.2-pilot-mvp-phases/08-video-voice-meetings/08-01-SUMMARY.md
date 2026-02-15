---
phase: 08-video-voice-meetings
plan: 01
subsystem: api, database
tags: [protobuf, grpc, livekit, video, meetings, presence, reactions, migrations, postgresql]

requires:
  - phase: 07-calendar-scheduling
    provides: "CalendarService proto, LiveKit service foundation, calendar tables"
  - phase: 03-chat-messaging
    provides: "ChatService proto, messages table (FK target for reactions)"
provides:
  - "VideoService gRPC definition with 31 RPCs across call, recording, meeting, notes, presence domains"
  - "ChatService reaction RPCs (ToggleReaction, ListReactions, GetReactionSummary)"
  - "9 database tables: call_sessions, call_participants, meetings, meeting_attendees, meeting_notes, meeting_action_items, recordings, recording_consents, message_reactions"
  - "presence_config table with default seed data"
  - "Go model packages: video, meeting, recording, presence, reaction"
  - "livekit/server-sdk-go/v2 dependency for room management and egress"
affects: [08-02, 08-03, 08-04, 08-05, 08-06, 08-07, 08-08, 08-09]

tech-stack:
  added: [livekit/server-sdk-go/v2 v2.13.3]
  patterns: [domain-scoped model packages under internal/work/, tools.go for SDK pre-registration]

key-files:
  created:
    - backend/proto/video/v1/video.proto
    - backend/proto/video/v1/video.pb.go
    - backend/proto/video/v1/video_grpc.pb.go
    - backend/migrations/000036_create_call_sessions.up.sql
    - backend/migrations/000036_create_call_sessions.down.sql
    - backend/migrations/000037_create_meetings.up.sql
    - backend/migrations/000037_create_meetings.down.sql
    - backend/migrations/000038_create_reactions_and_presence.up.sql
    - backend/migrations/000038_create_reactions_and_presence.down.sql
    - backend/internal/work/video/models.go
    - backend/internal/work/meeting/models.go
    - backend/internal/work/recording/models.go
    - backend/internal/work/presence/models.go
    - backend/internal/work/reaction/models.go
    - backend/tools.go
  modified:
    - backend/proto/chat/v1/chat.proto
    - backend/proto/chat/v1/chat.pb.go
    - backend/proto/chat/v1/chat_grpc.pb.go
    - backend/internal/config/config.go
    - backend/go.mod
    - backend/go.sum
    - backend/Makefile

key-decisions:
  - "tools.go with build constraint to keep server-sdk-go/v2 in go.mod before application code imports it"
  - "Domain-scoped model packages (internal/work/video/, meeting/, etc.) for Phase 8 models rather than central internal/models/"
  - "31 RPCs in VideoService covering 5 domains: calls (5), recording (6), meetings (7), notes/actions (8), presence (5)"
  - "Presence stored primarily in Redis (model struct only, no DB table for runtime state); presence_config in PostgreSQL"

patterns-established:
  - "VideoService proto: single service covering call, recording, meeting, notes, and presence RPCs"
  - "tools.go pattern: //go:build tools constraint for SDK pre-registration in go.mod"

duration: 6min
completed: 2026-02-11
---

# Phase 8 Plan 01: Proto + Migrations + Models Foundation Summary

**VideoService proto with 31 RPCs, ChatService reaction extension, 3 migrations (9 tables + presence_config), 5 Go model packages, and livekit/server-sdk-go/v2 dependency**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-11T11:44:57Z
- **Completed:** 2026-02-11T11:51:25Z
- **Tasks:** 2/2
- **Files modified:** 22

## Accomplishments

- VideoService proto defines 31 RPCs across call management, recording, meetings, notes/action items, and presence
- ChatService extended with 3 reaction RPCs (ToggleReaction, ListReactions, GetReactionSummary) plus Reaction/ReactionSummary messages
- 3 migrations create 9 domain tables (call_sessions, call_participants, meetings, meeting_attendees, meeting_notes, meeting_action_items, recordings, recording_consents, message_reactions) plus presence_config with seed
- 5 Go model packages with type-safe constants, validation maps, and JSON-tagged structs mirror all DB schemas
- livekit/server-sdk-go/v2 v2.13.3 installed for room management and Egress recording in subsequent plans

## Task Commits

Each task was committed atomically:

1. **Task 1: Proto definitions (video.proto + chat.proto reaction extension)** - `2a9cf9d` (feat)
2. **Task 2: Migrations + Go models + Go dependency** - `e4bf51c` (feat)

## Files Created/Modified

- `backend/proto/video/v1/video.proto` - VideoService with 31 RPCs, 6 enums, 15+ message types
- `backend/proto/video/v1/video.pb.go` - Generated protobuf Go code
- `backend/proto/video/v1/video_grpc.pb.go` - Generated gRPC server/client stubs
- `backend/proto/chat/v1/chat.proto` - Extended with 3 reaction RPCs + message types
- `backend/proto/chat/v1/chat.pb.go` - Regenerated with reaction messages
- `backend/proto/chat/v1/chat_grpc.pb.go` - Regenerated with reaction stubs
- `backend/migrations/000036_create_call_sessions.up.sql` - call_sessions + call_participants tables
- `backend/migrations/000036_create_call_sessions.down.sql` - Rollback for call tables
- `backend/migrations/000037_create_meetings.up.sql` - meetings, attendees, notes, action_items, recordings, consents
- `backend/migrations/000037_create_meetings.down.sql` - Rollback for meeting tables
- `backend/migrations/000038_create_reactions_and_presence.up.sql` - message_reactions + presence_config
- `backend/migrations/000038_create_reactions_and_presence.down.sql` - Rollback for reactions/presence
- `backend/internal/work/video/models.go` - CallSession, CallParticipant models with status constants
- `backend/internal/work/meeting/models.go` - Meeting, MeetingAttendee, MeetingNotes, MeetingActionItem, MeetingSummary
- `backend/internal/work/recording/models.go` - Recording, RecordingConsent, RecordingConsentStatus
- `backend/internal/work/presence/models.go` - UserPresence, PresenceConfig with level constants
- `backend/internal/work/reaction/models.go` - Reaction, ReactionSummary models
- `backend/internal/config/config.go` - Added LiveKitEgressTemplateURL env var
- `backend/go.mod` - Added livekit/server-sdk-go/v2 v2.13.3
- `backend/go.sum` - Updated dependency checksums
- `backend/Makefile` - Added video proto compilation target
- `backend/tools.go` - Build-constrained import to retain server-sdk-go/v2 in go.mod

## Decisions Made

- **tools.go for SDK retention:** Used `//go:build tools` constraint to keep server-sdk-go/v2 in go.mod before any application code imports it. This prevents `go mod tidy` from removing it while avoiding unused import errors.
- **Domain-scoped model packages:** Created models under `internal/work/{video,meeting,recording,presence,reaction}/` instead of central `internal/models/`. These are new Phase 8 domains that will grow into full service packages.
- **31 RPCs in single VideoService:** Kept all call, recording, meeting, notes, and presence RPCs in one service rather than splitting. This mirrors the work service consolidation pattern from the roadmap.
- **Presence: Redis-primary, config in PostgreSQL:** Runtime presence state (online/away/dnd) will be managed in Redis. Only the admin-configurable timeout is persisted in PostgreSQL via presence_config table.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] tools.go for server-sdk-go/v2 retention**
- **Found during:** Task 2 (Go dependency installation)
- **Issue:** `go mod tidy` removed server-sdk-go/v2 because no Go source files import it yet
- **Fix:** Created `backend/tools.go` with `//go:build tools` constraint importing the SDK
- **Files modified:** backend/tools.go
- **Verification:** `go mod tidy` retains the dependency, `go build ./...` succeeds
- **Committed in:** e4bf51c (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Standard Go pattern for pre-registering dependencies. No scope creep.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All proto definitions ready for gRPC server implementation (Plans 08-02 through 08-05)
- All database tables defined for repository layer implementation
- Go model types ready for service layer use
- livekit/server-sdk-go/v2 available for call management and recording services

## Self-Check: PASSED

---
*Phase: 08-video-voice-meetings*
*Completed: 2026-02-11*
