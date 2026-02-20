---
phase: 12-rechnungen-finanzen
plan: 03
subsystem: api, database
tags: [go, service-layer, repository, postgres, decimal, creditnote, payment, dunning, dashboard, pdf, maroto, bgb288]

# Dependency graph
requires:
  - phase: 12-rechnungen-finanzen
    plan: 02
    provides: "Quote and invoice services with GoBD immutability, gap-free numbering, and JSONB snapshots"
provides:
  - "Credit note service with GS-prefix numbering and original invoice validation"
  - "Payment service with auto-transition to paid when sum >= gross_total and revert on deletion"
  - "3-level dunning service with configurable intervals, fees, and BGB 288 Verzugszinsen calculation"
  - "Finance dashboard service with revenue, pipeline, and status metrics aggregation"
  - "PDF generator using maroto/v2 for quotes, invoices, credit notes, and dunning letters with Pflichtangaben"
affects: [12-04, 12-05]

# Tech tracking
tech-stack:
  added: []
  patterns: [InvoiceReader/InvoiceStatusUpdater interfaces for cross-service dependencies, ConfigRepository with upsert for tenant-scoped config, level-based dunning tone escalation, maroto/v2 A4 PDF generation with registered header/footer]

key-files:
  created:
    - backend/internal/biz/creditnote/service.go
    - backend/internal/biz/creditnote/repository.go
    - backend/internal/biz/creditnote/postgres_repository.go
    - backend/internal/biz/creditnote/errors.go
    - backend/internal/biz/payment/service.go
    - backend/internal/biz/payment/repository.go
    - backend/internal/biz/payment/postgres_repository.go
    - backend/internal/biz/dunning/service.go
    - backend/internal/biz/dunning/repository.go
    - backend/internal/biz/dunning/postgres_repository.go
    - backend/internal/biz/dunning/errors.go
    - backend/internal/biz/dashboard/service.go
    - backend/internal/biz/dashboard/repository.go
    - backend/internal/biz/dashboard/postgres_repository.go
    - backend/internal/biz/pdf/generator.go
    - backend/internal/biz/pdf/templates.go
  modified:
    - backend/internal/models/finance.go

key-decisions:
  - "InvoiceReader and InvoiceStatusUpdater as separate interfaces for payment service (read vs write separation)"
  - "ConfigRepository with upsert pattern for dunning config (auto-creates defaults on first access)"
  - "Dashboard forecast: average monthly revenue * remaining months in year (simple, no ML)"
  - "PDF uses maroto/v2 with registered footer for Pflichtangaben on every page"
  - "Dunning tone escalation: Zahlungserinnerung (friendly) -> 1. Mahnung (formal) -> 2. Mahnung (threatens Inkasso)"
  - "Credit note uses repo.Update for Send (not Create), added Update to Repository interface"

patterns-established:
  - "InvoiceReader interface pattern: cross-package read access to invoices without circular imports"
  - "InvoiceStatusUpdater interface: write-only status update for payment auto-transition"
  - "ConfigRepository with lazy default creation: GetConfig auto-creates if missing"
  - "Dunning level detection: iterate overdue invoices, check existing records, create next-level draft if interval elapsed"
  - "PDF template functions: composable row builders (buildHeader, buildFooter, buildLineItemRow, etc.) for document types"

requirements-completed: [FIN-04, FIN-07]

# Metrics
duration: 9min
completed: 2026-02-18
---

# Phase 12 Plan 03: Supporting Finance Services Summary

**Credit note, payment, dunning, dashboard, and PDF generation services completing the finance business logic layer with BGB 288 Verzugszinsen, auto-paid transitions, and maroto/v2 A4 documents**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-18T01:09:13Z
- **Completed:** 2026-02-18T01:18:23Z
- **Tasks:** 2
- **Files modified:** 17

## Accomplishments
- Credit note service with GS-prefix gap-free numbering, original invoice state validation (sent/paid/overdue only), and customer data copy from invoice
- Payment recording with automatic invoice-to-paid transition when sum of payments >= gross_total, and revert to sent/overdue on payment deletion
- 3-level semi-automatic dunning with configurable intervals and fees, DetectAndCreateDunnings batch detection, and BGB 288 Verzugszinsen (B2C +5%, B2B +9% over Basiszinssatz)
- Finance dashboard aggregating revenue, pipeline, status breakdown, recent invoices, expiring quotes, and pending dunnings with date range filtering and simple forecast
- PDF generator for all 4 document types (quotes, invoices, credit notes, dunning letters) with clean A4 layout, Pflichtangaben footer, accent color theming, and tax mode notes

