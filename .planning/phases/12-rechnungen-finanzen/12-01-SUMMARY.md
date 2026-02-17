---
phase: 12-rechnungen-finanzen
plan: 01
subsystem: api, database
tags: [grpc, protobuf, postgres, decimal, tax, finance, tdd]

# Dependency graph
requires:
  - phase: 11-documents-files
    provides: "Sequential service port pattern (document :50057/:9097), migration numbering (000044)"
provides:
  - "FinanceService gRPC proto with 34 RPCs (quotes, invoices, credit notes, payments, dunning, dashboard, DATEV)"
  - "Migration 000045 with 8 finance tables (company_settings, number_sequences, quotes, invoices, credit_notes, payments, dunning_records, dunning_config)"
  - "Go domain models with decimal.Decimal for all monetary fields"
  - "Tax calculator with 100% coverage handling Standard/Reverse Charge/Kleinunternehmer modes"
  - "Biz service binary scaffold on :50058 (gRPC) and :9098 (health)"
  - "maroto/v2 dependency for PDF generation in subsequent plans"
affects: [12-02, 12-03, 12-04, 12-05]

# Tech tracking
tech-stack:
  added: [maroto/v2 (PDF generation)]
  patterns: [per-line tax rounding to 2dp, decimal-string proto encoding, JSONB for line items/snapshots]

key-files:
  created:
    - backend/proto/biz/v1/biz.proto
    - backend/proto/biz/v1/biz.pb.go
    - backend/proto/biz/v1/biz_grpc.pb.go
    - backend/migrations/000045_create_finance_tables.up.sql
    - backend/migrations/000045_create_finance_tables.down.sql
    - backend/internal/models/finance.go
    - backend/cmd/biz/main.go
    - backend/tools/biz_deps.go
    - backend/internal/biz/tax/calculator.go
    - backend/internal/biz/tax/calculator_test.go
  modified:
    - backend/internal/config/config.go
    - backend/go.mod
    - backend/go.sum

key-decisions:
  - "34 RPCs in FinanceService covering full quote-to-payment lifecycle plus dunning and DATEV export"
  - "All monetary values as string in proto (protobuf has no native decimal type) and decimal.Decimal in Go"
  - "JSONB for line_items, tax_breakdown, snapshot_data, company_snapshot to preserve document-level flexibility"
  - "Per-line rounding to 2 decimal places prevents cent discrepancies in tax totals"
  - "TaxByRate keys use truncated rate strings (e.g., '19' not '19.00') for clean aggregation"

patterns-established:
  - "Tax calculator as pure function: stateless, no DB, deterministic, fully tested"
  - "Biz service binary follows document service pattern: gRPC + health/metrics HTTP"
  - "Customer/Company snapshots denormalized into financial documents for immutability"

requirements-completed: [FIN-01, FIN-02, FIN-03, FIN-07]

# Metrics
duration: 12min
completed: 2026-02-18
---

# Phase 12 Plan 01: Data Foundation Summary

**FinanceService proto with 34 RPCs, 8 database tables, Go models with shopspring/decimal, and TDD-verified tax calculator with 100% coverage**

## Performance

- **Duration:** 12 min
- **Started:** 2026-02-18
- **Completed:** 2026-02-18
- **Tasks:** 2 (Task 2 had TDD RED/GREEN sub-phases)
- **Files modified:** 13

## Accomplishments
- FinanceService proto defines complete quote-to-payment lifecycle with 34 RPCs across 8 operation groups
- Migration 000045 creates all finance tables with proper constraints, FKs, indexes, and CHECK constraints
- Tax calculator handles all 4 German tax modes with per-line decimal rounding and 100% test coverage
- Biz service binary compiles and scaffolds gRPC + health/metrics servers

## Task Commits

Each task was committed atomically:

1. **Task 1: Proto + migrations + models + config + biz binary scaffold** - `73980f8` (feat)
2. **Task 2 RED: Failing tax calculator tests** - `3d30308` (test)
3. **Task 2 GREEN: Tax calculator implementation** - `599e0cb` (feat)

## Files Created/Modified
- `backend/proto/biz/v1/biz.proto` - FinanceService gRPC definition with 34 RPCs
- `backend/proto/biz/v1/biz.pb.go` - Generated protobuf Go code
- `backend/proto/biz/v1/biz_grpc.pb.go` - Generated gRPC Go code
- `backend/migrations/000045_create_finance_tables.up.sql` - 8 finance tables with indexes
- `backend/migrations/000045_create_finance_tables.down.sql` - Reverse migration
- `backend/internal/models/finance.go` - Domain models with decimal.Decimal for money
- `backend/internal/config/config.go` - Added BizGRPCPort :50058, BizHealthPort :9098
- `backend/cmd/biz/main.go` - Biz service entry point with graceful shutdown
- `backend/tools/biz_deps.go` - Retains maroto/v2 in go.mod
- `backend/internal/biz/tax/calculator.go` - Pure-function tax engine
- `backend/internal/biz/tax/calculator_test.go` - 15 test cases, 100% coverage

## Decisions Made
- Used string encoding for all decimal values in proto (protobuf lacks native decimal)
- JSONB columns for line_items, tax_breakdown, snapshot_data for flexible document storage
- Per-line rounding (not total-level) matches German accounting practice and prevents cent errors
- TaxByRate keys truncated to integer strings (e.g., "19") for clean map lookups
- No FK from invoices to quotes (source_quote_id is optional reference, not enforced)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Proto definitions ready for repository and service layer implementation (plan 12-02)
- Tax calculator importable by invoice/quote services for automatic tax computation
- Migration ready to run against development database
- Biz binary ready for gRPC service registration in plan 12-02

## Self-Check: PASSED

All 10 created files verified present. All 3 task commits verified in git log (73980f8, 3d30308, 599e0cb).

---
*Phase: 12-rechnungen-finanzen*
*Completed: 2026-02-18*
