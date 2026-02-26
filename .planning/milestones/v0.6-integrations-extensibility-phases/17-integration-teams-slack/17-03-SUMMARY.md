---
phase: 17-integration-teams-slack
plan: 03
subsystem: desktop, settings, integration
tags: [teams, slack, react, tanstack-query, tailwind, wizard, channel-mapping, account-linking]

# Dependency graph
requires:
  - phase: 17-integration-teams-slack
    plan: 02
    provides: 18 HTTP endpoints for integration config, channel mappings, account linking, webhooks
provides:
  - TypeScript types for integration entities (IntegrationConfig, ChannelMapping, AccountLinkStatus)
  - TanStack Query hooks (14 hooks) for all integration API endpoints with optimistic updates
  - IntegrationCard reusable component (designed for Phase 18-19 Bexio/Abacus reuse)
  - TeamsSetupWizard (4-step Azure AD credential + channel mapping wizard)
  - SlackSetupWizard (4-step manual/OAuth credential + channel mapping wizard)
  - ChannelMappingEditor with colored module chips and inline CRUD
  - AccountLinkDialog with token-based verification flow
  - IntegrationsSettingsTab wired into SettingsPage for admin users
affects: [18-bexio, 19-abacus-rma]

# Tech tracking
tech-stack:
  added: []
  patterns: [reusable IntegrationCard component for multi-platform settings, wizard dialog pattern for multi-step configuration, colored module chip toggles for channel mapping]

key-files:
  created:
    - desktop/src/renderer/src/api/integration-types.ts
    - desktop/src/renderer/src/api/integration-client.ts
    - desktop/src/renderer/src/api/hooks/useIntegration.ts
    - desktop/src/renderer/src/modules/settings/integrations/IntegrationCard.tsx
    - desktop/src/renderer/src/modules/settings/integrations/TeamsSetupWizard.tsx
    - desktop/src/renderer/src/modules/settings/integrations/SlackSetupWizard.tsx
    - desktop/src/renderer/src/modules/settings/integrations/ChannelMappingEditor.tsx
    - desktop/src/renderer/src/modules/settings/integrations/AccountLinkDialog.tsx
    - desktop/src/renderer/src/modules/settings/tabs/IntegrationsSettingsTab.tsx
  modified:
    - desktop/src/renderer/src/modules/settings/SettingsPage.tsx
    - desktop/src/renderer/src/i18n/messages/de.json
    - desktop/src/renderer/src/i18n/messages/en.json

key-decisions:
  - "IntegrationCard accepts generic props (name, icon, status, callbacks) with no Teams/Slack-specific logic for Phase 18-19 reuse"
  - "Wizard state managed via useState (transient, not Zustand) since wizard is a dialog that resets on close"
  - "ChannelMappingEditor is a shared component used by both TeamsSetupWizard and SlackSetupWizard"
  - "integration-client.ts follows caldav-client.ts pattern with typed fetch helper, auth injection, and 401 retry"
  - "Optimistic updates for config enable/disable toggle and channel mapping is_active toggle"
  - "Slack wizard offers dual credential modes: manual (bot token + signing secret) or OAuth install in new window"
  - "Future platform placeholders (Bexio, Abacus) shown as IntegrationCard with comingSoon badge"

patterns-established:
  - "IntegrationCard reusable pattern: generic platform card with status badge, configure button, and toggle"
  - "4-step wizard dialog pattern: step indicator dots, back/forward navigation, transient useState"
  - "Colored module chips: INTEGRATION_MODULES constant with id/label/color used by ChannelMappingEditor toggles"

requirements-completed: [INT-04, INT-05, INT-06]

# Metrics
duration: 8min
completed: 2026-02-25
---

# Phase 17 Plan 03: Frontend Integration Settings Summary

**Complete frontend UI for Teams & Slack integration: admin settings tab with platform cards, 4-step setup wizards, channel mapping editor with colored module chips, and token-based account linking dialog**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-25
- **Completed:** 2026-02-25
- **Tasks:** 2
- **Files created:** 9
- **Files modified:** 3

## Accomplishments
- Created TypeScript types (integration-types.ts) and API client (integration-client.ts) matching all 18 backend HTTP endpoints from Plan 17-02
- Built 14 TanStack Query hooks with query key factory, proper cache invalidation, and optimistic updates for toggle operations
- Designed IntegrationCard as a reusable, platform-agnostic component accepting generic props (name, icon, status, callbacks) for Phase 18-19 reuse with Bexio and Abacus
- Implemented TeamsSetupWizard (4-step: Azure AD credentials, channel mapping, test notification, activation) and SlackSetupWizard (4-step: manual/OAuth credentials, channel mapping, test, activation)
- Built ChannelMappingEditor with colored module chip toggles from INTEGRATION_MODULES constant, inline create/edit forms, delete confirmation, and optimistic is_active toggle
- Created AccountLinkDialog with token-based verification flow (/kmuhub link command instructions, token input, success auto-close)
- Wired IntegrationsSettingsTab into SettingsPage with admin-only visibility, platform cards grid, future platform placeholders, and account linking section for all users
- Added i18n strings for German and English locale files

