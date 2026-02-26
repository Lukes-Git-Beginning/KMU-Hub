---
phase: 18-bexio-integration
plan: 01
subsystem: bexio
tags: [integration, oauth2, api-client, data-foundation]

requires:
  - phase: 17.5-guest-chat
    plan: 03
    provides: Phase 17.5 complete, all prior infrastructure available
provides:
  - Bexio REST API v2.0 client with OAuth2 token management
  - Rate limiter respecting X-RateLimit-* headers
  - Database schema for sync tracking (configs, entity mappings, field mappings, sync log)
  - Repository layer for all Bexio-specific tables
  - Domain models for Bexio API types and KMU Hub persistence
affects: [18-02, 18-03, 18-04]

tech-stack:
  added: []
  patterns: [vault-backed-oauth, adaptive-rate-limiter, bexio-api-quirks]

key-files:
  created:
    - backend/migrations/000055_add_bexio_integration.up.sql
    - backend/migrations/000055_add_bexio_integration.down.sql
    - backend/internal/models/bexio.go
    - backend/internal/biz/bexio/models.go
    - backend/internal/biz/bexio/errors.go
    - backend/internal/biz/bexio/types.go
    - backend/internal/biz/bexio/auth.go
    - backend/internal/biz/bexio/rate_limiter.go
    - backend/internal/biz/bexio/client.go
    - backend/internal/biz/bexio/contacts.go
    - backend/internal/biz/bexio/invoices.go
    - backend/internal/biz/bexio/quotes.go
    - backend/internal/biz/bexio/repository.go
    - backend/internal/biz/bexio/postgres_repository.go
  modified: []

key-decisions:
  - "VaultService interface in types.go matches vault.Service signature (ctx + createdBy uuid) for direct implementation"
  - "Token cache uses 30s safety margin before expiry to prevent edge-case token usage during refresh"
  - "Rate limiter uses adaptive approach: starts with defaults (50/10s), adjusts from X-RateLimit-* headers"
  - "Client.do() handles 429 with exponential backoff (1s, 2s, 4s) and 401 with single token refresh retry"
  - "Bexio API quirks encoded: POST for updates, Content-Length:0 for GET, salutation_id 0 = none"
  - "GetFieldMappings returns nil (not error) when no mapping exists, following GetLatestSyncLog pattern"
  - "BexioListResponse generic type for future typed pagination"
  - "UpdateLastSyncTime uses column name switch instead of dynamic SQL for safety (only 2 valid columns)"

patterns-established:
  - "VaultService interface pattern for decoupled OAuth token storage"
  - "Adaptive rate limiter pattern from HTTP response headers"
  - "Bexio API client with retry, backoff, and token refresh"
  - "Upsert with ON CONFLICT DO UPDATE for idempotent sync operations"

requirements-completed: []

duration: 15min
completed: 2026-02-26
---

# Phase 18 Plan 01: Bexio Data Foundation Summary

**Built database schema, API client with OAuth2/rate-limiting, and repository layer for Bexio integration.**

## Performance
- **Duration:** 15 min
- **Tasks:** 3
- **Files created:** 14
- **Files modified:** 0

## Accomplishments
- Migration 000055 creates 4 Bexio-specific tables (sync_configs, entity_mappings, field_mappings, sync_log) and extends integration_configs CHECK constraint to include 'bexio'
- Custom Bexio REST API v2.0 client with exponential backoff (1s/2s/4s, max 3 retries), adaptive rate limiting from X-RateLimit-* headers, and automatic OAuth2 token refresh
- TokenManager with vault-backed refresh token storage and in-memory access token caching with 30s safety margin
- Repository interface with PostgresRepository implementing all CRUD + upsert operations using parameterized queries
- Domain models: 6 KMU Hub persistence types (BexioSyncConfig, BexioEntityMapping, BexioFieldMapping, BexioSyncLog, BexioSyncStatus, BexioFieldMappingEntry) and 8 Bexio API types (BexioContact, BexioInvoice, BexioQuote, BexioPaymentStatus, etc.)
- Resource methods for contacts (list/get/create/update with delta sync), invoices (create/update/get/payments), and quotes (create/update/get)

## Task Commits
1. **Task 1-3: Data Foundation** - `4e4bb42` (feat)

## Decisions Made
- VaultService interface matches vault.Service exactly for direct implementation without adapter
- Token cache uses 30-second safety margin to prevent using tokens during refresh window
- Rate limiter starts with default 50 requests/10 seconds, dynamically adjusts from response headers
- UpdateLastSyncTime uses switch statement for column name (not string interpolation) for SQL safety

## Deviations from Plan
None — all three tasks (migration + models, API client, repository) implemented as specified.

## Issues Encountered
None

## Self-Check: PASSED
- ✅ Migration 000055 up/down files exist (2 files)
- ✅ No fmt.Println in bexio package
- ✅ api.bexio.com/2.0 in client config
- ✅ bexio_oauth_refresh_ in auth.go
- ✅ X-RateLimit in rate_limiter.go
- ✅ BexioSyncConfig in models/bexio.go
- ✅ Repository interface in repository.go
- ✅ go build ./internal/models/... compiles
- ✅ go build ./internal/biz/bexio/... compiles
- ✅ All SQL uses parameterized queries ($1, $2, ...)
- ✅ Rate limiter is thread-safe (sync.Mutex)

---
*Phase: 18-bexio-integration*
*Completed: 2026-02-26*
