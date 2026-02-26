---
phase: 18-bexio-integration
plan: 02
subsystem: bexio
tags: [sync-engine, contact-sync, invoice-push, payment-poll, scheduler]

requires:
  - phase: 18-bexio-integration
    plan: 01
    provides: Bexio API client, repository, domain models, database schema
provides:
  - Bidirectional contact sync with last-write-wins conflict resolution
  - Invoice and quote push to Bexio API
  - Payment status polling with automatic mark-paid
  - Configurable field mapping engine with default DACH mappings
  - Background scheduler with per-tenant goroutines
  - Central Bexio service orchestrating all sync operations
  - OAuth2 connect/disconnect handler
affects: [18-03, 18-04]

tech-stack:
  added: []
  patterns: [last-write-wins-sync, per-tenant-scheduler, field-mapper-engine]

key-files:
  created:
    - backend/internal/biz/bexio/field_mapper.go
    - backend/internal/biz/bexio/lookup_cache.go
    - backend/internal/biz/bexio/oauth_handler.go
    - backend/internal/biz/bexio/event_emitter.go
    - backend/internal/biz/bexio/contact_sync.go
    - backend/internal/biz/bexio/invoice_push.go
    - backend/internal/biz/bexio/quote_push.go
    - backend/internal/biz/bexio/payment_poll.go
    - backend/internal/biz/bexio/service.go
    - backend/internal/biz/bexio/scheduler.go
  modified: []

key-decisions:
  - "ContactService/InvoiceReader/QuoteReader interfaces defined in bexio package to avoid circular imports"
  - "ContactSyncData intermediate struct decouples Bexio API types from CRM service types"
  - "Last-write-wins uses bexio_updated_at vs kmuhub_updated_at from entity mapping table"
  - "parseBexioTime supports RFC3339, datetime, and date-only formats for API response flexibility"
  - "IntegrationConfigRepo interface for shared integration_configs table access (same table as Teams/Slack)"
  - "OAuthHandler creates default sync config + field mappings for all 3 entity types on connect"
  - "Scheduler uses per-tenant goroutines with context cancellation for graceful shutdown"
  - "Payment poller skips invoices already marked paid in KMU Hub for efficiency"
  - "noopEmitter default for EventEmitter (same pattern as invoice/quote services)"
  - "Single-tenant scheduler mode with comment for future multi-tenant extension"

patterns-established:
  - "Bidirectional sync with last-write-wins conflict resolution"
  - "Per-tenant background scheduler with context-based lifecycle"
  - "Field mapper with default mappings and custom override support"
  - "Intermediate sync data types to decouple external API from internal models"

requirements-completed: []

duration: 18min
completed: 2026-02-26
---

# Phase 18 Plan 02: Bexio Sync Engine Summary

**Built complete sync engine: bidirectional contacts, invoice/quote push, payment polling, field mapping, scheduler.**

## Performance
- **Duration:** 18 min
- **Tasks:** 3
- **Files created:** 10
- **Files modified:** 0

## Accomplishments
- Field mapper engine with bidirectional contact mapping (KMU Hub <-> Bexio), invoice/quote mapping (outbound), default DACH mappings, and validation
- Lookup cache for Bexio reference data (countries, currencies, salutations) with thread-safe access
- OAuth handler for connect/disconnect flow with automatic default config creation
- Contact syncer with full sync (initial) and delta sync (subsequent), last-write-wins conflict resolution using timestamps from both systems
- Invoice pusher triggered on status change to "sent", with entity mapping tracking
- Quote pusher for outbound push on create/update
- Payment poller that checks Bexio payment status and auto-marks paid invoices in KMU Hub
- Central Service struct orchestrating all sync operations with event emission
- Background scheduler with per-tenant goroutines, configurable intervals (15min contacts, 5min payments), context-based lifecycle management

## Task Commits
1. **Task 1-3: Sync Engine** - `91d84d6` (feat)

## Decisions Made
- Interface-based dependencies (ContactService, InvoiceReader, etc.) defined within bexio package
- ContactSyncData as intermediate type prevents coupling to CRM models
- Scheduler runs single-tenant mode with clear extension point for multi-tenant
- Payment poller skips already-paid invoices for O(n) vs O(n*m) efficiency

## Deviations from Plan
- Plan specified `StartScheduler` method name, implemented as `StartAll`/`StopAll` for clarity
- Added `IntegrationConfigRepo` interface and `IntegrationConfig` type in oauth_handler.go for shared table access

## Issues Encountered
None

## Self-Check: PASSED
- ✅ SyncContacts in contact_sync.go
- ✅ PushInvoice in invoice_push.go
- ✅ PollPayments in payment_poll.go
- ✅ StartAll (scheduler) in scheduler.go
- ✅ type Service struct in service.go
- ✅ MapContactToBexio in field_mapper.go
- ✅ No fmt.Println in bexio package
- ✅ go build ./internal/biz/bexio/... succeeds

---
*Phase: 18-bexio-integration*
*Completed: 2026-02-26*
