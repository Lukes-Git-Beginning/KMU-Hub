---
phase: 14-event-infrastructure-unified-inbox
plan: 04
subsystem: ui
tags: [react, typescript, tanstack-query, zustand, radix-ui, inbox, kommunikation]

# Dependency graph
requires:
  - phase: 14-02
    provides: Inbox service layer (message CRUD, team inboxes, routing rules)
  - phase: 14-03
    provides: Inbox gRPC server, gateway HTTP routes, event consumer
provides:
  - KommunikationPage three-column unified inbox UI
  - TanStack Query hooks with optimistic updates for inbox triage
  - Inbox API client (/api/v1/inbox) with auth injection
  - TypeScript types for all inbox entities
  - Zustand UI store for view/sidebar state
  - RoutingRulesEditor with AND/OR condition builder
  - TeamInboxSettings with manual/round-robin assignment
  - SnoozePopover with presets and custom date/time
  - /kommunikation route in App.tsx and sidebar navigation
affects: [phase-16-automation, phase-20-plugins]

# Tech tracking
tech-stack:
  added: []
  patterns: [channel-adaptive-inline-reply, optimistic-triage-mutations, condition-tree-ui-builder]

key-files:
  created:
    - desktop/src/renderer/src/api/inbox-types.ts
    - desktop/src/renderer/src/api/inbox-client.ts
    - desktop/src/renderer/src/api/hooks/useInbox.ts
    - desktop/src/renderer/src/stores/kommunikation.ts
    - desktop/src/renderer/src/modules/kommunikation/KommunikationPage.tsx
    - desktop/src/renderer/src/modules/kommunikation/InboxSidebar.tsx
    - desktop/src/renderer/src/modules/kommunikation/MessageList.tsx
    - desktop/src/renderer/src/modules/kommunikation/MessageDetail.tsx
    - desktop/src/renderer/src/modules/kommunikation/SnoozePopover.tsx
    - desktop/src/renderer/src/modules/kommunikation/TeamInboxSettings.tsx
    - desktop/src/renderer/src/modules/kommunikation/RoutingRulesEditor.tsx
  modified:
    - desktop/src/renderer/src/App.tsx
    - desktop/src/renderer/src/components/layout/sidebar/nav-items.ts

key-decisions:
  - "Channel-adaptive inline reply: email=textarea, chat=single-line, notification=no-reply"
  - "Keyboard shortcuts j/k/e/s/r for GTD-style inbox triage without mouse"
  - "30s staleTime for messages, 15s for unread counts for near-real-time inbox"
  - "Routing rules test panel embedded in editor dialog (not separate page)"
  - "Clock icon for snooze quick action (not timer/loader icon)"

patterns-established:
  - "Channel badge colors: blue=email, green=chat, orange=notification (consistent across components)"
  - "Optimistic triage mutations: mark read/star/archive update cache immediately, rollback on error"
  - "Condition tree builder: recursive ConditionRow component with AND/OR toggle and visual nesting"

requirements-completed: [INBOX-01, INBOX-02, INBOX-03]

# Metrics
duration: 9min
completed: 2026-02-20
---

# Phase 14 Plan 04: Unified Inbox Frontend Summary

**Three-column KommunikationPage with channel-adaptive reply, snooze presets, team inbox management, and AND/OR routing rules builder backed by TanStack Query hooks with optimistic triage updates**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-19T23:54:54Z
- **Completed:** 2026-02-20T00:04:03Z
- **Tasks:** 2
- **Files modified:** 13 (11 created, 2 modified)

## Accomplishments
- Full Unified Inbox frontend with three-column layout (sidebar | message list | detail)
- Channel-colored badges (blue/green/orange) and channel-adaptive inline reply (email textarea, chat input, notification no-reply)
- 25+ TanStack Query hooks with optimistic updates for triage actions (mark read, star, archive, snooze)
- SnoozePopover with 3 presets (1h, tomorrow 09:00, next Monday 09:00) and custom date/time picker
- TeamInboxSettings dialog with manual/round-robin assignment mode and member management
- RoutingRulesEditor with recursive AND/OR condition tree builder and inline rule testing
- Keyboard shortcuts (j/k/e/s/r/Escape) for GTD-style inbox-zero workflow

## Task Commits

Each task was committed atomically:

1. **Task 1: Types + API client + TanStack Query hooks + Zustand store** - `e42a0e6` (feat)
2. **Task 2: KommunikationPage UI components + App.tsx routing** - `c45eeaa` (feat)

