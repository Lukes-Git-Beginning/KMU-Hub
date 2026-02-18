---
phase: 12-rechnungen-finanzen
plan: 07
subsystem: api, ui
tags: [grpc, gateway, crm, finance, quotes, deals, react, tanstack-query]

# Dependency graph
requires:
  - phase: 12-rechnungen-finanzen
    provides: "Quote service with deal_id support (12-02), gateway routes (12-04), useCreateQuoteFromDeal hook (12-05)"
provides:
  - "POST /api/v1/finance/deals/{dealId}/quote gateway route with CRM data pre-population"
  - "Angebot erstellen button on DealDetailPage connecting CRM deals to finance quotes"
  - "Cross-service gRPC call pattern (BizRoutes calling CRM service)"
affects: [crm, finance, unified-inbox]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Cross-service gRPC: BizRoutes.getCRMClient() for CRM data enrichment in finance gateway"
    - "Customer snapshot pre-population from CRM deal/contact/company hierarchy"

key-files:
  modified:
    - "backend/internal/gateway/route_biz.go"
    - "desktop/src/renderer/src/modules/crm/deals/DealDetailPage.tsx"

key-decisions:
  - "Customer name prefers company over contact (B2B DACH invoicing norm)"
  - "Contact/company fetch errors are graceful (quote still created with partial data)"
  - "Tax mode defaults to standard (19%), user adjusts in quote form"

patterns-established:
  - "Cross-service gateway pattern: route handler obtains multiple gRPC clients for data enrichment"

requirements-completed: [FIN-05]

# Metrics
duration: 5min
completed: 2026-02-18
---

# Phase 12 Plan 07: Deal-to-Quote Gateway Route and UI Trigger Summary

**Cross-service gateway route fetching CRM deal/contact/company data to pre-populate finance quotes, with Angebot erstellen button on DealDetailPage**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-18T11:07:44Z
- **Completed:** 2026-02-18T11:12:32Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- POST /api/v1/finance/deals/{dealId}/quote route created with full CRM data pre-population
- DealDetailPage shows "Angebot erstellen" button that creates quotes from deals seamlessly
- Previously orphaned useCreateQuoteFromDeal hook is now consumed in UI
- FIN-05 requirement ("User can convert a CRM deal to a quote in a seamless flow") satisfied

## Task Commits

Each task was committed atomically:

1. **Task 1: Add deal-to-quote gateway route with CRM data pre-population** - `c920863` (feat)
2. **Task 2: Add Angebot erstellen button to DealDetailPage** - `354b888` (feat)

## Files Created/Modified
- `backend/internal/gateway/route_biz.go` - Added getCRMClient method, deal-to-quote route registration, HandleCreateQuoteFromDeal handler
- `desktop/src/renderer/src/modules/crm/deals/DealDetailPage.tsx` - Added useCreateQuoteFromDeal hook, FileText icon, Angebot erstellen button with loading state

## Decisions Made
- Customer name prefers company_name over contact_name -- B2B DACH norm where invoices address the company
- Contact and company fetch errors are handled gracefully (not fatal) -- quote is still created with whatever data is available
- Tax mode defaults to TAX_MODE_STANDARD (19% MWSt) -- user can change in the quote form
- Line items left empty -- user fills them in via the quote editor
- ValidUntil left empty -- uses company default from CompanySettings

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- FIN-05 gap closure complete -- all Phase 12 gap closure plans (12-06, 12-07) done
- Phase 12 (Rechnungen & Finanzen) is fully complete with all 7 plans executed
- Ready for Phase 13 (HR)

## Self-Check: PASSED

- [x] backend/internal/gateway/route_biz.go -- FOUND
- [x] desktop/src/renderer/src/modules/crm/deals/DealDetailPage.tsx -- FOUND
- [x] .planning/phases/12-rechnungen-finanzen/12-07-SUMMARY.md -- FOUND
- [x] Commit c920863 -- FOUND in git log
- [x] Commit 354b888 -- FOUND in git log

---
*Phase: 12-rechnungen-finanzen*
*Completed: 2026-02-18*
