---
phase: 07-calendar-scheduling
plan: 01
subsystem: database, api
tags: [grpc, protobuf, postgresql, migrations, rrule, livekit, calendar, scheduling, resources]

# Dependency graph
requires:
  - phase: 06-project-management
    provides: Work service pattern, models pattern, migration numbering (up to 000031)
  - phase: 04-notifications-gateway
    provides: event_types table for notification seeding, permissions/role_permissions for RBAC
provides:
  - CalendarService gRPC proto with 40 RPCs
  - Database schema for calendars, events, resources, holidays (12 tables)
  - Go model types for all calendar entities
  - rrule-go and LiveKit SDK dependencies in go.mod
affects: [07-02, 07-03, 07-04, 07-05, 07-06, 07-07, 07-08, 07-09, 14-caldav]

# Tech tracking
tech-stack:
  added: [teambition/rrule-go v1.8.2, livekit/server-sdk-go/v2, livekit/protocol]
  patterns: [separate proto file per domain (calendar.proto alongside work.proto), deferred FK constraints across migrations, EXCLUDE USING GIST for double-booking prevention]

key-files:
  created:
    - backend/proto/calendar/v1/calendar.proto
    - backend/proto/calendar/v1/calendar.pb.go
    - backend/proto/calendar/v1/calendar_grpc.pb.go
    - backend/migrations/000032_create_calendars.up.sql
    - backend/migrations/000033_create_events.up.sql
    - backend/migrations/000034_create_resources.up.sql
    - backend/migrations/000035_create_holidays.up.sql
    - backend/internal/models/calendar.go
    - backend/internal/models/calendar_event.go
    - backend/internal/models/resource.go
    - backend/internal/models/holiday.go
    - backend/internal/models/event_category.go
  modified:
    - backend/go.mod
    - backend/go.sum
    - backend/Makefile

key-decisions:
  - "Separate calendar.proto file rather than extending work.proto (cleaner separation, same binary registers both services)"
  - "Deferred FK constraints: resource_id in calendar_events/event_exceptions added via ALTER TABLE in migration 000034 after resources table exists"
  - "40 RPCs covering calendars, events, resources, bookings, holidays, preferences, task deadlines, and LiveKit token generation"
  - "EXCLUDE USING GIST constraint for database-level double-booking prevention on resource_bookings"

patterns-established:
  - "Separate proto file per domain: calendar/v1/calendar.proto alongside work/v1/work.proto"
  - "Deferred FK constraints across sequential migrations via ALTER TABLE ADD CONSTRAINT"
  - "Calendar-prefixed model naming to avoid collision with notification Event model"

# Metrics
duration: 7min
completed: 2026-02-10
---

# Phase 7 Plan 01: Calendar Data Foundation Summary

**CalendarService gRPC proto with 40 RPCs, 12-table database schema with RRULE/RSVP/resource booking support, Go models, and rrule-go + LiveKit SDK dependencies**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-10T22:12:32Z
- **Completed:** 2026-02-10T22:19:17Z
- **Tasks:** 3
- **Files modified:** 21 (6 proto/go.mod/Makefile + 8 migrations + 5 models + go.sum + generated pb.go)

## Accomplishments
- CalendarService gRPC definition with 40 RPCs covering the full calendar domain (calendars, events, resources, bookings, holidays, preferences, LiveKit)
- 4 database migrations creating 12 tables with proper FK chains, indexes, EXCLUDE USING GIST for double-booking, GIN index for holiday subdivisions, and seeded notification event types + RBAC permissions
- 5 Go model files with all structs, constants, and validation maps matching the database schema
- New Go dependencies: rrule-go for RFC 5545 RRULE expansion, LiveKit SDK for video call token generation

## Task Commits

Each task was committed atomically:

1. **Task 1: Proto definition + new Go dependencies** - `1b3bf85` (feat)
2. **Task 2: Database migrations 000032-000035** - `7abdabb` (feat)
3. **Task 3: Go models** - `722594a` (feat)

