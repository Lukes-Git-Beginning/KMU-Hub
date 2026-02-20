---
phase: 13-hr-zeiterfassung
plan: 01
subsystem: api, database, compliance
tags: [grpc, protobuf, postgresql, arbzg, burlg, tdd, decimal, go]

# Dependency graph
requires:
  - phase: 12-rechnungen-finanzen
    provides: biz service pattern, shopspring/decimal dependency, migration numbering
provides:
  - HRService gRPC definition with 29 RPCs
  - Migration 000046 with 9 HR tables and seeded data
  - Go domain models for all HR entities
  - BUrlG leave balance compliance engine (pure functions)
  - ArbZG work time validation engine (pure functions)
affects: [13-02 leave service, 13-03 time tracking service, 13-04 frontend]

# Tech tracking
tech-stack:
  added: []
  patterns: [pure compliance functions (no I/O), TDD RED-GREEN for labor law rules]

key-files:
  created:
    - backend/proto/hr/v1/hr.proto
    - backend/proto/hr/v1/hr.pb.go
    - backend/proto/hr/v1/hr_grpc.pb.go
    - backend/migrations/000046_create_hr_tables.up.sql
    - backend/migrations/000046_create_hr_tables.down.sql
    - backend/internal/models/hr.go
    - backend/internal/biz/hr/compliance/types.go
    - backend/internal/biz/hr/compliance/burlg.go
    - backend/internal/biz/hr/compliance/burlg_test.go
    - backend/internal/biz/hr/compliance/arbzg.go
    - backend/internal/biz/hr/compliance/arbzg_test.go
  modified: []

key-decisions:
  - "29 RPCs in single HRService proto (leave, time, absences, employees, settings)"
  - "System leave types and doc categories seeded with zero UUID tenant_id for per-tenant copy pattern"
  - "Partial unique index on active shifts ensures single active shift per employee at DB level"
  - "Pure compliance functions use shopspring/decimal throughout for half-day precision"
  - "BUrlG carryover expires after March 31 (inclusive) with CarryoverExpired flag"

patterns-established:
  - "Pure compliance package: no I/O, no context, pure functions for labor law rules"
  - "TDD RED-GREEN: failing tests first, then minimal implementation, 98% coverage"
  - "HR as sub-domain in biz package tree: internal/biz/hr/compliance/"

requirements-completed: [HR-01, HR-02, HR-04, HR-05]

# Metrics
duration: 7min
completed: 2026-02-19
---

# Phase 13 Plan 01: HR Data Foundation Summary

**HR proto with 29 RPCs, 9-table migration, BUrlG/ArbZG compliance engine with 27 TDD tests at 98% coverage**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-19T21:37:32Z
- **Completed:** 2026-02-19T21:44:56Z
- **Tasks:** 2 (3 commits: 1 feat + 1 test RED + 1 feat GREEN)
- **Files modified:** 11

## Accomplishments
- HRService proto with 29 RPCs covering leave management, time tracking, absences, employee profiles, and HR settings
- Migration 000046 creates 9 tables with constraints, indexes, and seeded system leave types (10) and document categories (4)
- Go domain models with decimal.Decimal for leave balances and proper nullable types
- BUrlG compliance engine: pro-rata for part-time and mid-year starts, carryover with March 31 expiry, half-day support
- ArbZG compliance engine: break calculation (30min/45min), auto-deduction deficit, severity thresholds, 11h rest period validation

## Task Commits

Each task was committed atomically:

1. **Task 1: HR proto + migrations + models** - `e81f8ba` (feat)
2. **Task 2 RED: Failing BUrlG and ArbZG tests** - `6dbedf9` (test)
3. **Task 2 GREEN: BUrlG and ArbZG implementation** - `8bf9315` (feat)

## Files Created/Modified
- `backend/proto/hr/v1/hr.proto` - HRService gRPC definition with 29 RPCs, enums, domain messages
- `backend/proto/hr/v1/hr.pb.go` - Generated protobuf Go code
- `backend/proto/hr/v1/hr_grpc.pb.go` - Generated gRPC Go code
- `backend/migrations/000046_create_hr_tables.up.sql` - 9 HR tables with constraints, indexes, seed data
- `backend/migrations/000046_create_hr_tables.down.sql` - Reverse migration dropping all HR tables
- `backend/internal/models/hr.go` - Go domain structs for all HR entities
- `backend/internal/biz/hr/compliance/types.go` - Shared types for compliance functions
- `backend/internal/biz/hr/compliance/burlg.go` - BUrlG leave balance calculation
- `backend/internal/biz/hr/compliance/burlg_test.go` - 12 BUrlG test cases
- `backend/internal/biz/hr/compliance/arbzg.go` - ArbZG work time validation
- `backend/internal/biz/hr/compliance/arbzg_test.go` - 15 ArbZG test cases

## Decisions Made
- 29 RPCs in a single HRService (not splitting into multiple services) for simplicity and consistency with FinanceService pattern
- System leave types and document categories use zero UUID as tenant_id placeholder; application layer copies these per tenant on first access
- Partial unique index on `hr_work_time_entries(employee_id) WHERE status = 'active'` enforces single active shift per employee at database level
- BUrlG carryover comparison: expires when CalculationDate is on or after April 1 (March 31 is the last valid day)
- ArbZG severity at exactly 8h is Info (not None) and at exactly 9h is Warning (not Info), matching the "warns at 8h, warns harder at 9h" requirement

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Proto definitions ready for service layer implementation (Plan 02)
- Compliance package ready to be consumed by leave and time tracking services
- Migration ready to run against development database
- Models ready for repository and service layer usage

---
*Phase: 13-hr-zeiterfassung*
*Completed: 2026-02-19*
