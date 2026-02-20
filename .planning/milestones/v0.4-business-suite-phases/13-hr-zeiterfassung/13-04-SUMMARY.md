---
phase: 13-hr-zeiterfassung
plan: 04
subsystem: ui
tags: [tanstack-query, react, typescript, hr, zeiterfassung, leave-management, arbzg, burlg]

# Dependency graph
requires:
  - phase: 13-hr-zeiterfassung
    provides: HR gRPC service, gateway routes, time tracking with ArbZG enforcement
provides:
  - TypeScript types for all HR entities (hr-types.ts)
  - API client with auth refresh for HR endpoints (hr-client.ts)
  - 30+ TanStack Query hooks for leave, time, absences, employees, documents, settings (hr-hooks.ts)
  - Header ClockInButton widget with live timer and ArbZG toast warnings
  - Wired HR UI pages replacing Zustand mocks with real API hooks
affects: [14-unified-inbox, 16-automation, 20-plugins]

# Tech tracking
tech-stack:
  added: []
  patterns: [hr-query-key-factory, arbzg-toast-notifications, raf-timer-display, half-day-leave-increments]

key-files:
  created:
    - desktop/src/renderer/src/api/hr-types.ts
    - desktop/src/renderer/src/api/hr-client.ts
    - desktop/src/renderer/src/api/hooks/hr-hooks.ts
    - desktop/src/renderer/src/components/header/ClockInButton.tsx
  modified:
    - desktop/src/renderer/src/modules/team/TeamPage.tsx
    - desktop/src/renderer/src/modules/team/AbsenceCalendar.tsx
    - desktop/src/renderer/src/modules/team/HRApprovalDialog.tsx
    - desktop/src/renderer/src/modules/team/MemberDetailPanel.tsx
    - desktop/src/renderer/src/modules/profil/tabs/AbwesenheitenTab.tsx
    - desktop/src/renderer/src/modules/profil/tabs/ZeiterfassungTab.tsx
    - desktop/src/renderer/src/modules/profil/tabs/DokumenteTab.tsx
    - desktop/src/renderer/src/components/header/index.ts
    - desktop/src/renderer/src/components/layout/Header.tsx

key-decisions:
  - "requestAnimationFrame for live timer display (matching Phase 6 pattern, smoother than setInterval)"
  - "Payroll/training tabs kept on Zustand mock stores (payroll = anti-feature, training not in Phase 13 scope)"
  - "MemberDetailPanel changed to memberId-based API lookup instead of full TeamMember prop"
  - "AbsenceCalendar changed to self-fetching (no props) instead of parent-provided data"
  - "Deutschland-First locale: EUR formatting, de-DE date locale throughout all HR pages"
  - "30s polling for WorkTimeStatus in header ClockInButton for near-real-time display"

patterns-established:
  - "HR query key factory: hrKeys object with methods for all query keys ['hr', domain, ...params]"
  - "ArbZG toast pattern: showArbZGToast() helper with severity-based sonner notifications (info/warning/error)"
  - "Half-day leave increments: morning/afternoon Select dropdowns toggled by isHalfDay checkboxes"
  - "BUrlG carryover warning: pre-expiry and expired banners based on carryoverDeadline and carryoverExpired"

requirements-completed: [HR-01, HR-02, HR-03, HR-04, HR-05, HR-06, HR-07]

# Metrics
duration: 13min
completed: 2026-02-19
---

# Phase 13 Plan 04: Frontend TanStack Query Integration Summary

**HR types, API client, 30+ TanStack Query hooks, and 10 UI pages wired to real HR API with ArbZG toast warnings, leave management, and header clock-in/out button**

## Performance

- **Duration:** ~13 min
- **Started:** 2026-02-19
- **Completed:** 2026-02-19
- **Tasks:** 2/2
- **Files modified:** 13

## Accomplishments

- Created complete TypeScript type system for HR entities (leave, time tracking, absences, employees, documents, settings) with all mutation input types
- Built API client with auth token refresh, offline guard, and typed fetch wrapper for 30+ HR endpoints organized into 5 API groups
- Implemented 30+ TanStack Query hooks with proper cache invalidation, ArbZG compliance toast notifications, and 30s polling for real-time work time status
- Wired 8 existing HR UI pages from Zustand mock stores to TanStack Query hooks backed by real API
- Added header ClockInButton widget with requestAnimationFrame-based live timer, break controls, and ArbZG toast warnings

