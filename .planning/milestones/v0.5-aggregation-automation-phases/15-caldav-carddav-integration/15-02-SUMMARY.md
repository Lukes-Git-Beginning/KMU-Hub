---
phase: 15-caldav-carddav-integration
plan: 02
subsystem: api
tags: [caldav, carddav, go-webdav, go-ical, go-vcard, icalendar, vcard, vtimezone, etag, rfc5545, rfc6350]

# Dependency graph
requires:
  - phase: 15-caldav-carddav-integration
    provides: App-specific passwords, sync token service, CalDAV models, go-webdav/go-ical deps
  - phase: 07-calendar-scheduling
    provides: Calendar events, attendees, exceptions, CalendarService gRPC
  - phase: 02-crm-core
    provides: Contact models, CRMService gRPC for CardDAV sync
provides:
  - CalDAVBackend implementing caldav.Backend with gRPC calendar service integration
  - CardDAVBackend implementing carddav.Backend with gRPC CRM service integration
  - Bidirectional iCalendar converter (EventToICal, ICalToEventInput) with RRULE, EXDATE, RECURRENCE-ID, ATTENDEE, VTIMEZONE
  - Bidirectional vCard 4.0 converter (ContactToVCard, VCardToContactInput) with N, FN, EMAIL, TEL, ORG, TITLE, NOTE
  - VTIMEZONE generator with DACH CET/CEST hardcoded definition and minimal fallback
  - Deterministic ETag from SHA-256 of UUID + timestamp, CTag from sync version
  - Context key pattern (UserFromCtx/CtxWithUser) for CalDAV Basic Auth user propagation
  - gRPC-to-WebDAV error mapping utility
