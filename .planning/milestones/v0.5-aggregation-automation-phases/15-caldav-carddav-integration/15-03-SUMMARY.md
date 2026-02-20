---
phase: 15-caldav-carddav-integration
plan: 03
subsystem: api
tags: [caldav, carddav, gateway, basic-auth, well-known, app-passwords, webdav-push, push-notifications, settings-wizard, admin-panel]

# Dependency graph
requires:
  - phase: 15-caldav-carddav-integration
    provides: CalDAV/CardDAV backend adapters, app-specific password service, sync token service
  - phase: 07-calendar-scheduling
    provides: Calendar events and models for CalDAV sync
  - phase: 02-crm-core
    provides: Contact models for CardDAV sync
provides:
  - Gateway CalDAV/CardDAV routes at /caldav/ and /carddav/ with HTTP Basic Auth
  - .well-known/caldav and .well-known/carddav auto-discovery redirects
  - REST API for app-specific password CRUD under /api/v1/caldav/
  - Admin API for org-wide CalDAV toggle and user audit under /api/v1/admin/caldav/
  - WebDAV-Push subscription service with DB-backed storage and TTL
  - Push notifier firing HTTP POST to subscribed clients on SyncToken changes
  - Frontend CalDAV settings tab with per-client setup wizard (Thunderbird, macOS, Outlook)
  - Frontend admin CalDAV page with org toggle and user audit table
  - TanStack Query hooks for all CalDAV API endpoints
affects: [16-automation-engine]

# Tech tracking
tech-stack:
  added: []
  patterns: [CalDAVPasswordService interface to break gateway-caldav import cycle, adapter pattern in main.go, WebDAV-Push fire-and-forget with 410 auto-unsubscribe, variadic PushNotifier injection for graceful degradation]

key-files:
  created:
    - backend/internal/gateway/route_caldav.go
    - backend/internal/caldav/push_subscription.go
    - backend/internal/caldav/push_notifier.go
    - backend/migrations/000051_create_push_subscriptions.up.sql
    - backend/migrations/000051_create_push_subscriptions.down.sql
    - desktop/src/renderer/src/api/caldav-client.ts
    - desktop/src/renderer/src/api/hooks/useCaldav.ts
    - desktop/src/renderer/src/modules/settings/tabs/CalDAVSettingsTab.tsx
    - desktop/src/renderer/src/modules/admin/CalDAVAdminPage.tsx
  modified:
    - backend/cmd/gateway/main.go
    - backend/internal/config/config.go
    - backend/internal/caldav/app_password.go
    - backend/internal/caldav/sync_token.go
    - desktop/src/renderer/src/modules/settings/SettingsPage.tsx
    - desktop/src/renderer/src/App.tsx

key-decisions:
  - "CalDAVPasswordService interface in gateway package breaks import cycle (caldav->gateway->caldav)"
  - "Adapter pattern in main.go bridges AppPasswordService to CalDAVPasswordService interface"
  - "Variadic PushNotifier parameter on NewSyncTokenService for backward-compatible graceful degradation"
  - "Push notifications are fire-and-forget in goroutines to never block CalDAV write operations"
  - "Auto-unsubscribe on 410 Gone from push endpoints per WebDAV-Push draft spec"
  - "Pool accessor method on AppPasswordService for direct DB queries in route handlers"

patterns-established:
  - "Interface adapter pattern: gateway defines interface, main.go creates adapter wrapping caldav service"
  - "CalDAV/CardDAV route registration outside RouteRegistrar loop (same as WOPI pattern)"
  - "Push subscription with upsert semantics and TTL-based expiry cleanup"
  - "Per-client setup wizard pattern: expandable sections with copy-to-clipboard URLs"

requirements-completed: [INT-01, INT-02, INT-03]

# Metrics
duration: 9min
completed: 2026-02-20
---

# Phase 15 Plan 03: Gateway Integration, Frontend Wizard & WebDAV-Push Summary

**CalDAV/CardDAV gateway routes with Basic Auth and .well-known discovery, REST API for app-specific passwords, frontend settings wizard with per-client instructions, admin toggle page, and WebDAV-Push notification infrastructure**

## Performance

- **Duration:** 9 min
- **Started:** 2026-02-20T11:18:17Z
- **Completed:** 2026-02-20T11:27:22Z
- **Tasks:** 3
- **Files modified:** 15

## Accomplishments
- Gateway serves CalDAV at /caldav/ and CardDAV at /carddav/ with HTTP Basic Auth via app-specific passwords
- .well-known auto-discovery redirects for client auto-configuration
- REST API endpoints for password CRUD, user status, and admin controls under /api/v1/caldav/
- Frontend settings tab with status toggle, app-specific password management, and per-client setup wizard (Thunderbird, macOS Calendar, Outlook+CalDav Synchronizer)
- Admin page for org-wide CalDAV/CardDAV toggle and user audit with password revocation
- WebDAV-Push notification infrastructure: subscriptions stored in DB with TTL, push notifier sends HTTP POST to subscribers on sync token changes
- Push notifications integrate transparently with SyncTokenService -- polling via sync tokens remains as universal fallback

