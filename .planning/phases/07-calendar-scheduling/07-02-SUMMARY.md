---
phase: 07-calendar-scheduling
plan: 02
subsystem: ui
tags: [react, tanstack-query, zustand, rrule, calendar, typescript, date-fns]

# Dependency graph
requires:
  - phase: 05-desktop-foundation
    provides: Electron app shell, routing, TanStack Query, Zustand, AppShell Sidebar
provides:
  - Calendar TypeScript types for all API entities
  - TanStack Query hooks for calendar, event, resource, holiday CRUD
  - Zustand calendar UI state store with persistence
  - CalendarLayout module shell with toolbar and nested routing
  - Calendar API fetch wrapper for pre-OpenAPI endpoints
  - Sidebar navigation entry for Kalender module
affects: [07-06-calendar-views, 07-07-event-forms, 07-08-calendar-sidebar, 07-09-resource-booking]

# Tech tracking
tech-stack:
  added: [rrule@2.8.1]
  patterns: [calendar-client fetch wrapper for pre-OpenAPI typed hooks, Set serialization in Zustand persist]

key-files:
  created:
    - desktop/src/renderer/src/api/calendar-types.ts
    - desktop/src/renderer/src/api/calendar-client.ts
    - desktop/src/renderer/src/api/hooks/useCalendars.ts
    - desktop/src/renderer/src/api/hooks/useEvents.ts
    - desktop/src/renderer/src/api/hooks/useResources.ts
    - desktop/src/renderer/src/api/hooks/useHolidays.ts
    - desktop/src/renderer/src/stores/calendar.ts
    - desktop/src/renderer/src/modules/calendar/CalendarLayout.tsx
  modified:
    - desktop/package.json
    - desktop/package-lock.json
    - desktop/src/renderer/src/App.tsx
    - desktop/src/renderer/src/components/layout/Sidebar.tsx

key-decisions:
  - "Separate calendar-types.ts instead of modifying auto-generated types.ts (openapi-typescript)"
  - "calendar-client.ts fetch wrapper mirrors openapi-fetch auth/error patterns for pre-OpenAPI hooks"
  - "Set<string> serialized as array in Zustand persist localStorage adapter"
  - "CalendarLayout uses internal Routes/Route pattern (same as WorkLayout)"

patterns-established:
  - "Calendar API client: lightweight fetch wrapper with auth injection and 401 refresh for modules without OpenAPI spec"
  - "Zustand Set persistence: custom storage adapter serializes Set as array for JSON compatibility"

# Metrics
duration: 5min
completed: 2026-02-10
---

# Phase 7 Plan 02: Calendar Frontend Foundation Summary

