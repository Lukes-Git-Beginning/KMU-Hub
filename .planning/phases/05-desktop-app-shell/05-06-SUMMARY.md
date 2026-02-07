---
phase: 05-desktop-app-shell
plan: 06
subsystem: dashboard-layouts
tags: [dashboard, server-sync, admin-settings, role-defaults, postgresql, zustand]
depends_on:
  requires: ["05-05"]
  provides: ["server-side dashboard layout persistence", "admin role default configuration", "dashboard layout REST API"]
  affects: ["future admin settings pages", "onboarding flow"]
tech-stack:
  added: []
  patterns: ["gateway direct-DB service (not gRPC)", "Zustand server sync with localStorage offline cache", "debounced server sync"]
key-files:
  created:
    - backend/migrations/000023_create_dashboard_layouts.up.sql
    - backend/migrations/000023_create_dashboard_layouts.down.sql
    - backend/internal/models/dashboard.go
    - backend/internal/gateway/dashboard_repository.go
    - backend/internal/gateway/dashboard_service.go
    - backend/internal/gateway/route_dashboard.go
    - desktop/src/renderer/src/api/hooks/useDashboard.ts
    - desktop/src/renderer/src/modules/settings/DashboardSettings.tsx
  modified:
    - backend/cmd/gateway/main.go
    - backend/api/openapi.yaml
    - desktop/src/renderer/src/api/types.ts
    - desktop/src/renderer/src/stores/dashboard.ts
    - desktop/src/renderer/src/modules/dashboard/DashboardPage.tsx
    - desktop/src/renderer/src/App.tsx
    - desktop/src/renderer/src/components/layout/Sidebar.tsx
key-decisions:
  - decision: "Dashboard service runs in gateway with direct DB access (not gRPC)"
    reason: "Follows existing pattern for file upload handler; no separate microservice needed for simple CRUD"
  - decision: "2-second debounced server sync for layout changes"
    reason: "Prevents hammering server during drag operations while keeping sync responsive"
  - decision: "RequireRole middleware for admin-only endpoints instead of RequirePermission"
    reason: "Dashboard defaults are role-scoped; role check is more semantically appropriate"
  - decision: "localStorage as offline cache with server as source of truth"
    reason: "Fast local startup with eventual consistency; graceful degradation when offline"
metrics:
  duration: ~8min
  completed: 2026-02-07
---

# Phase 05 Plan 06: Role-Based Dashboard Layouts Summary

Server-side dashboard layout persistence with role-based defaults, admin settings page, and Zustand store sync (2s debounce, offline fallback).

## Performance

- 2 tasks, 2 commits
- Duration: ~8 minutes
- Zero deviations from plan
- TypeScript and Go both compile cleanly

## Accomplishments

### Task 1: Backend Dashboard Layouts
- **Migration 000023**: Two new tables -- `dashboard_defaults` (role-based defaults with seeded admin/manager/member layouts) and `user_dashboard_layouts` (per-user overrides with unique constraint on user_id)
- **DashboardRepository**: PostgreSQL implementation with UPSERT queries for both tables, proper error handling with `ErrDashboardNotFound` sentinel
- **DashboardService**: Three-tier fallback logic (user override > role default > hardcoded fallback) with structured logging
- **DashboardRoutes**: 5 REST endpoints implementing RouteRegistrar interface -- GET/PUT/DELETE `/dashboard/layout` for user layouts, GET/PUT `/dashboard/defaults/{role}` for admin-only role management
- **Gateway main.go**: Dashboard repo + service initialized with existing pool, registered in route registrars list
- **OpenAPI spec**: Dashboard tag, 5 path entries, 4 schema definitions (DashboardLayoutResponse, SaveDashboardLayoutRequest, DashboardDefault, SaveDashboardDefaultsRequest)

### Task 2: Dashboard Server Sync and Admin Settings
- **useDashboard hooks**: 5 React Query hooks for dashboard CRUD -- layout fetch, save, reset, defaults fetch, defaults save
- **Dashboard store**: Upgraded from localStorage-only to server-synced with offline fallback; 2-second debounced PUT on layout changes; `initFromServer()` merges server layout with local minW/minH constraints; `partialize` persists only layouts and activeWidgets to localStorage
- **DashboardPage**: Added cloud sync status indicator (synced/syncing/offline icons with German tooltips); server init on mount; reset calls DELETE then refetches role default
- **DashboardSettings**: Admin-only page at `/settings/dashboard` with role tabs (Admin/Manager/Mitarbeiter), widget toggle cards with active badge, save button, "copy current layout as default" button, success/error feedback
- **Sidebar**: Added Cog icon "Einstellungen" nav item filtered to admin role only
- **App.tsx**: Added lazy-loaded `/settings/dashboard` route

## Task Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 975c99b | Backend dashboard layouts: migration, service, API endpoints |
| 2 | 26b8d2b | Dashboard server sync and admin settings page |

## Files Changed

**Created (8):**
- `backend/migrations/000023_create_dashboard_layouts.up.sql` -- dashboard_defaults + user_dashboard_layouts tables
- `backend/migrations/000023_create_dashboard_layouts.down.sql` -- reverse migration
- `backend/internal/models/dashboard.go` -- DashboardDefault, UserDashboardLayout, DashboardLayoutResponse models
- `backend/internal/gateway/dashboard_repository.go` -- PostgresDashboardRepository with UPSERT queries
- `backend/internal/gateway/dashboard_service.go` -- DashboardService with three-tier fallback
- `backend/internal/gateway/route_dashboard.go` -- 5 HTTP handlers with DashboardRoutes registrar
- `desktop/src/renderer/src/api/hooks/useDashboard.ts` -- React Query hooks for dashboard API
- `desktop/src/renderer/src/modules/settings/DashboardSettings.tsx` -- Admin role default configuration page

**Modified (7):**
- `backend/cmd/gateway/main.go` -- Dashboard service registration
- `backend/api/openapi.yaml` -- Dashboard endpoints and schemas
- `desktop/src/renderer/src/api/types.ts` -- Regenerated from OpenAPI spec
- `desktop/src/renderer/src/stores/dashboard.ts` -- Server sync, offline cache
- `desktop/src/renderer/src/modules/dashboard/DashboardPage.tsx` -- Sync indicator, server init
- `desktop/src/renderer/src/App.tsx` -- Settings route
- `desktop/src/renderer/src/components/layout/Sidebar.tsx` -- Admin settings nav item

## Decisions Made

1. **Gateway direct-DB service**: Dashboard service runs in gateway with direct PostgreSQL pool access (same pattern as file upload handler). No separate gRPC microservice needed for this simple CRUD.
2. **2-second debounced sync**: Layout changes debounce for 2 seconds before server PUT, preventing excessive requests during drag/resize operations.
3. **RequireRole("admin") for defaults endpoints**: Role-based check is more appropriate than permission-based for role-scoped configuration.
4. **localStorage as offline cache**: Store partializes only `layouts` and `activeWidgets` to localStorage for fast startup, then reconciles with server as source of truth.
5. **primaryRole priority**: admin > manager > member for determining which role default to fall back to.

## Deviations from Plan

None -- plan executed exactly as written.

## Issues Encountered

- **TypeScript type mismatch**: OpenAPI-generated types use `{ [key: string]: unknown }[]` for layout items, which required `as unknown as Layout[]` casts in the store. This is expected when bridging server JSON schemas with stricter client-side types.

## Next Phase Readiness

- All dashboard functionality complete for Phase 5
- Plan 05-07 (final plan in phase) can proceed
- No blockers

## Self-Check: PASSED
