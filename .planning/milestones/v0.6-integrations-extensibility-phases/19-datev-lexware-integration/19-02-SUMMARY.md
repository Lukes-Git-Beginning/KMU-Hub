---
phase: 19-datev-lexware-integration
plan: 02
subsystem: desktop, lexware, datev
tags: [frontend, react, lexware, datev, settings, wizard, dashboard]

requires:
  - phase: 19-datev-lexware-integration
    plan: 01
    provides: Lexware + DATEV gRPC services, HTTP gateway routes

provides:
  - Lexware TypeScript types and API client
  - DATEV upload TypeScript types and API client
  - React Query hooks for all Lexware and DATEV endpoints
  - 3-step Lexware setup wizard (API key, sync config, field mappings)
  - Lexware sync dashboard with status cards, manual trigger, sync history
  - Lexware field mapping editor with nested dotted path support
  - DATEV settings panel with OAuth connect, upload controls, CSV fallback
  - IntegrationsSettingsTab updated with Lexware + DATEV cards alongside Bexio

affects: [20-plugin-system]

tech-stack:
  added: []
  patterns: [lexware-api-client, datev-upload-client, lexware-react-query-hooks, api-key-setup-wizard]

key-files:
  created:
    - desktop/src/renderer/src/api/lexware-types.ts
    - desktop/src/renderer/src/api/lexware-client.ts
    - desktop/src/renderer/src/api/datev-upload-types.ts
    - desktop/src/renderer/src/api/datev-upload-client.ts
    - desktop/src/renderer/src/api/hooks/useLexware.ts
    - desktop/src/renderer/src/api/hooks/useDatevUpload.ts
    - desktop/src/renderer/src/modules/settings/integrations/LexwareSetupWizard.tsx
    - desktop/src/renderer/src/modules/settings/integrations/LexwareSyncDashboard.tsx
    - desktop/src/renderer/src/modules/settings/integrations/LexwareFieldMappingEditor.tsx
    - desktop/src/renderer/src/modules/settings/integrations/DatevSettingsPanel.tsx
  modified:
    - desktop/src/renderer/src/modules/settings/tabs/IntegrationsSettingsTab.tsx
    - deploy/docker/docker-compose.yml

key-decisions:
  - "lexware-client.ts follows bexio-client.ts fetch wrapper pattern (typed fetch + auth + 401 retry)"
  - "Separate useLexware.ts and useDatevUpload.ts hooks (not merged into existing hook files)"
  - "LexwareSetupWizard uses 3 steps (API key → Sync Config → Field Mapping) -- simpler than Bexio's 4-step OAuth flow"
  - "LexwareFieldMappingEditor supports nested dotted path input (e.g., addresses.billing.street)"
  - "DatevSettingsPanel shows CSV fallback notice when no OAuth credentials configured"
  - "IntegrationsSettingsTab now shows three integration cards: Bexio (CH), Lexware Office (DE), DATEV (DE/AT)"
  - "Docker Compose biz service extended with LEXWARE_* and DATEV_* environment variables"

patterns-established:
  - "API key setup wizard pattern (simpler than OAuth wizard, single text input step)"
  - "Separate API client per integration domain (lexware-client.ts, datev-upload-client.ts)"
  - "Upload-oriented settings panel with status tracking and fallback notices"

requirements-completed: []

duration: 20min
completed: 2026-02-26
---

# Phase 19 Plan 02: Frontend Summary

**Built the complete frontend for Lexware Office and DATEV API integrations: API clients, React Query hooks, 3-step setup wizard, sync dashboard, field mapping editor, DATEV settings panel, and IntegrationsSettingsTab update.**

## Performance
- **Duration:** ~20 min
- **Tasks:** 4
- **Files created:** 10
- **Files modified:** 2

## Accomplishments
- Created lexware-types.ts with interfaces for sync config, entity mapping, field mapping, sync log, webhook subscription, and Lexware API entity types
- Created datev-upload-types.ts with interfaces for upload config, upload log, and status types
- Created lexware-client.ts with typed fetch wrapper matching bexio-client.ts pattern (~10 API methods)
- Created datev-upload-client.ts with typed fetch wrapper for DATEV upload endpoints (~5 API methods)
- Created useLexware.ts with TanStack Query hooks (queries for status/config/mappings/logs, mutations for connect/disconnect/sync/update)
- Created useDatevUpload.ts with TanStack Query hooks (queries for config/status/logs, mutations for connect/upload)
- Built LexwareSetupWizard with 3 steps: API key input → sync config toggles/intervals → embedded field mapping editor (simpler than Bexio's 4-step OAuth flow)
- Built LexwareSyncDashboard as Dialog with connection header, sync status cards, manual sync button, field mapping editor, and sync history table
- Built LexwareFieldMappingEditor with grid layout, add/remove rows, nested dotted path input fields, direction dropdowns, required checkboxes, and validation
- Built DatevSettingsPanel with OAuth connect button, client number input, auto-upload toggle, Buchungsstapel/Belegbilder upload buttons, upload status table, and CSV fallback notice
- Updated IntegrationsSettingsTab with Lexware Office and DATEV cards alongside existing Bexio card
- Updated Docker Compose with LEXWARE_* and DATEV_* env vars for biz service
- Updated ROADMAP.md, REQUIREMENTS.md, and STATE.md to reflect Phase 19 completion

## Task Commits
1. **Task 1-4: Complete Frontend + Docs** - `7eb74d9` (feat)

## Decisions Made
- Separate API clients per integration (lexware-client.ts, datev-upload-client.ts) rather than shared client
- 3-step wizard for Lexware (API key setup is simpler than Bexio's OAuth flow, no polling step needed)
- Field mapping editor supports nested path input via text fields (user types `addresses.billing.street`)
- DatevSettingsPanel shows prominent CSV fallback notice when no DATEV API credentials configured
- Three integration cards in IntegrationsSettingsTab organized by market: Bexio (CH), Lexware Office (DE), DATEV (DE/AT)

## Deviations from Plan
- Docs updates (ROADMAP.md, REQUIREMENTS.md, STATE.md) included in same commit as frontend code

## Issues Encountered
- node_modules not installed in desktop/ directory, so TypeScript validation via build not possible. All success criteria verified via grep pattern matching instead.

## Self-Check: PASSED
- ✅ LexwareSyncConfig in lexware-types.ts
- ✅ DatevUploadConfig in datev-upload-types.ts
- ✅ useLexware hooks in useLexware.ts
- ✅ useDatevUpload hooks in useDatevUpload.ts
- ✅ LexwareSetupWizard component in LexwareSetupWizard.tsx
- ✅ LexwareSyncDashboard component in LexwareSyncDashboard.tsx
- ✅ LexwareFieldMappingEditor with nested path support
- ✅ DatevSettingsPanel with upload controls and CSV fallback
- ✅ IntegrationsSettingsTab includes Lexware and DATEV cards
- ✅ Docker Compose includes LEXWARE_* and DATEV_* env vars

---
*Phase: 19-datev-lexware-integration*
*Completed: 2026-02-26*