## Task Commits

Each task was committed atomically:

1. **Task 1: Gateway CalDAV/CardDAV routes + Basic Auth + .well-known + API endpoints** - `90cd214` (feat)
2. **Task 2: Frontend settings wizard + admin page + API hooks** - `4115b69` (feat)
3. **Task 3: WebDAV-Push notification subscription and delivery** - `4f5811c` (feat)

## Files Created/Modified
- `backend/internal/gateway/route_caldav.go` - CalDAV/CardDAV routes with Basic Auth, .well-known, REST API, admin endpoints
- `backend/internal/caldav/push_subscription.go` - Push subscription service with subscribe, unsubscribe, cleanup
- `backend/internal/caldav/push_notifier.go` - Fire-and-forget push notification sender with 410 auto-unsubscribe
- `backend/migrations/000051_create_push_subscriptions.up.sql` - Push subscriptions table with upsert unique index
- `backend/migrations/000051_create_push_subscriptions.down.sql` - Down migration
- `backend/cmd/gateway/main.go` - CalDAV wiring: handlers, routes, push notifier, adapter
- `backend/internal/config/config.go` - CalDAVEnabled env var field
- `backend/internal/caldav/app_password.go` - Pool() accessor method
- `backend/internal/caldav/sync_token.go` - PushNotifier integration in IncrementAndLog
- `desktop/src/renderer/src/api/caldav-client.ts` - Typed fetch wrapper for CalDAV API
- `desktop/src/renderer/src/api/hooks/useCaldav.ts` - TanStack Query hooks for passwords, status, admin
- `desktop/src/renderer/src/modules/settings/tabs/CalDAVSettingsTab.tsx` - Settings tab with password CRUD and setup wizard
- `desktop/src/renderer/src/modules/admin/CalDAVAdminPage.tsx` - Admin page with org toggle and user table
- `desktop/src/renderer/src/modules/settings/SettingsPage.tsx` - Added caldav tab entry
- `desktop/src/renderer/src/App.tsx` - Added admin/caldav route

## Decisions Made
- Used CalDAVPasswordService interface in gateway package to break import cycle (caldav imports gateway for ServiceRegistry, so gateway cannot import caldav directly)
- Adapter pattern in main.go converts AppPasswordService to CalDAVPasswordService without exposing internal types
- Variadic PushNotifier parameter on NewSyncTokenService preserves backward compatibility (existing callers pass no notifier)
- Push notifications fire in goroutines with 30s timeout, never blocking the CalDAV write path
- Auto-unsubscribe on HTTP 410 Gone from push endpoints per WebDAV-Push draft spec
- Pool() accessor on AppPasswordService for status/enable/disable handlers that need direct DB access to users table

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Resolved import cycle between gateway and caldav packages**
- **Found during:** Task 1 (Gateway routes creation)
- **Issue:** Plan specified importing caldavpkg in route_caldav.go within the gateway package, but caldav_backend.go already imports gateway for ServiceRegistry -- creating an import cycle
- **Fix:** Defined CalDAVPasswordService interface + CalDAVCtxInjector type in gateway package, created caldavPasswordAdapter struct in main.go that bridges AppPasswordService to the interface
- **Files modified:** backend/internal/gateway/route_caldav.go, backend/cmd/gateway/main.go
- **Verification:** go build ./cmd/gateway/ compiles successfully
- **Committed in:** 90cd214 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Import cycle is an architectural constraint from prior plan decisions. The adapter pattern is clean and does not change external behavior. No scope creep.

## Issues Encountered
None beyond the import cycle addressed above.

## User Setup Required
None - no external service configuration required. CalDAV/CardDAV is disabled by default (both env var and org toggle).

## Next Phase Readiness
- Phase 15 (CalDAV/CardDAV Integration) is COMPLETE -- all 3 plans done
- CalDAV/CardDAV fully functional: data models, backend adapters, gateway routes, frontend UI, push notifications
- External clients (Thunderbird, macOS Calendar, Outlook+plugin) can connect via HTTP Basic Auth with app-specific passwords
- Push-capable clients receive change notifications; polling via sync tokens remains universal fallback
- Admin can toggle CalDAV/CardDAV org-wide and audit user access

## Self-Check: PASSED

- All 9 created files verified on disk
- Commit 90cd214 (Task 1) verified in git log
- Commit 4115b69 (Task 2) verified in git log
- Commit 4f5811c (Task 3) verified in git log
- Go build ./cmd/gateway/ compiles
- Go build ./internal/caldav/ compiles
- TypeScript npx tsc --noEmit compiles
- No fmt.Println in backend code
- No console.log in frontend code

---
*Phase: 15-caldav-carddav-integration*
*Completed: 2026-02-20*
