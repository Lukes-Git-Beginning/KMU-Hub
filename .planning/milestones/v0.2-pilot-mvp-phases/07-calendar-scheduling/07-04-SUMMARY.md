---
phase: 07-calendar-scheduling
plan: 04
subsystem: api
tags: [resource-booking, holidays, livekit, nager-date, double-booking-prevention, feature-flag]

requires:
  - phase: 07-01
    provides: "Calendar proto, migrations (resources, resource_bookings, resource_tags, public_holidays tables)"
  - phase: 06-01
    provides: "Work service patterns (repository/service/test structure)"

provides:
  - "Resource CRUD service with booking and DB-level double-booking prevention"
  - "Holiday service with Nager.Date API integration and lazy loading"
  - "LiveKit service with feature-flag pattern for video call token generation"
  - "Config fields for LiveKit environment variables"

affects:
  - 07-05 (gRPC server needs these services as dependencies)
  - 07-06 (gateway HTTP routes call these services)
  - 08-video-voice-meetings (LiveKit service reused for video calls)

tech-stack:
  added:
    - "github.com/livekit/protocol v1.44.0 (JWT token generation for video rooms)"
  patterns:
    - "Feature-flag service pattern (LiveKit enabled/disabled based on config)"
    - "Lazy loading with EnsureLoaded pattern (holiday data seeded on first access)"
    - "BookingConflictError with alternative suggestions on double-booking"

key-files:
  created:
    - "backend/internal/work/resource/service.go"
    - "backend/internal/work/resource/repository.go"
    - "backend/internal/work/resource/postgres_repository.go"
    - "backend/internal/work/resource/errors.go"
    - "backend/internal/work/resource/service_test.go"
    - "backend/internal/work/holiday/service.go"
    - "backend/internal/work/holiday/repository.go"
    - "backend/internal/work/holiday/postgres_repository.go"
    - "backend/internal/work/holiday/nager_client.go"
    - "backend/internal/work/holiday/errors.go"
    - "backend/internal/work/holiday/service_test.go"
    - "backend/internal/work/livekit/service.go"
    - "backend/internal/work/livekit/service_test.go"
  modified:
    - "backend/internal/config/config.go"
    - "backend/go.mod"
    - "backend/go.sum"

key-decisions:
  - "HolidayFetcher interface abstracts Nager client for testability"
  - "LiveKit disabled-by-default: empty config values = feature off"
  - "BookingConflictError carries alternative resource suggestions"
  - "Resource delete is soft-delete (is_active=false), bookings preserved"

patterns-established:
  - "Feature-flag service: NewService checks config, IsEnabled() guard, graceful degradation"
  - "Lazy loading: EnsureHolidaysLoaded checks HasDataForYear before seeding"
  - "Conflict error enrichment: catch DB constraint violation, enrich with alternatives"

duration: 8min
completed: 2026-02-10
---

# Phase 7 Plan 4: Resource, Holiday, and LiveKit Services Summary

**Resource booking with EXCLUDE GIST double-booking prevention and alternative suggestions, DACH holiday seeding via Nager.Date API with lazy loading, and LiveKit video call token generation with feature-flag pattern**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-10T22:23:41Z
- **Completed:** 2026-02-10T22:31:00Z
- **Tasks:** 2
- **Files modified:** 16

## Accomplishments
- Resource service with full CRUD, tag management, booking with DB-enforced conflict detection, and alternative resource suggestions on double-booking
- Holiday service with Nager.Date API integration, lazy loading pattern, DACH country support, subdivision filtering, and year boundary handling
- LiveKit service with feature-flag pattern generating JWT join tokens with VideoGrant when configured, graceful no-op when unconfigured
- 80 total tests across 3 packages (38 resource + 30 holiday + 12 livekit), all passing

## Task Commits

Each task was committed atomically:

1. **Task 1: Resource service package** - `e064ddc` (feat)
2. **Task 2: Holiday service + Nager client + LiveKit service** - `4631e88` (feat)

## Files Created/Modified
- `backend/internal/work/resource/errors.go` - Sentinel errors + BookingConflictError with alternatives
- `backend/internal/work/resource/repository.go` - Repository interface with CRUD, booking, and availability methods
- `backend/internal/work/resource/postgres_repository.go` - PostgreSQL implementation with EXCLUDE constraint handling
- `backend/internal/work/resource/service.go` - Business logic: Create/List/Update/Delete, Book with conflict detection, CancelBooking
- `backend/internal/work/resource/service_test.go` - 38 tests with mock repository
- `backend/internal/work/holiday/errors.go` - Sentinel errors for holiday operations
- `backend/internal/work/holiday/repository.go` - Repository interface with BulkUpsert, ListByDateRange, HasDataForYear
- `backend/internal/work/holiday/postgres_repository.go` - PostgreSQL implementation with ON CONFLICT upsert
- `backend/internal/work/holiday/nager_client.go` - HTTP client for Nager.Date API + MapToModels conversion
- `backend/internal/work/holiday/service.go` - Seed/EnsureLoaded/ListHolidays with lazy loading and year boundaries
- `backend/internal/work/holiday/service_test.go` - 30 tests covering seeding, caching, filtering, boundaries
- `backend/internal/work/livekit/service.go` - Feature-flagged LiveKit room naming and JWT token generation
- `backend/internal/work/livekit/service_test.go` - 12 tests including JWT structure verification
- `backend/internal/config/config.go` - Added LiveKitAPIKey, LiveKitAPISecret, LiveKitWSURL fields
- `backend/go.mod` - Added livekit/protocol v1.44.0 dependency
- `backend/go.sum` - Updated dependency checksums

## Decisions Made
- HolidayFetcher interface abstracts the Nager.Date HTTP client for testability (mock in unit tests, real client in production)
- LiveKit service uses disabled-by-default pattern: empty config values mean feature is off, no errors on startup
- BookingConflictError is a rich error type carrying alternative resource suggestions (up to 5 alternatives of same type)
- Resource deletion is soft-delete (is_active = false) to preserve existing booking history
- Holiday ListHolidays handles year boundaries automatically (Dec-Jan queries load both years)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed TestListAvailability midnight boundary issue**
- **Found during:** Task 1 (Resource service tests)
- **Issue:** Test booked resource at time.Now()+1h but queried availability for time.Now() date, which failed near midnight when booking crossed into next day
- **Fix:** Used fixed date (2026-06-15 12:00 UTC) instead of time.Now() for deterministic test behavior
- **Files modified:** backend/internal/work/resource/service_test.go
- **Verification:** Test passes consistently
- **Committed in:** e064ddc (Task 1 commit)

**2. [Rule 3 - Blocking] Resolved missing go-jose dependency for LiveKit**
- **Found during:** Task 2 (LiveKit service build)
- **Issue:** go.sum missing entry for github.com/go-jose/go-jose/v3 (transitive dep of livekit/protocol/auth)
- **Fix:** Ran `go get github.com/livekit/protocol/auth@v1.44.0` and `go mod tidy`
- **Files modified:** backend/go.mod, backend/go.sum
- **Verification:** Build succeeds, all tests pass
- **Committed in:** 4631e88 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes necessary for test reliability and build success. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required. LiveKit config is optional (feature-flagged).

## Next Phase Readiness
- All 3 service packages ready for gRPC server integration (plan 07-05)
- Repository interfaces defined for PostgreSQL implementations
- LiveKit dependency added and tested, ready for video call features in Phase 8

## Self-Check: PASSED

---
*Phase: 07-calendar-scheduling*
*Completed: 2026-02-10*
