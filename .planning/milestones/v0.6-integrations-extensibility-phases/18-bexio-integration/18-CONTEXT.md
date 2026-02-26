# Phase 18: Bexio Integration - Context

**Gathered:** 2026-02-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Swiss SMBs can connect their KMU Hub instance to Bexio accounting software via OAuth2. Contacts sync bidirectionally between CRM and Bexio with configurable field mapping and last-write-wins conflict resolution. Invoices and quotes created in KMU Hub are pushed to Bexio, and payment status is synced back via polling. An admin UI provides a setup wizard, field mapping editor, and sync dashboard with manual trigger capability.

</domain>

<decisions>
## Implementation Decisions

### Architecture: Biz Service Extension
- Bexio client lives in `internal/biz/bexio/` package (no new microservice)
- Shares gRPC port :50058 with Finance and HR services
- Reuses existing gateway integration routes pattern from Phase 17
- Reuses vault for OAuth credential storage (same pattern as Teams/Slack)

### OAuth2 via Keycloak (OIDC)
- Authorization endpoint: `https://auth.bexio.com/realms/bexio/protocol/openid-connect/auth`
- Token endpoint: `https://auth.bexio.com/realms/bexio/protocol/openid-connect/token`
- Scopes: `openid profile offline_access contact_edit kb_invoice_edit kb_offer_edit`
- Refresh token stored in vault with key `bexio_oauth_refresh_{tenant_id}`
- Access token cached in-memory with TTL, auto-refreshed before expiry
- Client ID and Secret stored as vault secrets

### Custom Bexio API Client (No SDK)
- No usable Go SDK exists -- build own minimal client
- Base URL: `https://api.bexio.com/2.0/`
- Built-in rate limiter respecting response headers (X-RateLimit-*)
- Exponential backoff on 429 (1s, 2s, 4s, max 3 retries)
- Known API quirks handled: POST for updates (not PATCH), Content-Length:0 for GET, salutation_id 0->null

### Bidirectional Contact Sync
- Last-Write-Wins conflict resolution via `updated_at` timestamps
- Polling interval: 15 minutes (configurable)
- Delta-sync via Bexio `updated_at` filter parameter
- Mapping table tracks KMU Hub contact_id <-> Bexio contact_id
- Initial full sync on first connection, then delta only
- Contact type mapping: Bexio "company" -> KMU Hub Company, Bexio "person" -> KMU Hub Contact

### Invoice + Quote Push to Bexio
- KMU Hub invoices pushed to Bexio on status change to "sent"
- KMU Hub quotes pushed to Bexio on creation/update
- Payment status polled from Bexio every 5 minutes
- Bexio invoice/quote ID stored in KMU Hub for reference tracking
- Line items, tax breakdown, and contact reference mapped

### Configurable Field Mapping
- Admin can configure field-to-field mapping per entity type (contact, invoice, quote)
- Default mapping provided out-of-the-box (email->email, name->name, etc.)
- Custom fields can be mapped to Bexio `remarks` or other text fields
- Mapping stored in JSONB column on sync config table
- Validation: required fields enforced, type compatibility checked

### Sync Mode: Auto + Manual
- Background polling scheduler (goroutine with configurable intervals)
- Manual "Sync Now" button in admin UI triggers immediate sync
- Sync status tracked per entity type (last_sync_at, items_synced, errors)
- Sync log for debugging (last N sync runs with success/error counts)

### No Webhooks (Bexio Limitation)
- Bexio does not support webhooks
- All inbound sync via polling with delta detection
- Outbound sync triggered by KMU Hub events (pg_notify)

### Claude's Discretion
- Internal package structure within `internal/biz/bexio/`
- Exact rate limiter implementation (token bucket vs sliding window)
- Sync batch size for initial full sync
- Retry strategy details for transient API errors
- Migration numbering (next available after current max)
- Bexio lookup table caching strategy (countries, currencies, salutations)
- Error message wording in sync logs

</decisions>

<specifics>
## Specific Ideas

- OAuth callback URL: `/api/v1/integrations/bexio/oauth/callback`
- Sync status endpoint: `GET /api/v1/integrations/bexio/sync/status`
- Manual sync trigger: `POST /api/v1/integrations/bexio/sync/trigger`
- Field mapping CRUD: `GET/PUT /api/v1/integrations/bexio/mappings/{entity_type}`
- Reuse Phase 17's `integration_configs` table with platform="bexio"
- New tables: `bexio_sync_mappings` (entity ID pairs), `bexio_field_mappings` (field config), `bexio_sync_log` (audit)
- Frontend: Bexio card in Integrations settings (same pattern as Teams/Slack cards)
- Setup wizard: 1. OAuth connect -> 2. Select sync directions -> 3. Configure field mappings -> 4. Initial sync
- Sync dashboard: Last sync time, items synced, error count, sync history table
- Bexio lookup tables cached at sync start (countries, currencies, salutations) to resolve ID-based references

</specifics>

<deferred>
## Deferred Ideas

- Bexio -> KMU Hub invoice/quote import (only push in v1)
- Article/product sync between systems
- Project sync (Bexio projects <-> KMU Hub projects)
- Dunning/reminder sync to Bexio
- Multi-company support (multiple Bexio orgs)
- Bexio sandbox/test mode (no sandbox available from Bexio)
- Real-time sync via polling < 1 minute (unnecessary for SMB use case)

</deferred>

---

*Phase: 18-bexio-integration*
*Context gathered: 2026-02-26*
