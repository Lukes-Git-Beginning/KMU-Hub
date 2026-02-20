---
phase: 11-documents-files-wopi-onlyoffice
plan: 06
subsystem: ui
tags: [global-search, onlyoffice, wopi, tanstack-query, react, iframe, debounce]

# Dependency graph
requires:
  - phase: 11-04
    provides: "Gateway routes, WOPI endpoints, global search API, Docker Compose"
provides:
  - "useGlobalSearch TanStack Query hook for global search API"
  - "Refactored SearchBar with API-driven grouped results"
  - "OnlyOfficeEditor iframe wrapper component with WOPI protocol"
  - "isOnlyOfficeEditable/isOnlyOfficeEditableByExtension helpers"
  - "DokumentePage 'In OnlyOffice bearbeiten' context menu integration"
affects: [phase-14-unified-inbox, phase-12-finanzen]

# Tech tracking
tech-stack:
  added: []
  patterns: [debounced-search-hook, wopi-iframe-editor, module-grouped-search-results]

key-files:
  created:
    - desktop/src/renderer/src/api/hooks/useGlobalSearch.ts
    - desktop/src/renderer/src/modules/dokumente/OnlyOfficeEditor.tsx
  modified:
    - desktop/src/renderer/src/components/header/SearchBar.tsx
    - desktop/src/renderer/src/modules/dokumente/DokumentePage.tsx

key-decisions:
  - "Global search types defined inline in useGlobalSearch.ts (not in document-types.ts) since they are API-specific and not shared"
  - "300ms debounce with 2-char minimum prevents excessive API calls while maintaining responsive UX"
  - "Module results grouped with collapsible sections and 'Mehr anzeigen' expansion for progressive disclosure"
  - "OnlyOffice editor runs as full-screen fixed overlay (z-50) for immersive editing experience"
  - "Token expiry warning at 30 minutes before TTL to prompt user to save work"

patterns-established:
  - "useDebouncedValue: custom hook for search input debouncing (300ms default)"
  - "Module config mapping: centralized icon/color/label/path per search module"
  - "WOPI iframe URL construction: ${ONLYOFFICE_URL}/hosting/wopi/${editorType}/edit?WOPISrc=...&access_token=..."
  - "File editability check: both MIME-type and extension-based helpers exported"

requirements-completed: [DOC-09]

# Metrics
duration: 7min
completed: 2026-02-17
---

# Phase 11 Plan 06: Global Search + OnlyOffice Editor Summary

**API-driven global search overlay with module-grouped results and OnlyOffice WOPI iframe editor with MIME-type detection and token-based auth**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-02-17T18:25:58Z
- **Completed:** 2026-02-17T18:32:47Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Refactored SearchBar from static mock data to real global search API with debounced TanStack Query hook
- Results grouped by module (Kontakte, Dateien, E-Mails, Aufgaben, Nachrichten) with counts and rich previews
- Created OnlyOffice WOPI editor component with full-screen iframe overlay, token expiry warning, and PostMessage handling
- Integrated "In OnlyOffice bearbeiten" context menu for 10+ editable file types (docx, xlsx, pptx, odt, etc.)

## Task Commits

Each task was committed atomically:

1. **Task 1: Global search hook + SearchBar refactor** - `1a62e70` (feat)
2. **Task 2: OnlyOffice editor component + DokumentePage integration** - `7244dd9` (feat)

## Files Created/Modified
- `desktop/src/renderer/src/api/hooks/useGlobalSearch.ts` - TanStack Query hook with debounce, auth token injection, type-safe API response
- `desktop/src/renderer/src/components/header/SearchBar.tsx` - Full refactor: API-driven grouped results, keyboard nav, loading/error states, module filters with counts
- `desktop/src/renderer/src/modules/dokumente/OnlyOfficeEditor.tsx` - WOPI iframe editor with word/cell/slide type detection, token expiry warning, cleanup on unmount
- `desktop/src/renderer/src/modules/dokumente/DokumentePage.tsx` - Added OnlyOffice editor state, WOPI token generation, context menu "In OnlyOffice bearbeiten" for editable files

## Decisions Made
- Global search response types defined locally in useGlobalSearch.ts rather than importing from document-types.ts, since the API response shape (modules array) differs from the shape defined in 11-05's document-types.ts (modules record). This avoids a type conflict.
- useDebouncedValue implemented as custom hook rather than importing a library, keeping the dependency footprint minimal.
- OnlyOffice editor uses `import.meta.env.VITE_ONLYOFFICE_URL` and `VITE_API_URL` env vars with localhost defaults for development.
- File editability check exported as both MIME-type and extension-based helpers for flexibility (DokumentePage currently uses extension-based since Zustand mock store doesn't have MIME types).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] getToken return type mismatch**
- **Found during:** Task 1 (useGlobalSearch hook)
- **Issue:** Auth store's accessToken is `string | null`, but getToken was typed as `Promise<string | undefined>`
- **Fix:** Changed return type to `Promise<string | null>` to match auth store
- **Files modified:** desktop/src/renderer/src/api/hooks/useGlobalSearch.ts
- **Verification:** `npx tsc --noEmit` passes for useGlobalSearch.ts
- **Committed in:** 1a62e70 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Trivial type fix for TypeScript strict mode compatibility. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviation listed above.

## User Setup Required
None - OnlyOffice URL defaults to localhost:8088 (matching Docker Compose from Plan 11-04).

## Next Phase Readiness
- Phase 11 frontend is complete: global search, OnlyOffice editor, document management UI
- Global search ready for extension when email search RPC is built (Phase 14 Unified Inbox)
- OnlyOffice collaborative editing is fully wired end-to-end (WOPI token -> iframe -> protocol)
- Phase 11 complete - ready to proceed to Phase 12 (Finanzen)

## Self-Check: PASSED

All 2 created files verified present. Both task commits (1a62e70, 7244dd9) found in git log.

---
*Phase: 11-documents-files-wopi-onlyoffice*
*Completed: 2026-02-17*
