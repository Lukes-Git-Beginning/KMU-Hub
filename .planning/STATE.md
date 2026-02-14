# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-08)

**Core value:** Every employee completes their entire workday without opening another program
**Current focus:** Phase 8 - Video, Voice & Meetings (Next)

## Current Position

Phase: 7 of 20 (Calendar & Scheduling) -- COMPLETE
Plan: 9 of 9 in current phase (all complete)
Status: Phase 7 complete, Phase 8 next
Last activity: 2026-02-11 -- Phase 7 complete via design integration (Plans 07-06 to 07-09 replaced by Darien's UI merge)

Progress: [████████████░░░░░░░░░░░░] 42% (29/63 plans across phases 4-20)

## Performance Metrics

**Velocity:**
- Total plans completed: 24
- Average duration: ~9 minutes
- Total execution time: ~3h 43min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 04 | 3/3 | ~46min | ~15min |
| 05 | 7/7 | ~66min | ~9min |
| 06 | 10/10 | ~88min | ~8.8min |
| 07 | 9/9 | ~48min | ~5min |

**Recent Trend:**
- Last 5 plans: 07-01 (~7min), 07-02 (~5min), 07-03 (~8min), 07-04 (~8min), 07-05+UI (~20min)
- Trend: Consistent ~5-10min per plan, design integration saved ~4 plans

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Service consolidation -- 3 new backend services (Work, Biz, Automation) instead of 8 separate ones
- [Roadmap]: Gateway refactoring bundled with Phase 4 (before adding new services)
- [Roadmap]: Notifications first (unblocks all future modules)
- [Roadmap]: Full IMAP+SMTP email in v1 (user decision despite research suggesting deferral)
- [Roadmap]: Automation and Plugins last (need stable APIs from all other modules)
- [Roadmap]: Feature gap analysis expansion -- 13 to 18 phases, Meeting Management merged into Phase 8, Security & Compliance as Phase 9 gatekeeper, Documents & Files as Phase 11, 3 integration mini-phases (14-16)
- [04-01]: WebSocket hub stays in main.go (cross-cutting, needs both chat + auth clients)
- [04-02]: Raw pgx over pgxlisten for event bus (pgxlisten pre-v1, unstable)
- [04-02]: Dual write (events table + pg_notify) for event durability
- [04-02]: DeliveryCallback pattern decouples notification service from WebSocket delivery
- [04-03]: Dual pg_notify channels: 'events' for notification processing, 'notification_delivery' for gateway WebSocket push
- [05-01]: electron-vite v5 with build.externalizeDeps (deprecated plugin replaced)
- [05-01]: TSconfig split: node (bundler resolution) + web (DOM, react-jsx, path aliases)
- [05-01]: CSP unsafe-inline for dev only (Vite HMR), production uses self only
- [05-02]: createHashRouter for Electron file:// protocol compatibility
- [05-02]: Auth init before render; GuestRoute guard on login page
- [05-03]: Routes/Route for CRM sub-navigation (module-level routing inside AppShell Outlet)
- [05-04]: WebSocket cache sync via queryClient.setQueryData with invalidation fallback
- [05-04]: Native push only when document.hasFocus() === false
- [05-05]: Widget registry pattern -- centralized definitions with lazy-loaded components
- [05-05]: Per-widget ErrorBoundary for crash isolation
- [05-06]: Dashboard service in gateway with direct DB access (not gRPC)
- [05-06]: localStorage as offline cache, server as source of truth
- [05-07]: 24h maxAge for TanStack Query cache with 5min staleTime
- [05-07]: Mutations blocked when offline via OfflineError in API client
- [05-07]: CORS origins include localhost:5173 for Electron dev
- [06-01]: Task constants prefixed with Task* to avoid collision with notification model priority constants
- [06-02]: Project key auto-normalizes to uppercase; validation rejects non-alphanumeric only
- [06-02]: Status service trusts caller for authorization (gRPC server checks membership)
- [06-02]: GetUserPreference returns nil when no preference set (caller applies defaults)
- [06-03]: Standalone tasks get task_number=0 (no project counter increment)
- [06-03]: Comment service depends on taskRepo.CreateActivity for activity logging
- [06-03]: @mention pattern uses @{uuid} format for deterministic user resolution
- [06-03]: Cycle detection only for blocking deps (blocks/blocked_by), not relates_to/duplicates
- [06-03]: MoveTask handles completed_at setting/clearing based on status is_closed flag
- [06-04]: gRPC server uses uuid.Nil + isAdmin=true (gateway handles auth)
- [06-04]: Template key auto-generated from name prefix + UUID suffix
- [06-04]: Work routes follow exact same RouteRegistrar pattern as CRM/Chat/Notification
- [06-05]: API types regenerated to include work/project/task endpoints from OpenAPI spec
- [06-05]: Project create dialog bundles status creation as sequential API calls after project POST
- [06-05]: My Tasks hooks auto-set assignee_id from auth store (user sees only own tasks)
- [06-06]: Client-side grouping for instant group switching without network roundtrip
- [06-06]: closestCorners collision detection for multi-container Kanban DnD
- [06-06]: Max 3 visual nesting levels on Kanban to keep board clean
- [06-06]: Subtask cards not independently draggable on Kanban (use list view or detail panel)
- [06-07]: Fixed overlay panel (CSS transform) for task detail slide-over, no Radix Sheet dependency
- [06-07]: Two-step file upload: multipart to MinIO via /files/upload, then JSON metadata to task files
- [06-07]: Nested Routes in ProjectDetailPage for board view vs task detail page
- [06-07]: Tab-based activity/comments view (Alle/Kommentare/Aktivitaet) for user control
- [06-08]: Context-aware auto-suggest shows banner but never auto-applies links
- [06-08]: Standalone tasks use task_number=0 with TASK system key
- [06-08]: CRM search API reused for entity linking (no new backend search endpoint)
- [06-08]: Custom fields reuse CRM engine with entity_type=task
- [06-08]: Move-to-project updates project_id via existing task update API
- [06-09]: Migration renumbered to 000031 to avoid collision with uncommitted time_entries migration
- [06-09]: Batch dependency fetching via useQueries for tasks with has_blocked_deps flag
- [06-09]: Gantt view is read-only in v1 (bars clickable, not draggable)
- [06-09]: Critical path uses forward/backward pass CPM with Kahn's topological sort
- [06-10]: Separate timeentry package under internal/work/ for clean separation of concerns
- [06-10]: Auto-stop previous timer in service layer ensures single-timer invariant at DB level
- [06-10]: Partial index idx_time_entries_active for O(1) active timer lookup
- [06-10]: requestAnimationFrame for timer display (smoother, auto-pauses in background tabs)
- [06-10]: Migration 000030 for time_entries (06-09 used 000031 for gantt view type)
- [07-01]: Separate calendar.proto file rather than extending work.proto (cleaner separation, same binary)
- [07-01]: Deferred FK constraints: resource_id FK added via ALTER TABLE in migration 000034 after resources table exists
- [07-01]: Calendar-prefixed model naming (CalendarEvent, EventCategory) to avoid collision with notification Event model
- [07-01]: 40 RPCs in CalendarService covering calendars, events, resources, bookings, holidays, preferences, LiveKit
- [07-02]: Separate calendar-types.ts instead of modifying auto-generated types.ts (openapi-typescript)
- [07-02]: calendar-client.ts fetch wrapper mirrors openapi-fetch auth/error patterns for pre-OpenAPI hooks
- [07-02]: Set<string> serialized as array in Zustand persist localStorage adapter
- [07-02]: CalendarLayout uses internal Routes/Route pattern (same as WorkLayout)
- [07-03]: Calendar permission hierarchy: view < edit < admin numeric levels, owner implicit admin
- [07-03]: EnsurePersonalCalendar on every ListByUser for auto-creation with DACH defaults
- [07-03]: Three-way recurring edit: this=exception, this_and_future=split with SetUntil, all=update master
- [07-03]: Event emitter optional via SetEventEmitter (same pattern as task service)
- [07-03]: rrule-go v1.8.2 re-added to go.mod (was missing from working tree despite 07-01 commit)
- [07-04]: HolidayFetcher interface abstracts Nager client for testability
- [07-04]: LiveKit disabled-by-default: empty config values = feature off
- [07-04]: BookingConflictError carries alternative resource suggestions
- [07-04]: Resource delete is soft-delete (is_active=false), bookings preserved
- [07-05]: CalendarService registered in same binary as WorkService (shared gRPC port :50055)
- [07-05]: CalendarRoutes uses ServiceName "work" to reuse existing gRPC connection from gateway
- [07-05]: Proto fields Date/DueDate are strings (YYYY-MM-DD), not Timestamps
- [07-UI]: Design integration: Darien's KalenderPage.tsx (1613 lines) merged from design/brainstorm
- [07-UI]: Adapter layer (adapters.ts) transforms backend types to UI types bidirectionally
- [07-UI]: Sonner toast system added to App.tsx (was pending todo)
- [07-UI]: Plans 07-06 to 07-09 made obsolete by design integration (saved ~4 plans)
- [07-UI]: D2 color system (globals.css) merged from design/brainstorm
- [07-UI]: New Radix UI components: alert-dialog, checkbox, dropdown-menu, sheet, switch

### Pending Todos

- CRUD action buttons are placeholder only -- need proper create/edit/delete dialogs in future plan

### Blockers/Concerns

- [Phase 12]: GoBD compliance requires Steuerberater consultation before data model design
- [Phase 13]: ArbZG/BUrlG implementation details need labor law expert review
- [Phase 12]: DATEV Buchungsstapel format spec not publicly detailed -- may need DATEV partner access

## Session Continuity

Last session: 2026-02-11
Stopped at: Phase 7 COMPLETE -- design integration merged, all 9 plans done
Resume file: None
Next: Phase 8 - Video, Voice & Meetings
