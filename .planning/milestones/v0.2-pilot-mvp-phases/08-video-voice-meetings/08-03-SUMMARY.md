---
phase: 08-video-voice-meetings
plan: 03
subsystem: api
tags: [livekit, video, recording, egress, dsgvo, consent, go, pgx]

# Dependency graph
requires:
  - phase: 08-01
    provides: Proto definitions, migrations (call_sessions, recordings, recording_consents), domain models
provides:
  - Video call service (CreateCall, JoinCall, EndCall, webhook handlers)
  - Recording service (StartRecording, StopRecording, DSGVO consent, 30-day retention)
  - RoomManager interface for LiveKit room/token abstraction
  - EgressManager interface for LiveKit Egress abstraction
  - Phase 11 integration point (ListRecordingsWithAccess, GetRecordingParticipants)
affects: [08-04, 08-05, 08-06, 11-documents-files]

# Tech tracking
tech-stack:
  added: []
  patterns: [RoomManager interface pattern, EgressManager interface pattern, DSGVO consent state machine, nil-manager disabled mode]

key-files:
  created:
    - backend/internal/work/video/repository.go
    - backend/internal/work/video/postgres_repository.go
    - backend/internal/work/video/service.go
    - backend/internal/work/video/service_test.go
    - backend/internal/work/video/errors.go
    - backend/internal/work/recording/repository.go
    - backend/internal/work/recording/postgres_repository.go
    - backend/internal/work/recording/service.go
    - backend/internal/work/recording/service_test.go
    - backend/internal/work/recording/errors.go
  modified: []

key-decisions:
  - "RoomManager/EgressManager interfaces with nil=disabled pattern for graceful LiveKit-off mode"
  - "DSGVO consent is checked at StartRecording time: all participants must have responded before Egress begins"
  - "30-day retention set on every recording via RetentionExpiresAt field"
  - "Phase 11 integration via ListRecordingsWithAccess (participant-only access via JOIN on call_participants/meeting_attendees)"
  - "Call auto-ends when last participant leaves (HandleParticipantLeft webhook handler)"

patterns-established:
  - "Nil-manager disabled mode: pass nil for RoomManager/EgressManager to disable LiveKit features gracefully"
  - "DSGVO consent state machine: consent must be collected before recording starts, not after"
  - "Repository interface per domain: video.Repository and recording.Repository separate from each other"

# Metrics
duration: 5min
completed: 2026-02-11
---

# Phase 8 Plan 3: Call + Recording Services Summary

**Video call service with room/token management and recording service with DSGVO consent state machine, 30-day retention, and Phase 11 file manager integration**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-11T11:56:19Z
- **Completed:** 2026-02-11T12:01:33Z
- **Tasks:** 2
- **Files created:** 10

## Accomplishments
- Video call service: full lifecycle (ringing -> active -> ended) with CreateCall, JoinCall, EndCall
- LiveKit webhook handlers for participant join/leave and room finished events
- Recording service with DSGVO-compliant consent gating (blocks Egress until all participants respond)
- 30-day retention automatically set on every recording, with cleanup job for expired recordings
- Phase 11 integration point: ListRecordingsWithAccess enforces participant-only access via JOINs
- Both services handle LiveKit-disabled mode gracefully (return typed errors, never crash)
- 41 unit tests total (20 video + 21 recording), all passing

## Task Commits

Each task was committed atomically:

1. **Task 1: Video call service** - `17ae272` (feat)
2. **Task 2: Recording service** - `a3a1051` (feat)

## Files Created/Modified
- `backend/internal/work/video/errors.go` - Typed error sentinels (ErrNotFound, ErrCallEnded, ErrMaxParticipants, ErrLiveKitNotConfigured)
- `backend/internal/work/video/repository.go` - Repository interface for call session persistence (10 methods)
- `backend/internal/work/video/postgres_repository.go` - PostgreSQL implementation with pgx pool
- `backend/internal/work/video/service.go` - Call lifecycle management (CreateCall, JoinCall, EndCall, webhook handlers)
- `backend/internal/work/video/service_test.go` - 20 unit tests with mock RoomManager and mock Repository
- `backend/internal/work/recording/errors.go` - Typed error sentinels (ErrConsentPending, ErrEgressNotConfigured, ErrRecordingNotActive)
- `backend/internal/work/recording/repository.go` - Repository interface for recordings + consents (13 methods)
- `backend/internal/work/recording/postgres_repository.go` - PostgreSQL implementation with consent upsert and access control queries
- `backend/internal/work/recording/service.go` - Recording lifecycle + DSGVO consent + retention + Phase 11 integration
- `backend/internal/work/recording/service_test.go` - 21 unit tests with mock EgressManager and mock Repository

## Decisions Made
- **RoomManager/EgressManager nil=disabled pattern:** If the manager is nil, the service sets enabled=false and returns typed errors for operations requiring LiveKit. This avoids crashes when LiveKit is not configured.
- **DSGVO consent checked at StartRecording:** The consent check happens when recording is initiated, not when egress starts. If any participant has not responded, ErrConsentPending is returned and egress never begins.
- **30-day retention via RetentionExpiresAt:** Set automatically at recording creation time. CleanupExpiredRecordings job scans for completed recordings past their expiry.
- **Phase 11 file manager integration:** ListRecordingsWithAccess uses JOIN-based queries on call_participants and meeting_attendees tables to enforce participant-only access. GetRecordingParticipants returns the ACL user list.
- **Auto-end on empty room:** HandleParticipantLeft checks active participant count; if zero, the call is automatically ended with duration calculated.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Video and recording services ready for gRPC server integration (Plan 08-05 or 08-06)
- RoomManager interface ready to be implemented by existing livekit.Service (adapter needed in gRPC server)
- EgressManager interface ready for LiveKit Egress SDK integration
- Meeting service (Plan 08-04) can reference recording service for meeting recordings

## Self-Check: PASSED

---
*Phase: 08-video-voice-meetings*
*Completed: 2026-02-11*
