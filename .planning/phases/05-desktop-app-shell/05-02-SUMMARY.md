---
phase: 05-desktop-app-shell
plan: 02
name: App Shell Auth & Navigation
status: complete
subsystem: desktop-renderer
tags: [react, auth, routing, websocket, zustand, api-client, electron]
dependency-graph:
  requires: [05-01]
  provides: [auth-flow, api-client, websocket-manager, app-shell-layout, sidebar-navigation, zustand-stores]
  affects: [05-03, 05-04, 05-05, 05-06, 05-07]
tech-stack:
  added: ["@radix-ui/react-avatar", "@radix-ui/react-label", "@radix-ui/react-separator", "@radix-ui/react-slot", "@radix-ui/react-tooltip", "@tanstack/query-sync-storage-persister"]
  patterns: [openapi-fetch-middleware, zustand-persist, hash-routing, lazy-loading, error-boundary]
key-files:
  created:
    - desktop/src/renderer/src/lib/constants.ts
    - desktop/src/renderer/src/lib/index.ts
    - desktop/src/renderer/src/api/client.ts
    - desktop/src/renderer/src/api/websocket.ts
    - desktop/src/renderer/src/stores/auth.ts
    - desktop/src/renderer/src/stores/ui.ts
    - desktop/src/renderer/src/hooks/useElectronIPC.ts
    - desktop/src/renderer/src/hooks/useOnlineStatus.ts
    - desktop/src/renderer/src/hooks/useWebSocket.ts
    - desktop/src/renderer/src/vite-env.d.ts
    - desktop/src/renderer/src/App.tsx
    - desktop/src/renderer/src/components/layout/AppShell.tsx
    - desktop/src/renderer/src/components/layout/Sidebar.tsx
    - desktop/src/renderer/src/components/layout/Header.tsx
    - desktop/src/renderer/src/components/layout/ModuleShell.tsx
    - desktop/src/renderer/src/components/ui/button.tsx
    - desktop/src/renderer/src/components/ui/input.tsx
    - desktop/src/renderer/src/components/ui/label.tsx
    - desktop/src/renderer/src/components/ui/card.tsx
    - desktop/src/renderer/src/components/ui/separator.tsx
    - desktop/src/renderer/src/components/ui/avatar.tsx
    - desktop/src/renderer/src/components/ui/badge.tsx
    - desktop/src/renderer/src/components/ui/tooltip.tsx
    - desktop/src/renderer/src/modules/auth/LoginPage.tsx
    - desktop/src/renderer/src/modules/dashboard/DashboardPage.tsx
    - desktop/src/renderer/src/modules/crm/CRMLayout.tsx
    - desktop/src/renderer/src/modules/chat/ChatLayout.tsx
    - desktop/src/renderer/src/modules/notifications/NotificationCenter.tsx
  modified:
    - desktop/src/renderer/src/main.tsx
    - desktop/package.json
    - desktop/package-lock.json
decisions:
  - id: 05-02-01
    description: "shadcn CLI installs to @/ literal path -- moved files to src/renderer/src/components/ui/ and added lib/index.ts barrel export"
  - id: 05-02-02
    description: "Added vite-env.d.ts for import.meta.env types (electron-vite v5 does not provide these by default)"
  - id: 05-02-03
    description: "createHashRouter for Electron file:// protocol compatibility (not createBrowserRouter)"
  - id: 05-02-04
    description: "Auth initialize() called before render in main.tsx; App shows loading state while isLoading is true"
  - id: 05-02-05
    description: "GuestRoute wrapper on login to redirect authenticated users back to app"
metrics:
  duration: ~10min
  completed: 2026-02-07
---

# Phase 05 Plan 02: App Shell Auth & Navigation Summary

**One-liner:** React app shell with openapi-fetch auth middleware, WebSocket manager with exponential backoff, Zustand stores for auth (safeStorage) and UI (localStorage), hash-based routing with lazy-loaded module placeholders, and collapsible sidebar navigation.

## What Was Built

### API Client (api/client.ts)
- Type-safe API client built on openapi-fetch with generated types from OpenAPI spec
- Request middleware injects `Authorization: Bearer` header from auth store
- Response middleware intercepts 401s, attempts transparent token refresh, retries request
- Concurrent refresh de-duplication via shared Promise (prevents parallel refresh races)
- Skips refresh for auth endpoints (/login, /refresh, /register) to avoid loops

### WebSocket Manager (api/websocket.ts)
- Singleton WebSocketManager class with connect/disconnect/send/on lifecycle
- Exponential backoff reconnection: [3s, 6s, 12s, 24s, 48s, 60s] with max 10 attempts
- Message routing via handlers Map -- on(type, handler) returns cleanup function
- Automatic reconnect on close (only when token exists), no reconnect after explicit disconnect
- HTTP-to-WS URL conversion for the backend WebSocket endpoint