**rrule dependency, 30+ TypeScript interfaces, 4 hook files with TanStack Query CRUD, Zustand calendar store with date-fns navigation, CalendarLayout shell with toolbar/sidebar/routing**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-10T22:15:28Z
- **Completed:** 2026-02-10T22:20:47Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Installed rrule for client-side recurrence preview in future event editor
- Defined 30+ TypeScript interfaces covering all calendar API entities, requests, and response wrappers
- Created calendar-client.ts with auth injection, 401 refresh, and offline mutation blocking
- Built 4 hook files: useCalendars (15 hooks), useEvents (9 hooks), useResources (8 hooks), useHolidays (2 hooks)
- Created Zustand calendar store with view/date/navigation/sidebar/visibility state and localStorage persistence
- Built CalendarLayout with German-labeled toolbar (Heute/Tag/Woche/Monat), navigation arrows, collapsible sidebar shell
- Registered /calendar/* route with lazy loading in App.tsx
- Added "Kalender" with CalendarDays icon to Sidebar navigation

## Task Commits

Each task was committed atomically:

1. **Task 1: Install rrule + TypeScript types** - `a3d976a` (feat)
2. **Task 2: TanStack Query hooks + Zustand store + module shell** - `955e644` (feat)

## Files Created/Modified
- `desktop/src/renderer/src/api/calendar-types.ts` - 30+ interfaces for calendar entities, requests, responses
- `desktop/src/renderer/src/api/calendar-client.ts` - Fetch wrapper with auth injection for pre-OpenAPI endpoints
- `desktop/src/renderer/src/api/hooks/useCalendars.ts` - 15 hooks for calendar CRUD, members, subscriptions, preferences, categories
- `desktop/src/renderer/src/api/hooks/useEvents.ts` - 9 hooks for events, attendees, RSVP, reminders, recurring edits
- `desktop/src/renderer/src/api/hooks/useResources.ts` - 8 hooks for resource CRUD, availability, booking
- `desktop/src/renderer/src/api/hooks/useHolidays.ts` - 2 hooks for holiday queries and admin seed
- `desktop/src/renderer/src/stores/calendar.ts` - Zustand store with view/date/navigation/sidebar/visibility state
- `desktop/src/renderer/src/modules/calendar/CalendarLayout.tsx` - Module layout with toolbar, sidebar shell, nested routing
- `desktop/package.json` - Added rrule@2.8.1 dependency
- `desktop/src/renderer/src/App.tsx` - Registered /calendar/* route with lazy CalendarLayout
- `desktop/src/renderer/src/components/layout/Sidebar.tsx` - Added Kalender nav item with CalendarDays icon

## Decisions Made
- **Separate types file**: Created `calendar-types.ts` instead of modifying auto-generated `types.ts` since the calendar backend OpenAPI spec doesn't exist yet (built in parallel via 07-01). Types will migrate to auto-generated spec once calendar routes are in openapi.yaml.
- **Calendar fetch client**: Created `calendar-client.ts` as a lightweight fetch wrapper that mirrors openapi-fetch patterns (auth header injection, 401 refresh, offline mutation blocking). This unblocks hook development before OpenAPI spec integration.
- **Set serialization**: Zustand persist middleware uses a custom storage adapter to serialize `Set<string>` as an array in localStorage, since Set is not JSON-serializable.
- **CalendarLayout Routes pattern**: Used internal Routes/Route from react-router-dom (same pattern as WorkLayout) for nested module routing.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created calendar-client.ts fetch wrapper**
- **Found during:** Task 2 (Hook creation)
- **Issue:** Plan specified hooks use apiClient from openapi-fetch, but apiClient is typed against auto-generated types.ts which has no calendar paths (backend built in parallel). Hooks could not use `apiClient.GET('/api/v1/calendar/...')`.
- **Fix:** Created calendar-client.ts with same auth/error patterns as the openapi-fetch client middleware. Hooks use `calendarApi.GET<T>()` instead of `apiClient.GET()`.
- **Files modified:** desktop/src/renderer/src/api/calendar-client.ts
- **Verification:** TypeScript compiles, production build succeeds
- **Committed in:** 955e644 (Task 2 commit)

**2. [Rule 3 - Blocking] Created calendar-types.ts instead of modifying types.ts**
- **Found during:** Task 1 (Type definition)
- **Issue:** Plan specified adding types to api/types.ts, but that file is auto-generated by openapi-typescript and has a "do not modify" header (7687 lines). Manual edits would be overwritten on next `npm run api:generate`.
- **Fix:** Created separate calendar-types.ts with all planned interfaces. Hooks import from this file instead.
- **Files modified:** desktop/src/renderer/src/api/calendar-types.ts
- **Verification:** TypeScript compiles, types importable from hooks
- **Committed in:** a3d976a (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both deviations necessary because the auto-generated types.ts cannot be manually modified. The calendar-client.ts and calendar-types.ts provide equivalent functionality and will be migrated to the typed openapi-fetch client once the backend OpenAPI spec includes calendar endpoints.

## Issues Encountered
- Pre-existing TypeScript errors in work module files (TaskDetailPage, TaskDetailPanel, etc.) were observed during tsc --noEmit. These are unrelated to calendar changes and existed before this plan. No calendar-related errors.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All frontend infrastructure in place for calendar views (07-06), event forms (07-07), sidebar (07-08), and resource booking (07-09)
- rrule package ready for recurrence editor
- All hook functions exported and ready for component consumption
- CalendarLayout shell renders toolbar with navigation and view switcher
- Placeholder content clearly marks what each subsequent plan will implement

## Self-Check: PASSED

---
*Phase: 07-calendar-scheduling*
*Completed: 2026-02-10*
