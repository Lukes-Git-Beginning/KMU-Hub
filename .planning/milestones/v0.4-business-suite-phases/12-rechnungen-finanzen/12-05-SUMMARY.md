---
phase: 12-rechnungen-finanzen
plan: 05
subsystem: ui
tags: [react, tanstack-query, typescript, zustand, EUR, de-DE, finance, invoices, quotes, dunning, datev]

# Dependency graph
requires:
  - phase: 12-04
    provides: gRPC server, gateway finance routes, biz service Docker config
  - phase: 05-01
    provides: Electron + React + TypeScript desktop app foundation
  - phase: 10-02
    provides: Zustand mock stores for design integration modules
provides:
  - TypeScript types matching finance proto messages (finance-types.ts)
  - Finance API client with typed fetch wrapper for ~30 endpoints
  - TanStack Query hooks for all finance CRUD operations with cache invalidation
  - Complete finance UI: dashboard, invoices, quotes, credit notes, dunning, DATEV export
  - EUR/de-DE formatting throughout with 19%/7% German MwSt rates
affects: [14-unified-inbox, 18-bexio, 20-plugins]

# Tech tracking
tech-stack:
  added: []
  patterns: [finance-client fetch wrapper with blob download support, Zustand for UI-only state with TanStack Query for server data, formatEUR utility]

key-files:
  created:
    - desktop/src/renderer/src/types/finance-types.ts
    - desktop/src/renderer/src/api/finance-client.ts
    - desktop/src/renderer/src/api/hooks/useFinance.ts
    - desktop/src/renderer/src/modules/finanzen/QuoteFormDialog.tsx
    - desktop/src/renderer/src/modules/finanzen/CreditNoteDialog.tsx
    - desktop/src/renderer/src/modules/finanzen/DunningPanel.tsx
    - desktop/src/renderer/src/modules/finanzen/FinanceDashboard.tsx
  modified:
    - desktop/src/renderer/src/stores/finance.ts
    - desktop/src/renderer/src/modules/finanzen/FinanzenPage.tsx
    - desktop/src/renderer/src/modules/finanzen/InvoiceFormDialog.tsx
    - desktop/src/renderer/src/modules/finanzen/InvoiceDetailPanel.tsx
    - desktop/src/renderer/src/modules/finanzen/PaymentRecordDialog.tsx
    - desktop/src/renderer/src/modules/finanzen/ExportDialog.tsx
    - desktop/src/renderer/src/modules/finanzen/ExpenseFormDialog.tsx

key-decisions:
  - "Finance store (Zustand) holds only UI state (active tab, filters, date range); all server data via TanStack Query"
  - "formatEUR utility centralized in stores/finance.ts for consistent EUR/de-DE formatting"
  - "requestBlob helper added to finance-client.ts for PDF and CSV binary downloads"
  - "ExpenseFormDialog replaced with null stub (expenses not in Phase 12 scope)"
  - "DunningConfigDialog embedded inline within DunningPanel (no separate file)"
  - "FinanceDashboard uses date presets (Monat/Quartal/Jahr/Letztes Jahr) plus custom range"

patterns-established:
  - "Blob download pattern: requestBlob + URL.createObjectURL + temporary anchor element for PDF/CSV"
  - "Finance query key factory: ['finance', domain, ...params] for granular cache invalidation"
  - "Tax mode info banners: Reverse Charge and Kleinunternehmer show contextual UStG info"
  - "Line item draft pattern: local state with key/position/description/quantity/unit_price/tax_rate"

requirements-completed: [FIN-01, FIN-02, FIN-03, FIN-04, FIN-05, FIN-06, FIN-07]

# Metrics
duration: 11min
completed: 2026-02-18
---

# Phase 12 Plan 05: Frontend Finance Module Summary

**Complete finance UI with TanStack Query hooks replacing Zustand mocks: invoices, quotes, credit notes, dunning, dashboard, and DATEV export, all with EUR/de-DE formatting and 19%/7% German MwSt rates**

## Performance

- **Duration:** 11 min
- **Started:** 2026-02-18T01:53:50Z
- **Completed:** 2026-02-18T02:05:09Z
- **Tasks:** 2
- **Files modified:** 14