### Auth Store (stores/auth.ts)
- Zustand store managing user, tokens, isAuthenticated, isLoading
- initialize(): restores tokens from Electron safeStorage, validates via GET /api/v1/auth/me
- login(): POST /api/v1/auth/login, stores tokens via IPC, connects WebSocket
- logout(): POST /api/v1/auth/logout, clears IPC tokens, disconnects WebSocket
- refreshToken(): POST /api/v1/auth/refresh with token rotation, updates stored tokens
- WebSocket auto-connect after both initialize and login

### UI Store (stores/ui.ts)
- Zustand persist middleware with localStorage key 'kmuhub-ui'
- sidebarCollapsed, sidebarWidth, locale (default 'de'), theme ('light'/'dark')

### Hooks
- useElectronIPC: useNativeNotification, useWindowControls, usePlatform
- useOnlineStatus: navigator.onLine with online/offline event listeners
- useWebSocket: manages wsManager lifecycle tied to auth state
- useWSSubscription: subscribe to specific WS message types with auto-cleanup

### App Shell Layout
- AppShell: flex layout with Sidebar + Header + Suspense-wrapped Outlet
- Sidebar: collapsible (64px/256px), NavLink items with active state, user avatar, logout
- Header: module name breadcrumb, online/offline badge, notification bell placeholder
- ModuleShell: loading fallback spinner, error boundary with retry

### Routing (App.tsx)
- createHashRouter for Electron file:// protocol compatibility
- ProtectedRoute guard (redirects unauthenticated to /login)
- GuestRoute guard (redirects authenticated to /)
- Lazy-loaded module routes: dashboard, CRM, chat, notifications
- React Query with PersistQueryClientProvider (localStorage, 24h GC time)

### Login Page
- Centered card with KMU Hub branding
- Email + password form with loading state and error display
- German locale ("Anmelden", "E-Mail", "Passwort")
- Redirects to / on successful login

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | API client, WebSocket manager, stores, and hooks | 2890f8e | api/client.ts, api/websocket.ts, stores/auth.ts, stores/ui.ts, hooks/* |
| 2 | App shell layout, sidebar navigation, login page, and router | 24b06ab | App.tsx, components/layout/*, modules/*, components/ui/* |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] shadcn CLI installed to wrong path**
- **Found during:** Task 2, step 1
- **Issue:** `npx shadcn@latest add` created files in `desktop/@/components/ui/` instead of `desktop/src/renderer/src/components/ui/` because the @/ alias in components.json was interpreted as a literal directory
- **Fix:** Moved files to correct location and created `lib/index.ts` barrel export for `@/lib` import resolution
- **Files modified:** desktop/src/renderer/src/lib/index.ts (created), all ui/* components moved

**2. [Rule 3 - Blocking] Missing Vite env types for import.meta.env**
- **Found during:** Task 1 verification
- **Issue:** `import.meta.env.RENDERER_VITE_API_URL` caused TypeScript error -- electron-vite v5 doesn't automatically provide ImportMeta types for the renderer process
- **Fix:** Created `vite-env.d.ts` with ImportMetaEnv interface declaring RENDERER_VITE_API_URL
- **Files created:** desktop/src/renderer/src/vite-env.d.ts
- **Commit:** 2890f8e

**3. [Rule 3 - Blocking] Missing @tanstack/query-sync-storage-persister**
- **Found during:** Task 2 setup
- **Issue:** createSyncStoragePersister is in a separate package, not bundled with @tanstack/react-query-persist-client
- **Fix:** `npm install @tanstack/query-sync-storage-persister`
- **Files modified:** desktop/package.json, desktop/package-lock.json

## Decisions Made

1. **shadcn component path fix**: Created lib/index.ts barrel export to support `@/lib` imports from shadcn-generated components
2. **vite-env.d.ts**: Added explicit ImportMeta env types since electron-vite v5 doesn't provide them for renderer
3. **Hash routing**: Used createHashRouter (not createBrowserRouter) per plan -- Electron file:// breaks HTML5 history API
4. **Auth init before render**: Called useAuthStore.getState().initialize() in main.tsx before createRoot -- App component shows loading state while checking stored tokens
5. **GuestRoute**: Added guest route guard on /login that redirects already-authenticated users to /

## Verification Results

- TypeScript compiles with zero errors (`npx tsc --noEmit -p tsconfig.web.json`)
- createHashRouter used exclusively (no createBrowserRouter)
- Auth store uses window.electronAPI for token persistence (6 IPC calls)
- Auth store initialize() and login() both call wsManager.connect()
- UI store uses zustand persist middleware with localStorage
- WebSocket manager has exponential backoff reconnection logic
- API client has 401 interceptor with refresh de-duplication

## Next Phase Readiness

Plan 05-03 (CRM UI) can proceed immediately -- the app shell, routing, API client, and stores are ready. Key integration points:
- Import `apiClient` for type-safe backend calls
- Use `useWSSubscription` for real-time updates
- Add routes as children of the AppShell layout in App.tsx
- Lazy-load module components for code splitting

## Self-Check: PASSED
