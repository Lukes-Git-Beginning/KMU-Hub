---
phase: 19-datev-lexware-integration
plan: 01
subsystem: biz, lexware, datev
tags: [integration, api-key-auth, oauth2, lexware, datev, webhooks, field-mapping]

requires:
  - phase: 18-bexio-integration
    plan: 04
    provides: Phase 18 complete, Bexio integration available, biz binary with integration patterns
provides:
  - Lexware Office REST API client with API key auth and 2 req/s rate limiting
  - Bidirectional contact sync with configurable field mapping (nested path support)
  - Invoice and quote push to Lexware Office
  - Webhook handler for real-time Lexware event notifications
  - DATEV API upload for Buchungsstapel and Belegbilder via OAuth2
  - Database schema for Lexware sync tracking and DATEV upload logging
  - Proto definitions and gRPC servers for both services
  - Gateway HTTP routes for all Lexware and DATEV endpoints
  - Biz binary wiring with optional initialization
affects: [19-02]

tech-stack:
  added: []
  patterns: [api-key-auth, webhook-handler, nested-field-mapper, datev-oauth2-upload, graceful-csv-fallback]

key-files:
  created:
    - backend/migrations/000056_add_lexware_datev_api.up.sql
    - backend/migrations/000056_add_lexware_datev_api.down.sql
    - backend/internal/models/lexware.go
    - backend/internal/models/datev_upload.go
    - backend/internal/biz/lexware/models.go
    - backend/internal/biz/lexware/errors.go
    - backend/internal/biz/lexware/types.go
    - backend/internal/biz/lexware/auth.go
    - backend/internal/biz/lexware/rate_limiter.go
    - backend/internal/biz/lexware/client.go
    - backend/internal/biz/lexware/contacts.go
    - backend/internal/biz/lexware/invoices.go
    - backend/internal/biz/lexware/quotes.go
    - backend/internal/biz/lexware/contact_sync.go
    - backend/internal/biz/lexware/invoice_push.go
    - backend/internal/biz/lexware/quote_push.go
    - backend/internal/biz/lexware/field_mapper.go
    - backend/internal/biz/lexware/repository.go
    - backend/internal/biz/lexware/postgres_repository.go
    - backend/internal/biz/lexware/postgres_config_repo.go
    - backend/internal/biz/lexware/service.go
    - backend/internal/biz/lexware/scheduler.go
    - backend/internal/biz/lexware/webhook_handler.go
    - backend/internal/biz/lexware/event_emitter.go
    - backend/internal/biz/datev/oauth.go
    - backend/internal/biz/datev/uploader.go
    - backend/internal/biz/datev/belegbilder.go
    - backend/internal/biz/datev/upload_service.go
    - backend/internal/biz/datev/upload_repository.go
    - backend/internal/biz/datev/postgres_config_repo.go
    - backend/internal/biz/datev/postgres_upload_repo.go
    - backend/proto/biz/v1/lexware.proto
    - backend/proto/biz/v1/datev_upload.proto
    - backend/internal/server/lexware_grpc.go
    - backend/internal/server/datev_upload_grpc.go
    - backend/internal/gateway/route_lexware.go
    - backend/internal/gateway/route_datev_upload.go
  modified:
    - backend/internal/config/config.go
    - backend/cmd/biz/main.go

key-decisions:
  - "Lexware uses API key auth (no OAuth2) -- simpler than Bexio, key stored in integration_configs"
  - "Lexware entity IDs are VARCHAR(36) UUIDs, not integers like Bexio"
  - "lexware_version column enables optimistic locking for concurrent sync protection"
  - "Webhooks over polling -- Lexware supports webhooks, eliminating background polling need"
  - "Field mapper supports dotted/nested path resolution (e.g., addresses.billing[0].street)"
  - "2 req/s rate limiter -- stricter than Bexio (50/10s) per Lexware API limits"
  - "DATEV as optional overlay -- extends Phase 12 CSV export with API upload when credentials available"
  - "Separate lexware/ and datev/ packages -- no shared abstraction, different auth models"
  - "credit_note entity type added to Lexware entity_type CHECK (Lexware supports credit notes natively)"
  - "lexware_webhook_subscriptions table tracks active subscriptions for lifecycle management"
  - "Both services optional in biz binary -- only initialized when config env vars present"

