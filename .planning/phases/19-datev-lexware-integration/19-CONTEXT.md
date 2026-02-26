# Phase 19: DATEV API + Lexware Office - Context

**Gathered:** 2026-02-26
**Status:** Ready for planning

<domain>
## Phase Boundary

German SMBs can connect their KMU Hub instance to Lexware Office (300k+ customers) via API key authentication. Contacts sync bidirectionally with configurable field mapping and last-write-wins conflict resolution. Invoices and quotes created in KMU Hub are pushed to Lexware Office. Lexware Office webhooks trigger real-time sync updates instead of polling. Separately, DATEV API integration (2.5M+ companies via Steuerberater) enables Buchungsstapel (CSV) and Belegbilder (PDF) upload via OAuth2. DATEV gracefully falls back to manual CSV export when no API credentials are configured. Together with Bexio (CH), this completes full DACH accounting coverage: Bexio CH + Lexware DE + DATEV DE/AT.

</domain>

<decisions>
## Implementation Decisions

### Architecture: Biz Service Extension
- Lexware client lives in `internal/biz/lexware/` package (no new microservice)
- DATEV upload extension lives in `internal/biz/datev/` package
- Both share gRPC port :50058 with Finance, HR, and Bexio services
- Reuses existing gateway integration routes pattern from Phases 17-18

### Lexware API Key Authentication (No OAuth)
- Lexware Office uses simple API key authentication (not OAuth2)
- API key stored in integration_configs via vault
- No token refresh, no callback flow — simpler than Bexio
- Base URL: `https://api.lexoffice.io/v1/`

### Webhooks over Polling
- Lexware Office supports webhooks for real-time event notification
- Webhook subscriptions tracked in `lexware_webhook_subscriptions` table
- Webhook handler validates Lexware signatures
- Eliminates need for background polling (unlike Bexio)

### VARCHAR(36) for Lexware IDs (Not INTEGER)
- Lexware uses UUID strings for entity IDs, not integers like Bexio
- `lexware_entity_mappings.lexware_id` is VARCHAR(36)

### lexware_version for Optimistic Locking
- Lexware API returns version numbers on entities
- Version stored in entity mapping for conflict detection on updates
- Prevents lost-update scenarios during concurrent sync

### DATEV as Optional Overlay
- DATEV upload extends existing DATEV CSV export (Phase 12)
- OAuth2 flow for DATEV API authentication
- Buchungsdatenservice for Buchungsstapel upload
- Belegbilder endpoint for document/receipt upload
- Falls back gracefully to manual CSV export when no API credentials

### Rate Limiting: 2 req/s for Lexware
- Lexware API rate limit: 2 requests per second
- Custom rate limiter (stricter than Bexio's 50/10s)

### Separate Packages (No Shared Code with Bexio)
- Lexware and DATEV are separate packages from Bexio
- Different auth models, different API patterns, different field formats
- No shared abstraction layer — explicit over implicit

### No Lookup Cache
- Lexware API responses include denormalized data (names, not IDs)
- No need for lookup table caching (unlike Bexio's ID-based references)

### Field Mapping with Nested Paths
- Lexware Office uses nested JSON structures (e.g., `addresses.billing[0].street`)
- Field mapper supports dotted path resolution for nested fields

### Claude's Discretion
- Internal package structure within `internal/biz/lexware/` and `internal/biz/datev/`
- Exact webhook signature validation implementation
- Sync batch size for initial full sync
- Retry strategy for transient API errors
- Error message wording in sync logs
- Lexware event types for webhook subscriptions

</decisions>

<specifics>
## Specific Ideas

### Endpoints
- Lexware OAuth connect: `POST /api/v1/integrations/lexware/connect` (API key setup)
- Lexware disconnect: `DELETE /api/v1/integrations/lexware/disconnect`
- Lexware sync status: `GET /api/v1/integrations/lexware/sync/status`
- Lexware manual sync: `POST /api/v1/integrations/lexware/sync/trigger`
- Lexware field mappings: `GET/PUT /api/v1/integrations/lexware/mappings/{entity_type}`
- Lexware webhooks: `POST /api/v1/integrations/lexware/webhooks` (Lexware callback)
- DATEV OAuth callback: `GET /api/v1/integrations/datev/oauth/callback`
- DATEV upload Buchungsstapel: `POST /api/v1/integrations/datev/upload/buchungsstapel`
- DATEV upload Belegbilder: `POST /api/v1/integrations/datev/upload/belegbilder`
- DATEV upload status: `GET /api/v1/integrations/datev/upload/status`

### Tables (Migration 000056)
- `lexware_sync_configs` — per-tenant sync configuration
- `lexware_entity_mappings` — KMU Hub ID <-> Lexware UUID string pairs
- `lexware_field_mappings` — configurable field mapping per entity type
- `lexware_sync_log` — sync operation audit trail
- `lexware_webhook_subscriptions` — tracked webhook subscriptions
- `datev_upload_configs` — per-tenant DATEV API configuration
- `datev_upload_log` — upload audit trail

### Frontend Components
- LexwareSetupWizard (3-step: API key → sync config → field mappings)
- LexwareSyncDashboard (sync status, manual trigger, sync history)
- LexwareFieldMappingEditor (nested path support)
- DatevSettingsPanel (OAuth connect, upload controls, status)
- IntegrationsSettingsTab update (Lexware + DATEV cards)

</specifics>

<deferred>
## Deferred Ideas

- Lexware article/product sync between systems
- Multi-organization support for Lexware
- DATEV bank statement import via API
- Abacus integration (CH-only, no significant market demand beyond Bexio)
- DATEV Buchungsdatenservice v2 (async job-based upload)
- Lexware Office credit note sync (push only in v1)
- Real-time DATEV upload status polling

</deferred>

---

*Phase: 19-datev-lexware-integration*
*Context gathered: 2026-02-26*