## Files Created/Modified
- `desktop/src/renderer/src/api/inbox-types.ts` - TypeScript types for InboxMessage, TeamInbox, RoutingRule, Condition
- `desktop/src/renderer/src/api/inbox-client.ts` - Fetch wrapper for /api/v1/inbox with auth injection and 401 retry
- `desktop/src/renderer/src/api/hooks/useInbox.ts` - 25+ TanStack Query hooks with optimistic updates
- `desktop/src/renderer/src/stores/kommunikation.ts` - Zustand UI store with localStorage persistence
- `desktop/src/renderer/src/modules/kommunikation/KommunikationPage.tsx` - Main three-column layout with keyboard shortcuts
- `desktop/src/renderer/src/modules/kommunikation/InboxSidebar.tsx` - Smart views, channel filters, team inboxes
- `desktop/src/renderer/src/modules/kommunikation/MessageList.tsx` - Message list with channel badges and quick actions
- `desktop/src/renderer/src/modules/kommunikation/MessageDetail.tsx` - Detail pane with channel-adaptive inline reply
- `desktop/src/renderer/src/modules/kommunikation/SnoozePopover.tsx` - Snooze presets + custom date/time picker
- `desktop/src/renderer/src/modules/kommunikation/TeamInboxSettings.tsx` - Team inbox configuration dialog
- `desktop/src/renderer/src/modules/kommunikation/RoutingRulesEditor.tsx` - AND/OR condition builder with rule testing
- `desktop/src/renderer/src/App.tsx` - Added /kommunikation route with lazy import
- `desktop/src/renderer/src/components/layout/sidebar/nav-items.ts` - Added Kommunikation nav entry with MessageSquareText icon

## Decisions Made
- Channel-adaptive inline reply: email gets textarea with "Antworten" button, chat gets single-line input with send button, notification shows "Keine Antwort moeglich"
- Keyboard shortcuts (j/k/e/s/r) for GTD-style triage registered on document.addEventListener with input/textarea guard
- 30s staleTime for messages, 15s for unread counts to balance near-real-time updates with API load
- Routing rules test panel embedded in editor dialog rather than separate page (simpler workflow)
- Used Clock icon for snooze quick action instead of Loader2 (spinner) icon for semantic clarity
- Routing rules editor as toggle panel in message list area rather than separate page (accessible via Settings icon)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed incorrect snooze icon (Loader2 -> Clock)**
- **Found during:** Task 2 (MessageList quick actions)
- **Issue:** Initially used Loader2 (spinner) icon for snooze action which was semantically wrong
- **Fix:** Replaced with Clock icon from lucide-react
- **Files modified:** MessageList.tsx
- **Verification:** TypeScript compiles clean
- **Committed in:** c45eeaa (Task 2 commit)

**2. [Rule 1 - Bug] Added missing ConfirmDialog import in RoutingRulesEditor**
- **Found during:** Task 2 (RoutingRulesEditor)
- **Issue:** ConfirmDialog was used in JSX but not imported
- **Fix:** Added import from @/components/shared
- **Files modified:** RoutingRulesEditor.tsx
- **Verification:** TypeScript compiles clean
- **Committed in:** c45eeaa (Task 2 commit)

**3. [Rule 1 - Bug] Removed unused imports (useCallback, toast) from RoutingRulesEditor**
- **Found during:** Task 2 verification
- **Issue:** useCallback and toast from sonner imported but never used (toasts are in hooks)
- **Fix:** Removed unused imports
- **Files modified:** RoutingRulesEditor.tsx
- **Verification:** TypeScript compiles clean with --noUnusedLocals
- **Committed in:** c45eeaa (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (3 bugs)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep.

## Issues Encountered
- ESLint config missing (eslint v9 migration pending) -- pre-existing issue, not related to this plan. TypeScript compilation used as primary verification.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 14 complete: all 4 plans (proto/types, services, gRPC/gateway, frontend) delivered
- Unified Inbox frontend connects to backend via /api/v1/inbox routes from 14-03
- Event infrastructure (pg_notify + events table) ready for Phase 16 Automation Engine
- Condition tree model in routing rules designed for Phase 16 reuse

## Self-Check: PASSED

- All 12 created/modified files verified on disk
- Both task commits verified in git log (e42a0e6, c45eeaa)
- TypeScript compilation clean (npx tsc --noEmit)
- TypeScript unused locals check clean (--noUnusedLocals)

---
*Phase: 14-event-infrastructure-unified-inbox*
*Completed: 2026-02-20*