## Task Commits

Each task was committed atomically:

1. **Task 1: TypeScript types + API client + TanStack Query hooks** - `300dcab` (feat)
2. **Task 2: UI components + SettingsPage integration + i18n** - `c0eac22` (feat)

## Files Created/Modified
- `desktop/src/renderer/src/api/integration-types.ts` - TypeScript types for IntegrationConfig, ChannelMapping, AccountLinkStatus, INTEGRATION_MODULES
- `desktop/src/renderer/src/api/integration-client.ts` - Typed fetch client for 13 integration API endpoints with auth injection
- `desktop/src/renderer/src/api/hooks/useIntegration.ts` - 14 TanStack Query hooks with query key factory and optimistic updates
- `desktop/src/renderer/src/modules/settings/integrations/IntegrationCard.tsx` - Reusable platform card with status badge, configure button, toggle switch
- `desktop/src/renderer/src/modules/settings/integrations/TeamsSetupWizard.tsx` - 4-step Teams configuration wizard dialog
- `desktop/src/renderer/src/modules/settings/integrations/SlackSetupWizard.tsx` - 4-step Slack configuration wizard with manual/OAuth modes
- `desktop/src/renderer/src/modules/settings/integrations/ChannelMappingEditor.tsx` - Module-to-channel mapping with colored chip toggles
- `desktop/src/renderer/src/modules/settings/integrations/AccountLinkDialog.tsx` - Token-based account linking with auto-close on success
- `desktop/src/renderer/src/modules/settings/tabs/IntegrationsSettingsTab.tsx` - Admin tab with platform cards grid and account linking section
- `desktop/src/renderer/src/modules/settings/SettingsPage.tsx` - Added Integrations tab (TabKey, TABS entry, rendering branch, Plug icon import)
- `desktop/src/renderer/src/i18n/messages/de.json` - Added settings.integrations.title and settings.integrations.subtitle
- `desktop/src/renderer/src/i18n/messages/en.json` - Added settings.integrations.title and settings.integrations.subtitle

## Decisions Made
- IntegrationCard is platform-agnostic: accepts name, icon (React node), description, status, callbacks. No Teams/Slack logic inside the component. Future phases (Bexio, Abacus) can reuse it directly.
- Wizard state uses transient useState, not Zustand, because wizard state is scoped to the dialog lifecycle and resets on close.
- ChannelMappingEditor is extracted as a shared component used by both wizard step 2 implementations to avoid code duplication.
- SlackSetupWizard offers dual credential modes: manual entry (bot token + signing secret + optional client ID/secret) and OAuth install (opens /api/v1/integrations/slack/oauth/install in new window).
- Future platform cards (Bexio, Abacus) shown with comingSoon prop as disabled placeholders in the grid layout.
- Account linking section is visible to all users (not admin-only) since any user can link their external account.

## Deviations from Plan

### Added Files

**1. [Enhancement] Created integration-client.ts API client**
- **Reason:** Following established codebase pattern (automation-client.ts, caldav-client.ts) where hooks call a dedicated client module rather than making fetch calls inline
- **Files added:** desktop/src/renderer/src/api/integration-client.ts
- **Impact:** Cleaner separation; hooks delegate to typed client functions

**2. [Enhancement] Added i18n strings for both locales**
- **Reason:** SettingsPage uses FormattedMessage with labelId, so the new tab needs corresponding i18n entries
- **Files modified:** de.json, en.json
- **Impact:** Tab label shows "Integrationen" in German and "Integrations" in English

---

**Total deviations:** 2 enhancements (0 blocking)
**Impact on plan:** No scope reduction. Added API client layer and i18n entries for completeness.

## Issues Encountered
- None

## User Setup Required
None - the Integrations tab appears automatically for admin users in Settings. Integration setup requires admin to configure credentials via the wizard dialogs.

## Next Phase Readiness
- Phase 17 (Teams & Slack Integration) is now complete across all 3 plans: database/proto (17-01), backend engine/adapters/gateway (17-02), frontend UI (17-03)
- IntegrationCard component is ready for reuse in Phase 18 (Bexio) and Phase 19 (Abacus/RmA)
- INTEGRATION_MODULES constant can be extended as new modules are added

## Self-Check: PASSED

All 9 created files and 3 modified files verified on disk. Both task commits (300dcab, c0eac22) verified in git log.

---
*Phase: 17-integration-teams-slack*
*Completed: 2026-02-25*
