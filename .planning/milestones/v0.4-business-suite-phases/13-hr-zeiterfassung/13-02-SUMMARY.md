---
phase: 13-hr-zeiterfassung
plan: 02
subsystem: api, database
tags: [go, postgresql, leave-management, absence-calendar, employee-profiles, burlg, rbac]

# Dependency graph
requires:
  - phase: 13-hr-zeiterfassung
    provides: HR proto (29 RPCs), migration 000046 (9 tables), models, BUrlG/ArbZG compliance engine
provides:
  - Leave request lifecycle service (create/approve/reject/cancel)
  - Leave balance management with BUrlG compliance integration
  - Absence calendar query service with configurable reason visibility
  - Employee profile CRUD with role-based field restrictions
  - Employee document management with category-based visibility
  - Repository interfaces and PostgreSQL implementations for all HR entities
affects: [13-03 gRPC server, 13-04 frontend]

# Tech tracking
tech-stack:
  added: []
  patterns: [leave approval state machine, role-based field restrictions, category-based document visibility, overlap-warn-but-allow pattern]

key-files:
  created:
    - backend/internal/biz/hr/leave/errors.go
    - backend/internal/biz/hr/leave/repository.go
    - backend/internal/biz/hr/leave/postgres_repository.go
    - backend/internal/biz/hr/leave/service.go
    - backend/internal/biz/hr/leave/service_test.go
    - backend/internal/biz/hr/absence/errors.go
    - backend/internal/biz/hr/absence/repository.go
    - backend/internal/biz/hr/absence/postgres_repository.go
    - backend/internal/biz/hr/absence/service.go
    - backend/internal/biz/hr/absence/service_test.go
    - backend/internal/biz/hr/employee/errors.go
    - backend/internal/biz/hr/employee/repository.go
    - backend/internal/biz/hr/employee/postgres_repository.go
    - backend/internal/biz/hr/employee/service.go
    - backend/internal/biz/hr/employee/service_test.go
  modified: []

key-decisions:
  - "Leave service uses EmployeeRepository interface for manager lookup (cross-package dependency via interface)"
  - "Overlap detection warns but allows approval (returns overlaps in ApproveResult for gRPC to surface)"
  - "HR fallback: when no manager assigned, gRPC layer enforces HR role for approval authorization"
  - "Employee self-service uses hasRestrictedFields() check rather than field-level allowlist for cleaner code"
  - "Absence calendar masks leave types to 'Abwesend' with neutral gray when ShowAbsenceReason is false"
  - "Leave balance auto-created on first access using BUrlG compliance engine with previous year carryover"

patterns-established:
  - "Leave approval state machine: pending -> approved/rejected/cancelled with balance deduction/restoration"
  - "Role-based field restrictions: callerRole string checked in service layer before field updates"
  - "Category-based document visibility: PostgreSQL query filters JOIN with categories based on callerRole"
  - "Cross-package interfaces: leave.EmployeeRepository interface avoids circular imports with employee package"

requirements-completed: [HR-01, HR-02, HR-03, HR-06, HR-07]

# Metrics
duration: 9min
completed: 2026-02-19
---

# Phase 13 Plan 02: HR Services Summary

**Leave approval workflow with BUrlG balance enforcement, absence calendar with reason masking, and employee profile CRUD with role-based self-service restrictions -- 34 tests across 3 service packages**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-19T21:48:22Z
- **Completed:** 2026-02-19T21:57:38Z
- **Tasks:** 2
- **Files modified:** 15

## Accomplishments
- Leave service with full request/approve/reject/cancel lifecycle, BUrlG balance enforcement via compliance engine, half-day support, overlap warnings, sick leave auto-approval with AU threshold flagging
- Absence calendar service that queries approved leaves with department filtering and configurable reason visibility (replaces leave types with "Abwesend" when company setting disabled)
- Employee profile CRUD with role-based field restrictions: employees can only update emergency contact and address; admin/HR/manager can update department, position, contract type
- Document management with category-based access control: hr_only categories hidden from managers and employees, manager categories hidden from employees

## Task Commits

Each task was committed atomically:

1. **Task 1: Leave service + absence service** - `c545838` (feat)
2. **Task 2: Employee service with profile CRUD + document management** - `e5a4ddc` (feat)