affects: [15-caldav-carddav-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [caldav.Backend adapter pattern, carddav.Backend adapter pattern, iCalendar bidirectional conversion, vCard bidirectional conversion, VTIMEZONE caching with sync.Map, deterministic ETag generation, compile-time interface compliance checks]

key-files:
  created:
    - backend/internal/caldav/etag.go
    - backend/internal/caldav/vtimezone.go
    - backend/internal/caldav/ical_converter.go
    - backend/internal/caldav/vcard_converter.go
    - backend/internal/caldav/caldav_backend.go
    - backend/internal/caldav/carddav_backend.go
  modified: []

key-decisions:
  - "CalDAVBackend queries event_exceptions directly from DB (no dedicated gRPC RPC for exceptions)"
  - "CardDAV uses two fixed address books per user: personal (Meine Kontakte) and company (Firmenkontakte)"
  - "VTIMEZONE cache uses sync.Map for thread-safe concurrent access"
  - "DACH CET/CEST timezone hardcoded for Europe/Berlin, Europe/Zurich, Europe/Vienna"
  - "Compile-time interface compliance checks via var _ caldav.Backend = (*CalDAVBackend)(nil)"
  - "Sync collection ID for address books generated deterministically via uuid.NewSHA1(userID, bookType)"

patterns-established:
  - "CalDAV/CardDAV Backend adapter: go-webdav interface backed by gRPC service via ServiceRegistry"
  - "UserFromCtx/CtxWithUser context key pattern for CalDAV authentication propagation"
  - "grpcToWebDAVError: centralized gRPC status code to HTTP status code mapping"
  - "Path parsing helpers: calendarIDFromPath, eventUIDFromPath, contactIDFromPath, addressBookTypeFromPath"
  - "Intermediate input types (CalEventInput, CalExceptionInput, ContactInput) decouple iCal/vCard parsing from gRPC request creation"

requirements-completed: [INT-01, INT-02]

# Metrics
duration: 7min
completed: 2026-02-20
---

# Phase 15 Plan 02: CalDAV/CardDAV Backend Adapters Summary

**CalDAV and CardDAV backend adapters implementing go-webdav interfaces backed by gRPC calendar/CRM services, with bidirectional iCalendar/vCard converters, VTIMEZONE generator, and ETag utility**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-20T11:06:51Z
- **Completed:** 2026-02-20T11:14:15Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Bidirectional iCalendar converter handling master events, recurring events with RRULE, cancelled exceptions (EXDATE), modified exceptions (RECURRENCE-ID), attendees with RSVP mapping, and VTIMEZONE components
- Bidirectional vCard 4.0 converter for name (N/FN), email, phone, org, title, notes with UID and REV fields
- CalDAVBackend fully implementing caldav.Backend: ListCalendars, GetCalendar, GetCalendarObject, ListCalendarObjects, QueryCalendarObjects, PutCalendarObject, DeleteCalendarObject with role-based write permissions
- CardDAVBackend fully implementing carddav.Backend: two address books (personal/company), CRUD contacts with visibility-based separation, admin/manager ACL for company contacts
- VTIMEZONE generator with hardcoded DACH CET/CEST transitions and minimal fallback for other timezones, cached via sync.Map
- Deterministic ETag generation from SHA-256(UUID + updatedAt.UnixNano) and CTag from sync version

## Task Commits

Each task was committed atomically:

1. **Task 1: iCal/vCard converters + VTIMEZONE + ETag utilities** - `e3fdae0` (feat)
2. **Task 2: CalDAV Backend + CardDAV Backend adapters** - `4997b7d` (feat)

## Files Created/Modified
- `backend/internal/caldav/etag.go` - Deterministic ETag from UUID+timestamp, CTag from sync version
- `backend/internal/caldav/vtimezone.go` - VTIMEZONE component generation with DACH CET/CEST and minimal fallback
- `backend/internal/caldav/ical_converter.go` - Bidirectional iCalendar conversion with RRULE, EXDATE, RECURRENCE-ID, ATTENDEE
- `backend/internal/caldav/vcard_converter.go` - Bidirectional vCard 4.0 conversion with N, FN, EMAIL, TEL, ORG, TITLE, NOTE
- `backend/internal/caldav/caldav_backend.go` - CalDAVBackend implementing caldav.Backend via calendar gRPC service
- `backend/internal/caldav/carddav_backend.go` - CardDAVBackend implementing carddav.Backend via CRM gRPC service

## Decisions Made
- CalDAVBackend queries event_exceptions table directly from DB since there is no dedicated gRPC RPC for listing event exceptions -- this avoids adding a new proto RPC just for CalDAV
- CardDAV uses two fixed address books per user (personal=personal contacts, company=shared contacts) mapping to CRM visibility model
- VTIMEZONE cache uses sync.Map for lock-free concurrent reads (most access is reads after initial generation)
- DACH timezone hardcoded for Europe/Berlin, Europe/Zurich, Europe/Vienna with full STANDARD/DAYLIGHT subcomponents; other timezones get minimal VTIMEZONE with current offset only
- Compile-time interface compliance via `var _ caldav.Backend = (*CalDAVBackend)(nil)` pattern
- Address book sync collection IDs generated deterministically via uuid.NewSHA1(userID, bookType) to ensure consistent sync token tracking

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed ical.Component children append pattern**
- **Found during:** Task 1 (VTIMEZONE generator)
- **Issue:** `ical.NewComponent()` returns `*ical.Component` directly, not a wrapper struct -- accessing `.Component` field was invalid
- **Fix:** Changed `vtimezone.Children = append(..., standard.Component)` to `vtimezone.Children = append(..., standard)` for all child appends and return values
- **Files modified:** backend/internal/caldav/vtimezone.go
- **Verification:** go build ./internal/caldav/ compiles successfully
- **Committed in:** e3fdae0 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed proto enum constant name for RecurringEventScope**
- **Found during:** Task 2 (CalDAV PutCalendarObject)
- **Issue:** Used `calendarv1.THIS_EVENT` but correct Go constant is `calendarv1.RecurringEventScope_THIS_EVENT`
- **Fix:** Changed all occurrences to use full qualified enum name
- **Files modified:** backend/internal/caldav/caldav_backend.go
- **Verification:** go build ./internal/caldav/ compiles successfully
- **Committed in:** 4997b7d (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both were minor naming/API issues caught at compile time. No scope creep.

## Issues Encountered
None beyond the auto-fixed compilation issues above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CalDAV and CardDAV backend adapters complete, ready for Plan 15-03 (HTTP handler registration)
- UserFromCtx/CtxWithUser context key pattern ready for Basic Auth middleware in Plan 15-03
- grpcToWebDAVError utility available for error handling in Plan 15-03 routes
- All 6 new Go files compile successfully with interface compliance verified

## Self-Check: PASSED

- All 6 created files verified on disk
- Commit e3fdae0 (Task 1) verified in git log
- Commit 4997b7d (Task 2) verified in git log
- Go build ./internal/caldav/ compiles
- CalDAVBackend satisfies caldav.Backend interface (compile-time check)
- CardDAVBackend satisfies carddav.Backend interface (compile-time check)
- No fmt.Println in any new file

---
*Phase: 15-caldav-carddav-integration*
*Completed: 2026-02-20*
