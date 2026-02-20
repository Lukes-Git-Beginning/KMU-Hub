---
phase: 14-event-infrastructure-unified-inbox
plan: 01
subsystem: api, database, infra
tags: [grpc, protobuf, postgresql, pg_notify, event-driven, inbox]

# Dependency graph
requires:
  - phase: 04-notification-gateway
    provides: EventBus, EmitEvent, pg_notify infrastructure, event_types table
  - phase: 10-email-integration
    provides: Email message service
  - phase: 11-documents-files
    provides: Document file service
  - phase: 12-rechnungen-finanzen
    provides: Invoice and quote services
  - phase: 13-hr-zeiterfassung
    provides: Leave and timetracking services
provides:
  - InboxService proto with 27 RPCs (messages, team inboxes, routing rules)
  - Database schema for inbox_messages, team_inboxes, team_inbox_members, routing_rules
  - 20 new event type constants across 5 modules (email, document, biz, hr, inbox)
  - 5 new module ID constants
  - Go domain models for all inbox entities
  - PGEventEmitter implementations in 6 service packages
  - Event emit calls wired into email, document, finance, and HR services
affects: [14-02, 14-03, 14-04, 16-automation-engine]

# Tech tracking
tech-stack:
  added: []
  patterns: [PGEventEmitter retrofit pattern for existing services, Condition/Action JSON tree for routing rules]

key-files:
  created:
    - backend/proto/inbox/v1/inbox.proto
    - backend/proto/inbox/v1/inbox.pb.go
    - backend/proto/inbox/v1/inbox_grpc.pb.go
    - backend/migrations/000047_create_inbox_tables.up.sql
    - backend/migrations/000047_create_inbox_tables.down.sql
    - backend/migrations/000048_seed_inbox_event_types.up.sql
    - backend/migrations/000048_seed_inbox_event_types.down.sql
    - backend/internal/models/inbox.go
    - backend/internal/email/message/event_emitter.go
    - backend/internal/document/file/event_emitter.go
    - backend/internal/biz/invoice/event_emitter.go
    - backend/internal/biz/quote/event_emitter.go
    - backend/internal/biz/hr/leave/event_emitter.go
    - backend/internal/biz/hr/timetracking/event_emitter.go
  modified:
    - backend/internal/notification/event/types.go
    - backend/internal/email/message/service.go
    - backend/internal/document/file/service.go
    - backend/internal/biz/invoice/service.go
    - backend/internal/biz/quote/service.go
    - backend/internal/biz/hr/leave/service.go
    - backend/internal/biz/hr/timetracking/service.go
    - backend/Makefile

key-decisions:
  - "HR timetracking event emitter placed in biz/hr/timetracking/ (actual package) not biz/hr/timeentry/ (plan path was wrong)"
  - "Document shared event emitted on LinkToEntity (entity linking is closest to sharing semantics in current service)"

patterns-established:
  - "PGEventEmitter retrofit: create event_emitter.go, add emitter field + SetEventEmitter + emitEvent helper to existing service, wire emit calls after successful operations"
  - "Condition/Action JSON tree model for routing rules (designed for Phase 16 Automation reuse)"

requirements-completed: [EVENT-01, INBOX-01]

# Metrics
duration: 9min
completed: 2026-02-20
---

# Phase 14 Plan 01: Event Infrastructure + Inbox Data Foundation Summary

**InboxService proto with 27 RPCs, 4 inbox database tables with optimized partial indexes, 20 new event type constants, and PGEventEmitter retrofit across 6 service packages (email, document, invoice, quote, leave, timetracking)**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-19T23:18:39Z
- **Completed:** 2026-02-19T23:27:52Z
- **Tasks:** 2
- **Files modified:** 22

## Accomplishments
- InboxService proto defines 27 RPCs covering messages (14), team inboxes (8), and routing rules (5) with full request/response types
- Migration 000047 creates inbox_messages, team_inboxes, team_inbox_members, routing_rules tables with 6 partial indexes for optimized queries
- Migration 000048 seeds 20 event types across email, document, biz, hr, and inbox modules with German display names
- All 4 previously non-emitting services (Email, Document, Finance, HR) now have PGEventEmitter implementations with SetEventEmitter + emit calls at CRUD operation points
- Extended event type constants cover all new modules with 5 new module ID constants