## Files Created/Modified
- `backend/internal/biz/hr/leave/errors.go` - Domain errors for leave lifecycle
- `backend/internal/biz/hr/leave/repository.go` - LeaveRequestRepository, LeaveBalanceRepository, LeaveTypeRepository, HRSettingsRepository interfaces
- `backend/internal/biz/hr/leave/postgres_repository.go` - PostgreSQL implementations with overlap detection, pagination, upsert balance
- `backend/internal/biz/hr/leave/service.go` - Leave approval state machine with BUrlG balance, AU flagging, manager/HR auth
- `backend/internal/biz/hr/leave/service_test.go` - 20 tests covering all leave lifecycle flows
- `backend/internal/biz/hr/absence/errors.go` - ErrInvalidDateRange
- `backend/internal/biz/hr/absence/repository.go` - AbsenceRepository interface, AbsenceFilter/AbsenceEntry types
- `backend/internal/biz/hr/absence/postgres_repository.go` - JOIN query with date range overlap and department filter
- `backend/internal/biz/hr/absence/service.go` - Absence calendar with configurable reason visibility
- `backend/internal/biz/hr/absence/service_test.go` - 4 tests for visibility masking and fallback behavior
- `backend/internal/biz/hr/employee/errors.go` - Domain errors for employee and document operations
- `backend/internal/biz/hr/employee/repository.go` - EmployeeRepository, DocumentCategoryRepository, EmployeeDocumentRepository interfaces
- `backend/internal/biz/hr/employee/postgres_repository.go` - PostgreSQL implementations with user JOIN for denormalized fields, visibility-filtered document queries
- `backend/internal/biz/hr/employee/service.go` - Employee profile CRUD with role-based restrictions, document management with category validation
- `backend/internal/biz/hr/employee/service_test.go` - 14 tests covering role restrictions and document visibility

## Decisions Made
- Leave service defines its own `EmployeeRepository` interface (single method: GetByUserID) to avoid circular import with employee package while enabling manager lookup for approval authorization
- Overlap detection returns overlaps in `ApproveResult` struct rather than blocking -- gRPC layer can surface these as warnings to the approver
- HR fallback approval: when employee has no manager assigned, service allows approval (returns nil from verifyApprover); gRPC layer enforces that caller has HR role
- Balance auto-creation: `getOrCreateBalance` creates a new balance record on first access using BUrlG compliance calculation with previous year carryover
- Sick leave auto-approval: leave types with `requires_approval=false` get status "approved" immediately at creation time with employee as approver
- Employee self-service check uses `hasRestrictedFields()` function that checks if any HR-only fields are set, rather than field-level allowlist iteration

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed decimal comparison in half-day tests**
- **Found during:** Task 1 (Leave service tests)
- **Issue:** `assert.Equal` fails for shopspring/decimal values with equivalent but different internal representation (e.g., `2` vs `2.0`)
- **Fix:** Changed to `decimal.Equal()` method for comparison in test assertions
- **Files modified:** backend/internal/biz/hr/leave/service_test.go
- **Committed in:** c545838

**2. [Rule 1 - Bug] Fixed overlap assertion expecting non-nil for empty result**
- **Found during:** Task 1 (Leave service tests)
- **Issue:** `assert.NotNil` on overlaps when mock returns nil (no overlaps)
- **Fix:** Changed to `assert.Empty` which handles both nil and empty slices
- **Files modified:** backend/internal/biz/hr/leave/service_test.go
- **Committed in:** c545838

---

**Total deviations:** 2 auto-fixed (2 bugs in test assertions)
**Impact on plan:** Both fixes are test-only assertion corrections. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All three service packages ready for gRPC server wiring (Plan 03)
- Leave service consumes compliance.CalculateLeaveBalance from Plan 01
- Employee service EmployeeRepository interface compatible with leave service's EmployeeRepository interface (same GetByUserID signature)
- PostgreSQL repository implementations ready for dependency injection in gRPC server

## Self-Check: PASSED

- All 15 created files verified present on disk
- Task 1 commit `c545838` verified in git log
- Task 2 commit `e5a4ddc` verified in git log
- All 34 tests pass across leave (20), absence (4), employee (14) packages

---
*Phase: 13-hr-zeiterfassung*
*Completed: 2026-02-19*