patterns-established:
  - "API key auth pattern for integration services (simpler alternative to OAuth2)"
  - "Webhook-based sync over polling for supported integrations"
  - "Nested field path resolution for complex JSON API responses"
  - "Upload-only DATEV pattern with graceful CSV fallback"

requirements-completed: []

duration: 30min
completed: 2026-02-26
---

# Phase 19 Plan 01: Backend Summary

**Built the complete backend for Lexware Office and DATEV API integrations: migration, API clients, sync engine, webhook handler, upload service, proto, gRPC, gateway, and biz binary wiring.**

## Performance
- **Duration:** ~30 min
- **Tasks:** 6
- **Files created:** 37
- **Files modified:** 2

## Accomplishments
- Migration 000056 creates 7 tables (5 Lexware: sync_configs, entity_mappings, field_mappings, sync_log, webhook_subscriptions; 2 DATEV: upload_configs, upload_log) and extends integration_configs CHECK constraint for 'lexware' and 'datev_api'
- Custom Lexware Office REST API v1 client with API key authentication and 2 req/s rate limiter (stricter than Bexio's 50/10s)
- Bidirectional contact sync with last-write-wins conflict resolution, configurable field mappings with nested JSON path support
- Invoice and quote push to Lexware Office with optimistic locking via lexware_version
- Webhook handler for real-time Lexware event notifications (replacing polling)
- DATEV OAuth2 token management with Buchungsstapel and Belegbilder upload endpoints
- DATEV gracefully falls back to manual CSV export when no API credentials configured
- Proto definitions for LexwareService and DatevUploadService
- gRPC servers (LexwareGRPCServer, DatevUploadGRPCServer) registered on biz binary port :50058
- Gateway HTTP routes: ~10 Lexware endpoints (connect, disconnect, sync, mappings, webhook) + ~5 DATEV endpoints (OAuth, upload, status)
- Domain models: LexwareSyncConfig, LexwareEntityMapping, LexwareFieldMapping, LexwareSyncLog, LexwareWebhookSubscription, DatevUploadConfig, DatevUploadLog
- Lexware API types with nested JSON structures matching Lexware Office API responses

## Task Commits
1. **Task 1-6: Complete Backend** - `7eb74d9` (feat)

## Decisions Made
- API key auth for Lexware is simpler than Bexio's OAuth2 -- no token refresh, no callback, key stored directly
- VARCHAR(36) for Lexware IDs because Lexware uses UUID strings (not integers like Bexio)
- Webhooks instead of polling for Lexware (unlike Bexio which doesn't support webhooks)
- Separate lexware/ and datev/ packages (no shared abstraction with bexio/) because auth models differ fundamentally
- credit_note added as entity type for Lexware (natively supported, unlike Bexio)
- Both services optional in biz binary: Lexware initialized when LEXWARE_API_URL is set, DATEV when DATEV_CLIENT_ID is set

## Deviations from Plan
None — all six tasks (migration+models, Lexware client+sync, DATEV upload, proto+gRPC, gateway, biz wiring) implemented as specified.

## Issues Encountered
None

## Self-Check: PASSED
- ✅ Migration 000056 up/down files exist (2 files)
- ✅ No fmt.Println in lexware or datev packages
- ✅ api.lexoffice.io/v1 in Lexware client config
- ✅ LexwareSyncConfig in models/lexware.go
- ✅ DatevUploadConfig in models/datev_upload.go
- ✅ FieldMapper in lexware/field_mapper.go
- ✅ WebhookHandler in lexware/webhook_handler.go
- ✅ LexwareService in lexware.proto
- ✅ DatevUploadService in datev_upload.proto
- ✅ LexwareGRPCServer in server/lexware_grpc.go
- ✅ DatevUploadGRPCServer in server/datev_upload_grpc.go
- ✅ LexwareRoutes and DatevUploadRoutes in gateway routes
- ✅ All SQL uses parameterized queries

---
*Phase: 19-datev-lexware-integration*
*Completed: 2026-02-26*
