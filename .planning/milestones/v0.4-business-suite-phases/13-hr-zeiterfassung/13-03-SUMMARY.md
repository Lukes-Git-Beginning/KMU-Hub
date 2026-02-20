---
phase: 13-hr-zeiterfassung
plan: 03
subsystem: api
tags: [go, grpc, arbzg, timetracking, hr, compliance, gateway, openapi]

# Dependency graph
requires:
  - phase: 13-01
    provides: "HR proto with 29 RPCs, 9-table migration, ArbZG/BUrlG compliance engine"
  - phase: 13-02
    provides: "Leave service, absence service, employee service with repositories"
provides:
  - "Time tracking service with ArbZG enforcement (clock in/out, breaks, corrections)"
  - "HRGRPCServer implementing all 29 RPCs from hr.proto"
  - "Gateway HTTP routes under /api/v1/hr/* for all HR domains"
  - "Biz binary extended to host both FinanceService and HRService"
  - "OpenAPI spec with HR endpoint schemas and domain types"
affects: [14-unified-inbox, frontend-hr-module]

# Tech tracking
tech-stack:
  added: []
  patterns: ["Multi-service gRPC binary (finance + HR on same port)", "Gateway route composition for missing RPCs (GetWorkTimeStatus)"]

key-files:
  created:
    - backend/internal/biz/hr/timetracking/errors.go
    - backend/internal/biz/hr/timetracking/repository.go
    - backend/internal/biz/hr/timetracking/postgres_repository.go
    - backend/internal/biz/hr/timetracking/service.go
    - backend/internal/biz/hr/timetracking/service_test.go
    - backend/internal/server/hr_grpc.go
    - backend/internal/gateway/route_hr.go
  modified:
    - backend/cmd/biz/main.go
    - backend/cmd/gateway/main.go
    - backend/api/openapi.yaml

key-decisions:
  - "HR services registered on same gRPC server as finance (biz binary), sharing port :50058"
  - "GetWorkTimeStatus composed in gateway from GetActiveShift + GetDailySummary RPCs (no dedicated proto RPC)"
  - "ArbZG severity at exactly 600 minutes (10h) returns warning, not error (CheckWorkTime uses > 600 for error)"

patterns-established:
  - "Multi-service gRPC binary: multiple protobuf services on one grpc.Server"
  - "Gateway route composition: combining multiple gRPC calls into a single HTTP endpoint"

requirements-completed: [HR-04, HR-05]

# Metrics
duration: 35min
completed: 2026-02-19
---

# Phase 13 Plan 03: Time Tracking Service Summary

**ArbZG-compliant time tracking service with clock in/out lifecycle, break management, correction workflow, 29-RPC gRPC server, and 30+ gateway HTTP routes**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-02-19T21:41:00Z
- **Completed:** 2026-02-19T22:16:44Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Full time tracking service with ArbZG enforcement: 10h daily block, 11h rest period validation, auto-break deduction, severity levels (info/warning/error)
- HRGRPCServer implementing all 29 RPCs from hr.proto covering leave, time tracking, absence, employee, and settings domains
- 30+ gateway HTTP routes under /api/v1/hr/* with proper auth middleware and permission checks
- Biz binary extended to host both FinanceService and HRService on the same gRPC port
- 16 passing unit tests covering clock in/out lifecycle, break management, ArbZG compliance, and correction workflow

## Task Commits

Each task was committed atomically:

1. **Task 1: Time tracking service with ArbZG enforcement** - `9054b10` (feat)
2. **Task 2: gRPC server + gateway routes + biz main.go extension** - `67fa9e0` (feat)

## Files Created/Modified
- `backend/internal/biz/hr/timetracking/errors.go` - Domain errors (ErrAlreadyClockedIn, ErrMaxDailyHoursExceeded, etc.)
- `backend/internal/biz/hr/timetracking/repository.go` - Repository interfaces, filter/summary types, WorkTimeStatus
- `backend/internal/biz/hr/timetracking/postgres_repository.go` - PostgreSQL implementations for work time and break repos
- `backend/internal/biz/hr/timetracking/service.go` - Service with ClockIn/Out, breaks, summaries, corrections, status
- `backend/internal/biz/hr/timetracking/service_test.go` - 16 tests with mock repositories
- `backend/internal/server/hr_grpc.go` - HRGRPCServer implementing 29 RPCs with proto conversion helpers
- `backend/internal/gateway/route_hr.go` - HRRoutes with 30+ HTTP handlers under /api/v1/hr/*
- `backend/cmd/biz/main.go` - HR repo/service initialization and gRPC registration
- `backend/cmd/gateway/main.go` - HRRoutes added to registrars list
- `backend/api/openapi.yaml` - HR paths and 20+ schema definitions

## Decisions Made
- HR services registered on same gRPC server as finance (biz binary) at port :50058, sharing the connection
- GetWorkTimeStatus endpoint composed in gateway from GetActiveShift + GetDailySummary RPCs since no dedicated proto RPC exists
- ArbZG severity at exactly 600 minutes (10h) returns "warning" not "error" -- consistent with compliance engine where error is strictly > 600 minutes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed ArbZG severity assertion at 10h boundary**
- **Found during:** Task 1 (service_test.go)
- **Issue:** Test expected "error" severity at exactly 600 minutes, but CheckWorkTime(600) returns "warning" because the threshold is > 600 (not >= 600)
- **Fix:** Updated test assertion to expect "warning" with explanatory comment
- **Files modified:** backend/internal/biz/hr/timetracking/service_test.go
- **Verification:** Test passes, consistent with compliance engine behavior
- **Committed in:** 9054b10 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed mock employee repo interface mismatch**
- **Found during:** Task 1 (service_test.go)
- **Issue:** mockEmployeeRepo.List used interface{} parameter instead of employee.EmployeeFilter
- **Fix:** Added employee package import and corrected parameter type to employee.EmployeeFilter
- **Files modified:** backend/internal/biz/hr/timetracking/service_test.go
- **Verification:** Tests compile and pass
- **Committed in:** 9054b10 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for correctness. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All HR backend services complete: leave, absence, employee, time tracking, compliance
- 29 gRPC RPCs implemented and 30+ HTTP routes exposed
- Ready for Phase 13 Plan 04 (integration tests / verification)
- Frontend can begin integrating with /api/v1/hr/* endpoints

## Self-Check: PASSED

- All 7 created files verified present on disk
- Commit 9054b10 (Task 1) verified in git log
- Commit 67fa9e0 (Task 2) verified in git log
- `go build ./...` passes clean
- `go test ./internal/biz/hr/timetracking/...` all 16 tests pass

---
*Phase: 13-hr-zeiterfassung*
*Completed: 2026-02-19*