## Task Commits

Each task was committed atomically:

1. **Task 1: HR types + API client + TanStack Query hooks** - `79b59bf` (feat)
2. **Task 2: Wire existing UI pages to TanStack Query hooks** - `cf521b9` (feat)

## Files Created/Modified

- `desktop/src/renderer/src/api/hr-types.ts` - TypeScript types for all HR entities (LeaveRequest, WorkTimeEntry, EmployeeProfile, etc.)
- `desktop/src/renderer/src/api/hr-client.ts` - Typed fetch wrapper with auth refresh for HR API endpoints (5 API groups: leave, time, absence, employee, settings)
- `desktop/src/renderer/src/api/hooks/hr-hooks.ts` - 30+ TanStack Query hooks with key factory, ArbZG toast helper, cache invalidation
- `desktop/src/renderer/src/components/header/ClockInButton.tsx` - Compact header clock-in/out widget with live RAF timer and break controls
- `desktop/src/renderer/src/components/header/index.ts` - Added ClockInButton export
- `desktop/src/renderer/src/components/layout/Header.tsx` - Added ClockInButton to header widget area
- `desktop/src/renderer/src/modules/team/AbsenceCalendar.tsx` - Replaced mock data with useAbsenceCalendar hook, added department filter, neutral gray when showAbsenceReason disabled
- `desktop/src/renderer/src/modules/team/HRApprovalDialog.tsx` - Replaced callback props with mutations, added overlap detection warning
- `desktop/src/renderer/src/modules/team/MemberDetailPanel.tsx` - Changed to memberId-based API lookup with leave balance card and documents list
- `desktop/src/renderer/src/modules/team/TeamPage.tsx` - Replaced Zustand team store with useEmployees/useLeaveRequests hooks, EUR formatting, de-DE locale
- `desktop/src/renderer/src/modules/profil/tabs/AbwesenheitenTab.tsx` - Leave request form with half-day options, balance card, carryover warnings, sick leave dialog
- `desktop/src/renderer/src/modules/profil/tabs/ZeiterfassungTab.tsx` - Full rewrite with clock-in/out, daily/weekly summaries, overtime display, time correction workflow
- `desktop/src/renderer/src/modules/profil/tabs/DokumenteTab.tsx` - Replaced mock data with API hooks for documents and categories

## Decisions Made

- **requestAnimationFrame for timer**: Matches Phase 6 task timer pattern. Smoother than setInterval, auto-pauses in background tabs.
- **Payroll/training tabs kept on Zustand**: Payroll is an anti-feature (never built, integration-only). Training not in Phase 13 scope. Both stay on mock stores.
- **MemberDetailPanel interface change**: Changed from `member: TeamMember` prop to `memberId: string` prop with API lookup. Cleaner data flow, avoids stale prop data.
- **AbsenceCalendar self-fetching**: Changed from parent-provided `{ members, requests }` props to internal hook-based data fetching. Simplifies parent components.
- **Deutschland-First locale**: EUR formatting via `formatEUR` helper, `de-DE` date locale throughout all HR pages. Consistent with Phase 12 finance module.
- **30s polling for WorkTimeStatus**: Provides near-real-time header display without WebSocket overhead. Matches presence heartbeat interval.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - TypeScript compilation passed with zero errors on both tasks.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 13 (HR & Zeiterfassung) is now COMPLETE with all 4 plans done
- All HR backend services (proto, migrations, business logic, gRPC, gateway routes) and frontend (types, client, hooks, UI pages) are in place
- Ready for Phase 14 (Unified Inbox) which will aggregate Email/Chat/Notifications with routing engine
- Event infrastructure (pg_notify + events table) from Phase 4 provides foundation for Phase 14

## Self-Check: PASSED

- All 4 created files verified present
- All 9 modified files verified present
- Commit `79b59bf` (Task 1) verified in git log
- Commit `cf521b9` (Task 2) verified in git log

---
*Phase: 13-hr-zeiterfassung*
*Completed: 2026-02-19*