## Files Created/Modified
- `backend/proto/calendar/v1/calendar.proto` - CalendarService gRPC definition with 40 RPCs
- `backend/proto/calendar/v1/calendar.pb.go` - Generated protobuf Go code
- `backend/proto/calendar/v1/calendar_grpc.pb.go` - Generated gRPC Go server/client code
- `backend/migrations/000032_create_calendars.up.sql` - calendars, calendar_members, event_categories tables
- `backend/migrations/000032_create_calendars.down.sql` - Reverse migration
- `backend/migrations/000033_create_events.up.sql` - calendar_events, event_attendees, event_exceptions, event_reminders, user_calendar_preferences tables
- `backend/migrations/000033_create_events.down.sql` - Reverse migration
- `backend/migrations/000034_create_resources.up.sql` - resources, resource_tags, resource_bookings with EXCLUDE USING GIST + deferred FK
- `backend/migrations/000034_create_resources.down.sql` - Reverse migration
- `backend/migrations/000035_create_holidays.up.sql` - public_holidays + seeded event types + RBAC permissions
- `backend/migrations/000035_create_holidays.down.sql` - Reverse migration
- `backend/internal/models/calendar.go` - Calendar, CalendarWithMemberInfo, CalendarMember structs
- `backend/internal/models/calendar_event.go` - CalendarEvent, ExpandedEvent, EventAttendee, EventException, EventReminder, UserCalendarPreferences, TaskDeadlineStub
- `backend/internal/models/resource.go` - Resource, ResourceBooking structs
- `backend/internal/models/holiday.go` - PublicHoliday struct
- `backend/internal/models/event_category.go` - EventCategory struct
- `backend/go.mod` - Added rrule-go, livekit/server-sdk-go/v2, livekit/protocol
- `backend/go.sum` - Updated checksums
- `backend/Makefile` - Added calendar proto generation target

## Decisions Made
- **Separate proto file**: Created `calendar/v1/calendar.proto` instead of extending `work.proto` (already 88 RPCs). Same Work service binary will register both CalendarService and WorkService. Follows research recommendation.
- **Deferred FK constraints**: `resource_id` columns in `calendar_events` and `event_exceptions` added WITHOUT FK in migration 000033, then FK constraints added via ALTER TABLE in 000034 after `resources` table exists. Clean table definitions without forward-reference issues.
- **Model naming**: Used `CalendarEvent` (not `Event`) and `EventCategory` (not `Category`) to avoid collision with existing `Event` and `EventType` types in notification models.
- **40 RPCs**: Slightly above the planned ~39 due to adding `UpdateCalendarMemberPermission` as a separate RPC from member CRUD (cleaner API for permission-only changes).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added event_exceptions.resource_id FK to migration 000034**
- **Found during:** Task 2 (migration creation)
- **Issue:** Plan specified deferred FK for calendar_events.resource_id but event_exceptions also has a resource_id column that needs the same deferred FK treatment
- **Fix:** Added `ALTER TABLE event_exceptions ADD CONSTRAINT fk_event_exceptions_resource` in 000034
- **Files modified:** backend/migrations/000034_create_resources.up.sql, backend/migrations/000034_create_resources.down.sql
- **Verification:** Down migration drops both constraints before dropping resources table
- **Committed in:** 7abdabb (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Essential for referential integrity. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Proto, migrations, and models form a complete data layer for calendar domain
- Plan 07-02 (Calendar Repository + Service) can build directly on these types
- Plan 07-04 (Resource Booking) will use the EXCLUDE USING GIST constraint
- Plan 07-08 (Holiday Data) will use the public_holidays table and seeded event types
- LiveKit SDK available for Plan 07-06 (Video Call Events)

## Self-Check: PASSED

---
*Phase: 07-calendar-scheduling*
*Completed: 2026-02-10*
