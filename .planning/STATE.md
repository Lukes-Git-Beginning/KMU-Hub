# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-07)

**Core value:** Every employee completes their entire workday without opening another program
**Current focus:** Phase 6 - Project Management (in progress)

## Current Position

Phase: 6 of 13 (Project Management)
Plan: 4 of 8 in current phase
Status: In progress
Last activity: 2026-02-08 -- Completed 06-04-PLAN.md (Work Service Wiring)

Progress: [████████████████████░] 44% (14/32 plans across phases 4-13)

## Performance Metrics

**Velocity:**
- Total plans completed: 14
- Average duration: ~10 minutes
- Total execution time: ~2h 22min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 04 | 3/3 | ~46min | ~15min |
| 05 | 7/7 | ~66min | ~9min |
| 06 | 4/8 | ~35min | ~8.8min |

**Recent Trend:**
- Last 5 plans: 06-01 (~6min), 06-02 (~7min), 06-03 (~10min), 06-04 (~12min)
- Trend: Wiring plans slightly slower due to larger file count

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

### Pending Todos

- Add toast notification system (sonner or similar) before CRUD forms phase
- CRUD action buttons are placeholder only -- need proper create/edit/delete dialogs in future plan

### Blockers/Concerns

- [Phase 10]: GoBD compliance requires Steuerberater consultation before data model design
- [Phase 11]: ArbZG/BUrlG implementation details need labor law expert review
- [Phase 10]: DATEV Buchungsstapel format spec not publicly detailed -- may need DATEV partner access

## Session Continuity

Last session: 2026-02-08
Stopped at: Completed 06-04-PLAN.md (Work Service Wiring)
Resume file: None