## Accomplishments
- Replaced entire Zustand mock data store with TanStack Query hooks backed by real finance API endpoints
- Created complete TypeScript type system matching proto messages with all finance domain types
- Built 4 new components: QuoteFormDialog, CreditNoteDialog, DunningPanel, FinanceDashboard
- Rewrote all existing components (FinanzenPage, InvoiceFormDialog, InvoiceDetailPanel, PaymentRecordDialog, ExportDialog) to use EUR/de-DE and API hooks
- Implemented 3-level dunning management panel with escalation, PDF download, and configuration dialog
- Added finance dashboard with revenue metrics, status breakdown, pipeline analytics, and date range filtering

## Task Commits

Each task was committed atomically:

1. **Task 1: TypeScript types + finance API client + TanStack Query hooks + store rewrite** - `3f76a6f` (feat)
2. **Task 2: FinanzenPage rewrite + all dialog/panel components** - `5b27a92` (feat)

## Files Created/Modified
- `desktop/src/renderer/src/types/finance-types.ts` - TypeScript interfaces matching proto messages (enums, domain types, request/response types)
- `desktop/src/renderer/src/api/finance-client.ts` - Typed fetch wrapper with auth/refresh for ~30 finance API endpoints including blob downloads
- `desktop/src/renderer/src/api/hooks/useFinance.ts` - TanStack Query hooks: 10 query hooks, 26 mutation hooks, 4 PDF download hooks
- `desktop/src/renderer/src/stores/finance.ts` - Rewritten to local UI state only (active tab, filters, date range) plus formatEUR/calc helpers
- `desktop/src/renderer/src/modules/finanzen/FinanzenPage.tsx` - Complete rewrite with 6 tabs, TanStack Query data, EUR formatting
- `desktop/src/renderer/src/modules/finanzen/InvoiceFormDialog.tsx` - Rewritten with 19%/7% MwSt, Reverse Charge, Kleinunternehmer modes
- `desktop/src/renderer/src/modules/finanzen/InvoiceDetailPanel.tsx` - Rewritten with API-backed data, payment history, immutability notice
- `desktop/src/renderer/src/modules/finanzen/PaymentRecordDialog.tsx` - Rewritten with EUR amounts, API mutation
- `desktop/src/renderer/src/modules/finanzen/ExportDialog.tsx` - Rewritten for DATEV Buchungsstapel CSV export
- `desktop/src/renderer/src/modules/finanzen/ExpenseFormDialog.tsx` - Replaced with null stub (not in scope)
- `desktop/src/renderer/src/modules/finanzen/QuoteFormDialog.tsx` - New: quote creation/editing with line items and tax modes
- `desktop/src/renderer/src/modules/finanzen/CreditNoteDialog.tsx` - New: credit note creation with invoice selection and partial credits
- `desktop/src/renderer/src/modules/finanzen/DunningPanel.tsx` - New: 3-level dunning management with config dialog
- `desktop/src/renderer/src/modules/finanzen/FinanceDashboard.tsx` - New: metrics cards, status breakdown, pipeline analytics, date range picker

## Decisions Made
- Finance store (Zustand) holds only UI state; all server data goes through TanStack Query -- consistent with established pattern from document module
- requestBlob helper added for binary downloads (PDFs and CSVs) since fetch wrapper defaults to JSON parsing
- ExpenseFormDialog becomes a null-rendering stub rather than deleted to prevent broken imports elsewhere
- DunningConfigDialog is an inline component within DunningPanel rather than a separate file to keep the dunning feature self-contained
- Tax mode banners reference UStG paragraphs for legal context (Paragraph 13b for Reverse Charge, Paragraph 19 for Kleinunternehmer)
- Dashboard uses metric cards rather than charts for simplicity; Recharts available but status breakdown uses CSS bars

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - TypeScript compiled cleanly after both tasks.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 12 (Rechnungen & Finanzen) is now complete with all 5 plans done
- Frontend finance module fully connected to backend API via TanStack Query hooks
- Ready for Phase 13 (HR) or any subsequent phase
- All EUR/de-DE formatting consistent across the entire finance module

## Self-Check: PASSED

All 14 created/modified files verified on disk. Both task commits (3f76a6f, 5b27a92) verified in git log. TypeScript compiles cleanly. Zero CHF references remaining in finance module.

---
*Phase: 12-rechnungen-finanzen*
*Completed: 2026-02-18*
