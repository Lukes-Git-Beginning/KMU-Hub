---
phase: 12-rechnungen-finanzen
plan: 02
subsystem: api, database
tags: [go, service-layer, repository, postgres, decimal, tax, gobd, quote, invoice]

# Dependency graph
requires:
  - phase: 12-rechnungen-finanzen
    plan: 01
    provides: "Finance models, tax calculator, migrations, biz service binary scaffold"
provides:
  - "Quote service with full lifecycle (draft/send/accept/reject/expire) and deal value auto-sync"
  - "Invoice service with GoBD immutability, gap-free sequential numbering, and JSONB snapshots"
  - "PostgreSQL repositories for quotes and invoices with JSONB line items and tenant-scoped queries"
  - "NumberSequenceRepo and CompanySettingsRepo shared between quote and invoice services"
  - "CreateFromQuote copy-and-link pattern for quote-to-invoice conversion"
  - "DealValueUpdater interface for CRM deal value sync (decoupled from gRPC)"
affects: [12-03, 12-04, 12-05]

# Tech tracking
tech-stack:
  added: []
  patterns: [DealValueUpdater interface for decoupled CRM sync, GoBD immutability via snapshot-at-send, gap-free numbering via SELECT FOR UPDATE, shared NumberSequenceRepo across document types]

key-files:
  created:
    - backend/internal/biz/quote/service.go
    - backend/internal/biz/quote/repository.go
    - backend/internal/biz/quote/postgres_repository.go
    - backend/internal/biz/quote/errors.go
    - backend/internal/biz/invoice/service.go
    - backend/internal/biz/invoice/repository.go
    - backend/internal/biz/invoice/postgres_repository.go
    - backend/internal/biz/invoice/errors.go
  modified: []

key-decisions:
  - "DealValueUpdater as interface (not direct gRPC import) for testability and graceful degradation"
  - "Shared NumberSequenceRepo implementation (PostgresNumberSequenceRepo) reused by both quote and invoice services"
  - "CompanySettingsRepo as interface shared between both services for default values (payment terms, quote validity)"
  - "QuoteReader interface in invoice package avoids circular import with quote package"
  - "GoBD immutability enforced at service layer: ErrInvoiceImmutable for any non-draft modification"
  - "Invoice snapshot includes full document state at send time (customer, company, line items, tax, metadata)"

patterns-established:
  - "DealValueUpdater nil-safe pattern: if dealUpdater is nil, sync is silently skipped (standalone mode)"
  - "Graceful degradation on deal sync failure: log warning but do not fail the quote operation"
  - "Tax recalculation on any line item or tax mode change in both quote and invoice Update methods"
  - "CompanySettings fallback chain: explicit input > company settings > hardcoded default"

requirements-completed: [FIN-01, FIN-02]

# Metrics
duration: 6min
completed: 2026-02-18
---

# Phase 12 Plan 02: Quote and Invoice Services Summary

**Quote service with deal value auto-sync and invoice service with GoBD-compliant immutability, gap-free RE-numbering, and copy-and-link quote conversion**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-18
- **Completed:** 2026-02-18
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Quote service with full lifecycle (draft, send, accept, reject, expire) and DealValueUpdater interface for CRM deal value sync on create/update
- Invoice service with GoBD immutability enforcement (ErrInvoiceImmutable), gap-free sequential numbering (RE-2026-0001), and complete JSONB snapshot at send time
- CreateFromQuote copy-and-link conversion from accepted quotes to draft invoices
- Shared NumberSequenceRepo and CompanySettingsRepo patterns reusable by credit notes and dunning

## Task Commits

Each task was committed atomically:

1. **Task 1: Quote service with deal value auto-sync** - `de6b457` (feat)
2. **Task 2: Invoice service with GoBD immutability and sequential numbering** - `8904d77` (feat)

## Files Created/Modified
- `backend/internal/biz/quote/service.go` - Quote CRUD, lifecycle transitions, deal sync, tax calculation
- `backend/internal/biz/quote/repository.go` - Quote repository interface, NumberSequenceRepo, CompanySettingsRepo, DealValueUpdater interfaces
- `backend/internal/biz/quote/postgres_repository.go` - PostgreSQL quote repo, number sequence repo, company settings repo implementations
- `backend/internal/biz/quote/errors.go` - ErrQuoteNotFound, ErrQuoteNotDraft, ErrQuoteNotSent, ErrNoLineItems
- `backend/internal/biz/invoice/service.go` - Invoice CRUD, GoBD immutability, Send with snapshot, CreateFromQuote, DetectOverdue
- `backend/internal/biz/invoice/repository.go` - Invoice repository interface, QuoteReader interface
- `backend/internal/biz/invoice/postgres_repository.go` - PostgreSQL invoice repo with JSONB scan helpers
- `backend/internal/biz/invoice/errors.go` - ErrInvoiceImmutable, ErrInvoiceNotDraft, ErrInvoiceAlreadyPaid, ErrQuoteNotAccepted

## Decisions Made
- DealValueUpdater as interface in repository.go (not direct gRPC import) for testability and decoupled implementation via gRPC wrapper in Plan 04
- Shared PostgresNumberSequenceRepo in quote package reused by invoice service (same table, different document_type/prefix)
- QuoteReader interface in invoice package prevents circular dependency between quote and invoice packages
- Invoice Send() builds full snapshot as map[string]any with customer/company/financial data frozen at send time
- CompanySettings fallback: explicit input > company_settings table > hardcoded 30-day default
- Deal sync fires on both Create and Update (locked decision from CONTEXT.md) with graceful degradation on failure

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed undefined tax.Calculator type reference**
- **Found during:** Task 1 (Quote service)
- **Issue:** Plan specified `taxCalc *tax.Calculator` in constructor, but tax package uses package-level functions (no Calculator struct)
- **Fix:** Removed taxCalc field from Service struct, call tax.Calculate() directly as package-level function
- **Files modified:** backend/internal/biz/quote/service.go
- **Verification:** go build and go vet pass
- **Committed in:** de6b457 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor type reference fix. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Quote and invoice services ready for use by downstream services (credit note, payment, dunning in Plan 03)
- DealValueUpdater interface ready for gRPC wrapper implementation in Plan 04
- NumberSequenceRepo pattern reusable for credit note numbering (prefix GS)
- Both services importable by gRPC server layer for registration in Plan 04

## Self-Check: PASSED

All 8 created files verified present. Both task commits verified in git log (de6b457, 8904d77).

---
*Phase: 12-rechnungen-finanzen*
*Completed: 2026-02-18*