## Task Commits

Each task was committed atomically:

1. **Task 1: Credit note, payment, and dunning service packages** - `5c5c48d` (feat)
2. **Task 2: Dashboard service + PDF generator** - `e544e7f` (feat)

## Files Created/Modified
- `backend/internal/biz/creditnote/service.go` - Credit note CRUD, Send with GS-prefix number, invoice state validation
- `backend/internal/biz/creditnote/repository.go` - Repository, InvoiceReader, NumberSequenceRepo interfaces
- `backend/internal/biz/creditnote/postgres_repository.go` - PostgreSQL credit note repo with JSONB scan helpers
- `backend/internal/biz/creditnote/errors.go` - ErrCreditNoteNotFound, ErrCreditNoteNotDraft, ErrInvalidInvoiceForCredit
- `backend/internal/biz/payment/service.go` - Payment Record with auto-paid transition, Delete with revert logic
- `backend/internal/biz/payment/repository.go` - Repository, InvoiceReader, InvoiceStatusUpdater interfaces
- `backend/internal/biz/payment/postgres_repository.go` - PostgreSQL payment repo with SumByInvoiceID aggregation
- `backend/internal/biz/dunning/service.go` - GetConfig, UpdateConfig, DetectAndCreateDunnings, Send, CalculateInterest
- `backend/internal/biz/dunning/repository.go` - Repository, ConfigRepository, InvoiceReader interfaces
- `backend/internal/biz/dunning/postgres_repository.go` - PostgreSQL dunning + config repos with upsert
- `backend/internal/biz/dunning/errors.go` - ErrDunningNotFound, ErrDunningNotDraft, ErrDunningMaxLevel
- `backend/internal/biz/dashboard/service.go` - GetDashboard with forecast calculation
- `backend/internal/biz/dashboard/repository.go` - Repository interface, Metrics struct
- `backend/internal/biz/dashboard/postgres_repository.go` - Multi-query aggregation (6 queries)
- `backend/internal/biz/pdf/generator.go` - GenerateQuotePDF, GenerateInvoicePDF, GenerateCreditNotePDF, GenerateDunningPDF
- `backend/internal/biz/pdf/templates.go` - Composable PDF layout builders (header, footer, line items, totals, dunning body)
- `backend/internal/models/finance.go` - Added FinanceDashboard, RevenueMetrics, PipelineMetrics, InvoiceStatusBreakdown models

## Decisions Made
- InvoiceReader and InvoiceStatusUpdater as separate interfaces in payment package: read vs write separation for clean dependency injection
- ConfigRepository with upsert and lazy default creation: GetConfig auto-creates 14/14/14 day, 0/5/10 EUR defaults on first access
- Dashboard forecast uses simple average monthly revenue extrapolation (no ML complexity for v1)
- PDF uses maroto/v2 registered footer for Pflichtangaben on every page (per UStG section 14)
- Dunning tone strings hardcoded in German (Deutschland-First per strategy decision)
- Credit note Send uses repo.Update (not Create) -- added Update method to Repository interface during implementation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added Update method to credit note Repository interface**
- **Found during:** Task 1 (Credit note service)
- **Issue:** Plan specified only Create, GetByID, List, SendCreditNote, GetByInvoiceID in repository but Send needs to update the existing record (not re-insert)
- **Fix:** Added Update method to Repository interface and PostgresRepository implementation
- **Files modified:** backend/internal/biz/creditnote/repository.go, backend/internal/biz/creditnote/postgres_repository.go
- **Verification:** go build and go vet pass
- **Committed in:** 5c5c48d (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor interface fix. Repository needed Update for Send to work correctly. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 7 service packages complete (tax, quote, invoice, creditnote, payment, dunning, dashboard, pdf)
- Ready for gRPC server wiring in Plan 04
- PDF generator directly usable by gRPC handlers for document download endpoints
- Dashboard repository ready to be instantiated with pgxpool in gRPC server
- InvoiceReader/InvoiceStatusUpdater interfaces bridge payment and invoice packages at wiring time

## Self-Check: PASSED

All 16 created files and 1 modified file verified present. Both task commits verified in git log (5c5c48d, e544e7f).

---
*Phase: 12-rechnungen-finanzen*
*Completed: 2026-02-18*