## Task Commits

Each task was committed atomically:

1. **Task 1: Proto + migrations + models + event type constants** - `05e090c` (feat)
2. **Task 2: Retrofit PGEventEmitter in Email, Document, Biz, and HR services** - `ac19fb5` (feat)

## Files Created/Modified
- `backend/proto/inbox/v1/inbox.proto` - InboxService gRPC contract with 27 RPCs
- `backend/proto/inbox/v1/inbox.pb.go` - Generated protobuf Go code
- `backend/proto/inbox/v1/inbox_grpc.pb.go` - Generated gRPC Go code
- `backend/migrations/000047_create_inbox_tables.up.sql` - Inbox database schema (4 tables, 7 indexes)
- `backend/migrations/000047_create_inbox_tables.down.sql` - Drop inbox tables
- `backend/migrations/000048_seed_inbox_event_types.up.sql` - Seed 20 event types for 5 modules
- `backend/migrations/000048_seed_inbox_event_types.down.sql` - Delete seeded event types
- `backend/internal/models/inbox.go` - Go models for InboxMessage, TeamInbox, TeamInboxMember, RoutingRule, Condition, Action, UnreadCount
- `backend/internal/notification/event/types.go` - Extended with 20 new event type constants and 5 module IDs
- `backend/internal/email/message/event_emitter.go` - Email PGEventEmitter
- `backend/internal/email/message/service.go` - Added emitter field, SetEventEmitter, emit on create/delete
- `backend/internal/document/file/event_emitter.go` - Document PGEventEmitter
- `backend/internal/document/file/service.go` - Added emitter field, SetEventEmitter, emit on upload/version/share
- `backend/internal/biz/invoice/event_emitter.go` - Invoice PGEventEmitter
- `backend/internal/biz/invoice/service.go` - Added emitter field, SetEventEmitter, emit on create/send
- `backend/internal/biz/quote/event_emitter.go` - Quote PGEventEmitter
- `backend/internal/biz/quote/service.go` - Added emitter field, SetEventEmitter, emit on create
- `backend/internal/biz/hr/leave/event_emitter.go` - Leave PGEventEmitter
- `backend/internal/biz/hr/leave/service.go` - Added emitter field, SetEventEmitter, emit on request/approve/reject
- `backend/internal/biz/hr/timetracking/event_emitter.go` - Timetracking PGEventEmitter
- `backend/internal/biz/hr/timetracking/service.go` - Added emitter field, SetEventEmitter, emit on clock-in/clock-out
- `backend/Makefile` - Added inbox proto generation target

## Decisions Made
- HR timetracking event emitter placed in `biz/hr/timetracking/` (actual package name) instead of plan's `biz/hr/timeentry/` path which does not exist
- Document "shared" event emitted on `LinkToEntity` since that is the closest operation to sharing in the current document file service

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Corrected HR timeentry package path to timetracking**
- **Found during:** Task 2 (HR event emitter creation)
- **Issue:** Plan specified `backend/internal/biz/hr/timeentry/event_emitter.go` but the actual package is `timetracking`
- **Fix:** Created event_emitter.go in `biz/hr/timetracking/` instead
- **Files modified:** backend/internal/biz/hr/timetracking/event_emitter.go, backend/internal/biz/hr/timetracking/service.go
- **Verification:** `go build ./internal/biz/...` passes
- **Committed in:** ac19fb5 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking path correction)
**Impact on plan:** Trivial path correction. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- InboxService proto ready for gRPC server and gateway route implementation (Plan 02)
- All inbox database tables defined; ready for repository + service layer (Plan 02)
- Event emitters wired; inbox consumer can register handlers on the EventBus to populate inbox_messages
- Routing rule Condition/Action models ready for evaluator implementation (Plan 02)

## Self-Check: PASSED

All 16 created/modified files verified present. Both task commits (05e090c, ac19fb5) confirmed in git log. `go build ./...` passes with zero errors.

---
*Phase: 14-event-infrastructure-unified-inbox*
*Completed: 2026-02-20*
