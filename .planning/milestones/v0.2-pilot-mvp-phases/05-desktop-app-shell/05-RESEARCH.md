# Phase 5: Desktop App Shell - Research

**Researched:** 2026-02-07
**Domain:** Electron + React 19 + TypeScript desktop application
**Confidence:** HIGH

## Summary

This phase delivers the first frontend for KMU Hub -- an Electron desktop application with React 19, TypeScript, and Tailwind CSS v4. The existing `desktop/` skeleton already declares `electron-vite` v2.4, `react` v19, `react-router-dom` v7, and `tailwindcss` v4 in package.json, but the `src/` directory is empty. The backend provides ~70 HTTP endpoints via OpenAPI spec and a WebSocket hub at `/api/v1/ws` with JWT auth, covering Auth, CRM, Chat, and Notifications.

The recommended approach uses electron-vite's three-process architecture (main/preload/renderer), shadcn/ui for the component library (copy-paste approach avoids dependency lock-in), TanStack Query for server state + Zustand for client state, and openapi-typescript + openapi-fetch for type-safe API consumption generated directly from the existing OpenAPI spec. For the widget/dashboard system, react-grid-layout v2 provides drag-and-drop with full TypeScript support. Offline caching uses TanStack Query's built-in persistence layer to localStorage/IndexedDB -- no SQLite needed for v1.

**Primary recommendation:** Build a modular Electron shell where each business module (CRM, Chat, future PM/Calendar/etc.) is a lazy-loaded route bundle with its own feature folder, sharing a common component library and API layer. Prioritize the shell architecture and component library in plan 05-01, defer detailed module UI to 05-02, and keep the widget system (05-03) and offline caching (05-04) as separate concerns.

## Standard Stack

The established libraries/tools for this domain:

### Core (already in package.json)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| electron | ^33.0.0 | Desktop runtime | Only production-grade option for web-tech desktop apps |
| electron-vite | ^2.4.0 | Build tooling | Purpose-built for Electron: handles main/preload/renderer, HMR, V8 bytecode |
| react | ^19.0.0 | UI framework | Already decided in CLAUDE.md |
| react-dom | ^19.0.0 | DOM rendering | Required by React |
| react-router-dom | ^7.0.0 | Client-side routing | v7 has lazy route discovery, data loading, modern API |
| tailwindcss | ^4.0.0 | Styling | Already decided; v4 uses Vite plugin, no PostCSS config needed |
| typescript | ^5.7.0 | Type safety | Already decided |
| vitest | ^2.1.0 | Testing | Already in devDependencies, fast Vite-native testing |

### To Add - API Layer

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| openapi-typescript | ^7.0.0 | Generate TS types from OpenAPI | Zero-runtime type generation from existing backend spec |
| openapi-fetch | ^0.13.0 | Type-safe HTTP client | Uses generated types, no codegen bloat, works with fetch API |

### To Add - State Management

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @tanstack/react-query | ^5.0.0 | Server state (API data) | Caching, background refresh, offline support, stale-while-revalidate |
| zustand | ^5.0.0 | Client state (UI, auth, prefs) | 1KB, hooks-based, no providers, persist middleware for localStorage |

### To Add - UI Components

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| shadcn/ui | latest CLI | Component library (copy-paste) | Tailwind v4 + React 19 compatible, full source ownership, accessible (Radix) |
| lucide-react | ^0.460.0 | Icons | 1500+ icons, tree-shakeable, TypeScript, consistent with shadcn/ui |
| react-grid-layout | ^2.2.0 | Dashboard widget grid | v2 is full TypeScript rewrite, hooks API, drag-and-drop + resize |
| class-variance-authority | ^0.7.0 | Component variant management | Used by shadcn/ui for styling variants |
| clsx | ^2.0.0 | Conditional classnames | Tiny utility, standard with Tailwind |
| tailwind-merge | ^2.0.0 | Tailwind class merging | Prevents conflicting Tailwind classes in component APIs |

### To Add - Utilities

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| date-fns | ^4.0.0 | Date formatting/manipulation | Tree-shakeable, locale support (de, fr, it for DACH) |
| @tanstack/react-query-persist-client | ^5.0.0 | Query cache persistence | Enables offline cache via localStorage/IndexedDB |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Zustand | Redux Toolkit | Redux is heavier (boilerplate, providers); Zustand is sufficient for client-only state when TanStack Query handles server state |
| shadcn/ui | MUI / Ant Design | MUI/Ant are full dependency-based; shadcn gives source ownership, better Tailwind integration, smaller bundles |
| openapi-fetch | axios + manual types | axios needs manual type maintenance; openapi-fetch auto-types from spec |
| react-grid-layout | dnd-kit + custom grid | dnd-kit is lower-level; react-grid-layout is purpose-built for dashboard grids |
| lucide-react | heroicons | Heroicons has only 316 icons vs 1500+; lucide has broader coverage for enterprise app |
| localStorage persistence | better-sqlite3 | SQLite is overkill for caching recent data; localStorage/IndexedDB sufficient for v1 offline |

**Installation:**
```bash
cd desktop

# Core dependencies (already in package.json)
npm install

# API layer
npm install openapi-typescript openapi-fetch

# State management
npm install @tanstack/react-query zustand @tanstack/react-query-persist-client

# UI components (shadcn/ui is installed via CLI, not npm)
npx shadcn@latest init
npm install lucide-react react-grid-layout
npm install class-variance-authority clsx tailwind-merge

# Utilities
npm install date-fns

# Dev dependencies
npm install -D @tailwindcss/vite @types/react-grid-layout
```

Note: `openapi-typescript` is a dev/build-time tool. The generated types file has zero runtime cost.

## Architecture Patterns

### Recommended Project Structure

```
desktop/
  electron.vite.config.ts          # Main/preload/renderer build config
  package.json                     # Already exists
  tsconfig.json                    # Already exists (update paths)
  tsconfig.node.json               # For main + preload processes
  tsconfig.web.json                # For renderer process
  components.json                  # shadcn/ui config
  src/
    main/                          # Electron main process
      index.ts                     # App lifecycle, BrowserWindow
      ipc/                         # IPC handler registrations
        auth.ts                    # Auth-related IPC (token storage)
        notifications.ts           # Native notification forwarding
        window.ts                  # Window management IPC
        app.ts                     # App-level IPC (updates, tray)
      tray.ts                      # System tray setup
      menu.ts                      # Application menu
      updater.ts                   # Auto-update logic
    preload/
      index.ts                     # contextBridge.exposeInMainWorld
      types.ts                     # IPC API type definitions
    renderer/
      index.html                   # Entry HTML
      src/
        main.tsx                   # React entry point
        App.tsx                    # Root component (providers, router)
        api/                       # API layer
          client.ts                # openapi-fetch client setup
          types.ts                 # Generated from OpenAPI (build step)
          websocket.ts             # WebSocket connection manager
          hooks/                   # Per-domain query hooks
            useAuth.ts
            useContacts.ts
            useDeals.ts
            useChannels.ts
            useMessages.ts
            useNotifications.ts
        stores/                    # Zustand stores (client state)
          auth.ts                  # Auth tokens, current user
          ui.ts                    # Sidebar collapsed, theme, locale
          dashboard.ts             # Widget layouts per role
        components/                # Shared component library
          ui/                      # shadcn/ui components (Button, Input, etc.)
          layout/                  # App shell components
            Sidebar.tsx
            Header.tsx
            ModuleShell.tsx
            AppShell.tsx
          widgets/                 # Dashboard widget components
            WidgetContainer.tsx
            WidgetRegistry.tsx
        modules/                   # Feature modules (lazy-loaded)
          dashboard/               # Home/dashboard module
            DashboardPage.tsx
            widgets/               # Dashboard-specific widgets
              RecentContacts.tsx
              DealPipeline.tsx
              UnreadMessages.tsx
              ActivityFeed.tsx
              QuickActions.tsx
          crm/                     # CRM module
            CRMLayout.tsx
            contacts/
            companies/
            deals/
            activities/
            search/
          chat/                    # Chat module
            ChatLayout.tsx
            channels/
            messages/
            threads/
          notifications/           # Notification center
            NotificationCenter.tsx
            NotificationBell.tsx
        hooks/                     # Shared React hooks
          useOnlineStatus.ts
          useWebSocket.ts
          useElectronIPC.ts
        lib/                       # Utilities
          cn.ts                    # clsx + tailwind-merge helper
          date.ts                  # date-fns wrappers with locale
          constants.ts
        styles/
          globals.css              # Tailwind @import, custom theme
          themes/                  # Light/dark theme variables
```

### Pattern 1: Electron IPC Bridge (Security-First)

**What:** All Electron native APIs accessed via typed IPC bridge, never directly from renderer.
**When to use:** Always -- this is the only secure pattern for modern Electron.

```typescript
// src/preload/types.ts
export interface ElectronAPI {
  // Auth
  getStoredTokens: () => Promise<{ accessToken: string; refreshToken: string } | null>;
  storeTokens: (tokens: { accessToken: string; refreshToken: string }) => Promise<void>;
  clearTokens: () => Promise<void>;

  // Notifications
  showNativeNotification: (title: string, body: string) => void;

  // Window
  minimize: () => void;
  maximize: () => void;
  close: () => void;
  isMaximized: () => Promise<boolean>;

  // App
  getAppVersion: () => Promise<string>;
  onDeepLink: (callback: (url: string) => void) => () => void;

  // Platform
  platform: NodeJS.Platform;
}

// src/preload/index.ts
import { contextBridge, ipcRenderer } from 'electron';
import type { ElectronAPI } from './types';

const electronAPI: ElectronAPI = {
  getStoredTokens: () => ipcRenderer.invoke('auth:get-tokens'),
  storeTokens: (tokens) => ipcRenderer.invoke('auth:store-tokens', tokens),
  clearTokens: () => ipcRenderer.invoke('auth:clear-tokens'),

  showNativeNotification: (title, body) =>
    ipcRenderer.send('notification:show', { title, body }),

  minimize: () => ipcRenderer.send('window:minimize'),
  maximize: () => ipcRenderer.send('window:maximize'),
  close: () => ipcRenderer.send('window:close'),
  isMaximized: () => ipcRenderer.invoke('window:is-maximized'),

  getAppVersion: () => ipcRenderer.invoke('app:version'),
  onDeepLink: (callback) => {
    const handler = (_event: any, url: string) => callback(url);
    ipcRenderer.on('app:deep-link', handler);
    return () => ipcRenderer.removeListener('app:deep-link', handler);
  },

  platform: process.platform,
};

contextBridge.exposeInMainWorld('electronAPI', electronAPI);

// src/renderer/src/global.d.ts
import type { ElectronAPI } from '../../preload/types';

declare global {
  interface Window {
    electronAPI: ElectronAPI;
  }
}
```

### Pattern 2: Type-Safe API Client from OpenAPI

**What:** Generate TypeScript types from the existing backend OpenAPI spec, use openapi-fetch for zero-overhead type-safe HTTP calls.
**When to use:** All API communication.

```typescript
// Generate types (build script in package.json):
// "api:generate": "openapi-typescript ../backend/api/openapi.yaml -o src/renderer/src/api/types.ts"

// src/renderer/src/api/client.ts
import createClient from 'openapi-fetch';
import type { paths } from './types';

const API_BASE = import.meta.env.RENDERER_VITE_API_URL || 'http://localhost:8080';

export const apiClient = createClient<paths>({
  baseUrl: API_BASE,
});

// Add auth interceptor
apiClient.use({
  async onRequest({ request }) {
    const tokens = await window.electronAPI.getStoredTokens();
    if (tokens?.accessToken) {
      request.headers.set('Authorization', `Bearer ${tokens.accessToken}`);
    }
    return request;
  },
  async onResponse({ response }) {
    if (response.status === 401) {
      // Trigger token refresh or logout
    }
    return response;
  },
});

// Usage in TanStack Query hook:
// src/renderer/src/api/hooks/useContacts.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';

export function useContacts(params?: { page?: number; limit?: number; search?: string }) {
  return useQuery({
    queryKey: ['contacts', params],
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/contacts', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
  });
}
```

### Pattern 3: Module Lazy Loading with React.lazy + Suspense

**What:** Each module (CRM, Chat, Dashboard) is a separate chunk loaded on demand.
**When to use:** All route-level module boundaries.

```typescript
// src/renderer/src/App.tsx
import { lazy, Suspense } from 'react';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';
import { AppShell } from './components/layout/AppShell';
import { ModuleLoadingFallback } from './components/layout/ModuleShell';

const DashboardModule = lazy(() => import('./modules/dashboard/DashboardPage'));
const CRMModule = lazy(() => import('./modules/crm/CRMLayout'));
const ChatModule = lazy(() => import('./modules/chat/ChatLayout'));
const NotificationCenter = lazy(() => import('./modules/notifications/NotificationCenter'));

const router = createBrowserRouter([
  {
    element: <AppShell />,
    children: [
      {
        path: '/',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <DashboardModule />
          </Suspense>
        ),
      },
      {
        path: '/crm/*',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <CRMModule />
          </Suspense>
        ),
      },
      {
        path: '/chat/*',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <ChatModule />
          </Suspense>
        ),
      },
      {
        path: '/notifications',
        element: (
          <Suspense fallback={<ModuleLoadingFallback />}>
            <NotificationCenter />
          </Suspense>
        ),
      },
    ],
  },
]);
```

### Pattern 4: WebSocket Manager (Singleton with Reconnection)

**What:** Single WebSocket connection managed in the renderer, handling chat, notifications, and typing indicators.
**When to use:** All real-time features.

```typescript
// src/renderer/src/api/websocket.ts
type WSMessageHandler = (message: WSMessage) => void;

class WebSocketManager {
  private ws: WebSocket | null = null;
  private handlers = new Map<string, Set<WSMessageHandler>>();
  private reconnectTimer: number | null = null;
  private url: string;

  constructor(baseUrl: string) {
    this.url = baseUrl.replace('http', 'ws') + '/api/v1/ws';
  }

  async connect(accessToken: string): Promise<void> {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    this.ws = new WebSocket(`${this.url}?token=${accessToken}`);

    this.ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      const handlers = this.handlers.get(msg.type);
      handlers?.forEach(handler => handler(msg));
    };

    this.ws.onclose = () => {
      this.scheduleReconnect();
    };
  }

  on(type: string, handler: WSMessageHandler): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set());
    }
    this.handlers.get(type)!.add(handler);
    return () => this.handlers.get(type)?.delete(handler);
  }

  send(message: object): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return;
    this.reconnectTimer = window.setTimeout(async () => {
      this.reconnectTimer = null;
      const tokens = await window.electronAPI.getStoredTokens();
      if (tokens) this.connect(tokens.accessToken);
    }, 3000);
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.ws?.close();
    this.ws = null;
  }
}

export const wsManager = new WebSocketManager(
  import.meta.env.RENDERER_VITE_API_URL || 'http://localhost:8080'
);
```

### Pattern 5: Widget System with React Grid Layout

**What:** Dashboard uses a widget registry + react-grid-layout for drag-and-drop personalization.
**When to use:** Dashboard/workspace personalization (DESK-02, DESK-03).

```typescript
// src/renderer/src/components/widgets/WidgetRegistry.tsx
import { lazy, ComponentType, LazyExoticComponent } from 'react';

export interface WidgetDefinition {
  id: string;
  name: string;
  description: string;
  defaultSize: { w: number; h: number };
  minSize: { w: number; h: number };
  component: LazyExoticComponent<ComponentType<any>>;
  roles?: string[]; // Which roles see this widget by default
}

export const widgetRegistry: Record<string, WidgetDefinition> = {
  'recent-contacts': {
    id: 'recent-contacts',
    name: 'Recent Contacts',
    description: 'Shows recently accessed contacts',
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/RecentContacts')),
    roles: ['admin', 'manager', 'member'],
  },
  'deal-pipeline': {
    id: 'deal-pipeline',
    name: 'Deal Pipeline',
    description: 'Visual overview of deal stages',
    defaultSize: { w: 6, h: 4 },
    minSize: { w: 4, h: 3 },
    component: lazy(() => import('../../modules/dashboard/widgets/DealPipeline')),
    roles: ['admin', 'manager'],
  },
  'unread-messages': {
    id: 'unread-messages',
    name: 'Unread Messages',
    description: 'Shows unread chat messages',
    defaultSize: { w: 4, h: 3 },
    minSize: { w: 2, h: 2 },
    component: lazy(() => import('../../modules/dashboard/widgets/UnreadMessages')),
    roles: ['admin', 'manager', 'member'],
  },
  // ... more widgets
};
```

### Anti-Patterns to Avoid

- **Accessing Node.js APIs directly in renderer:** Always use contextBridge. Never enable `nodeIntegration: true`.
- **Single massive bundle:** Use React.lazy for every module boundary. Each module should be a separate chunk.
- **Global state for server data:** Use TanStack Query for anything from the API. Zustand is ONLY for client-side UI state (sidebar open, theme, locale).
- **Manual API type definitions:** Generate types from the OpenAPI spec. Manual types will drift from the backend.
- **Storing auth tokens in renderer memory:** Tokens go through IPC to main process, stored in Electron's safeStorage or encrypted file. Never in localStorage.
- **Multiple WebSocket connections:** One WebSocket connection per user session, multiplexed for chat + notifications.
- **CSS-in-JS with Tailwind:** Use Tailwind utility classes only. No styled-components or emotion alongside Tailwind.
- **Direct window.electronAPI calls in components:** Wrap in custom hooks (useElectronIPC) for testability and abstraction.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API type safety | Manual TypeScript interfaces for 70+ endpoints | openapi-typescript + openapi-fetch | Spec drift, maintenance nightmare at scale |
| Server state caching | Custom cache layer with invalidation | TanStack Query | Cache invalidation, background refresh, stale-while-revalidate, retry logic |
| Dashboard drag-and-drop grid | Custom drag-drop with CSS Grid | react-grid-layout v2 | Collision detection, responsive breakpoints, resize handles, layout persistence |
| Component library | Custom buttons/inputs/modals from scratch | shadcn/ui (copy-paste into project) | Accessibility (Radix primitives), consistent design, keyboard navigation, focus management |
| Icon system | SVG sprite sheets or custom icons | lucide-react | Tree-shakeable, consistent style, 1500+ icons, TypeScript |
| CSS class management | String concatenation for Tailwind classes | clsx + tailwind-merge (via cn() helper) | Handles conditional classes, resolves Tailwind conflicts |
| Date formatting | Manual Intl.DateTimeFormat wrappers | date-fns with locale | Timezone handling, DACH locale support (de, de-AT, de-CH) |
| Form state | Custom useState per field | React 19 useActionState or react-hook-form | Validation, error handling, dirty tracking |
| Auth token refresh | Manual retry logic on 401 | openapi-fetch middleware + TanStack Query retry | Race conditions, concurrent request handling, token rotation |

**Key insight:** This is the first frontend code in the project. Building a component library from scratch would consume weeks that should go toward module features. shadcn/ui gives production-quality components with full source ownership -- you copy the code, you own the code, you can customize everything. Combined with openapi-typescript eliminating manual type maintenance for 70+ endpoints, the developer can focus on business logic instead of infrastructure.

## Common Pitfalls

### Pitfall 1: Electron Memory Bloat

**What goes wrong:** Electron app uses 500MB+ RAM, violating the 300MB constraint.
**Why it happens:** Loading all modules upfront, large node_modules in main process, memory leaks from WebSocket listeners.
**How to avoid:**
- Lazy load ALL module routes (React.lazy + dynamic import)
- Only import what you need in main process (no renderer libraries)
- Use `webPreferences.backgroundThrottling: true` (default)
- Profile with Chrome DevTools Memory tab regularly
- Clean up WebSocket event listeners on unmount
- Use `requestIdleCallback` for non-critical rendering
**Warning signs:** RAM usage exceeds 250MB with a single module active. Use `process.memoryUsage()` in main process and `performance.memory` in renderer.

### Pitfall 2: CSP vs Tailwind CSS

**What goes wrong:** Content Security Policy blocks Tailwind styles, or CSP is set to `unsafe-inline`/`unsafe-eval` which weakens security.
**Why it happens:** The CLAUDE.md explicitly warns about this: "Tailwind JIT (Runtime) braucht unsafe-eval, inkompatibel mit CSP Nonces."
**How to avoid:** Tailwind CSS v4 with the `@tailwindcss/vite` plugin compiles at BUILD time, not runtime. No `unsafe-eval` or `unsafe-inline` needed. Set a strict CSP in the main process BrowserWindow:
```
"default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self' ws://localhost:8080 http://localhost:8080; img-src 'self' data: blob:"
```
**Warning signs:** Electron console showing CSP violations. Test in production build, not just dev mode.

### Pitfall 3: IPC Security Holes

**What goes wrong:** Renderer gains access to arbitrary Node.js APIs or IPC channels.
**Why it happens:** Exposing `ipcRenderer.send` directly, or not validating IPC arguments in main process handlers.
**How to avoid:**
- Never expose raw `ipcRenderer` -- wrap each operation in a specific function
- Validate all arguments in `ipcMain.handle` handlers
- Use TypeScript types shared between preload and main for IPC contracts
- Never set `nodeIntegration: true` or `contextIsolation: false`
**Warning signs:** Code using `require('electron')` in renderer files.

### Pitfall 4: Token Storage Insecurity

**What goes wrong:** JWT tokens stored in localStorage are accessible to any script running in the renderer.
**Why it happens:** Treating Electron like a regular web app, using browser storage APIs for sensitive data.
**How to avoid:**
- Store tokens in main process using Electron's `safeStorage` API (OS keychain) or encrypted file
- Renderer requests tokens via IPC, never stores them directly
- Access token kept in memory (Zustand store), refresh token in main process only
**Warning signs:** Any `localStorage.setItem('token', ...)` or `sessionStorage` usage for auth tokens.

### Pitfall 5: WebSocket Reconnection Storms

**What goes wrong:** After network disconnect, app creates many WebSocket connections simultaneously, overwhelming the server.
**Why it happens:** No exponential backoff, no connection state tracking.
**How to avoid:**
- Use exponential backoff with jitter for reconnection (3s, 6s, 12s, max 60s)
- Track connection state (connecting/connected/disconnected) to prevent duplicate attempts
- Re-authenticate before reconnecting (access token may have expired)
- Limit reconnection attempts (max 10), then show "reconnect" button
**Warning signs:** Multiple WebSocket connections open simultaneously to the same user.

### Pitfall 6: OpenAPI Type Generation Drift

**What goes wrong:** Generated TypeScript types get out of sync with the actual backend API.
**Why it happens:** Developer forgets to regenerate types after backend changes.
**How to avoid:**
- Add `api:generate` script to package.json
- Run type generation as part of the build process
- Include type generation in CI pipeline
- Consider a pre-commit hook or dev script that watches the OpenAPI spec
**Warning signs:** Runtime 400/422 errors that TypeScript didn't catch at compile time.

### Pitfall 7: Dashboard Layout Performance

**What goes wrong:** Dashboard with 8+ widgets becomes sluggish, especially during drag operations.
**Why it happens:** Every widget re-renders during drag, heavy widgets (charts, data grids) in layout.
**How to avoid:**
- Wrap each widget in `React.memo`
- Use `Suspense` for each widget's data loading
- Limit widget grid to 12 columns maximum
- Debounce layout change persistence (save to API/localStorage)
- Virtualize long lists within widgets
**Warning signs:** Jank during widget drag, layout change event firing on every pixel move.

## Code Examples

### electron.vite.config.ts (Full Configuration)

```typescript
// Source: electron-vite.org/config + @tailwindcss/vite docs
import { resolve } from 'path';
import { defineConfig, externalizeDepsPlugin } from 'electron-vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  main: {
    plugins: [externalizeDepsPlugin()],
    build: {
      outDir: 'out/main',
    },
  },
  preload: {
    plugins: [externalizeDepsPlugin()],
    build: {
      outDir: 'out/preload',
    },
  },
  renderer: {
    root: resolve('src/renderer'),
    build: {
      outDir: 'out/renderer',
      rollupOptions: {
        input: resolve('src/renderer/index.html'),
      },
    },
    resolve: {
      alias: {
        '@': resolve('src/renderer/src'),
      },
    },
    plugins: [react(), tailwindcss()],
  },
});
```

### Main Process BrowserWindow (Secure Defaults)

```typescript
// Source: Electron official docs - security tutorial
// src/main/index.ts
import { app, BrowserWindow, shell } from 'electron';
import { join } from 'path';
import { registerIPCHandlers } from './ipc';
import { setupTray } from './tray';

let mainWindow: BrowserWindow | null = null;

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1024,
    minHeight: 700,
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      contextIsolation: true,       // DEFAULT and REQUIRED
      nodeIntegration: false,        // DEFAULT and REQUIRED
      sandbox: true,                 // Additional isolation
      webSecurity: true,             // DEFAULT
    },
    // Frameless for custom title bar (optional)
    // frame: false,
    // titleBarStyle: 'hidden',
    show: false, // Show after ready-to-show for smoother startup
  });

  // Prevent navigation to external URLs
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });

  mainWindow.on('ready-to-show', () => {
    mainWindow?.show();
  });

  // Load renderer
  if (process.env.NODE_ENV === 'development') {
    mainWindow.loadURL('http://localhost:5173');
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'));
  }
}

app.whenReady().then(() => {
  registerIPCHandlers();
  createWindow();
  setupTray(mainWindow!);
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});
```

### Zustand Auth Store with IPC Persistence

```typescript
// Source: Zustand persist docs + Electron IPC pattern
// src/renderer/src/stores/auth.ts
import { create } from 'zustand';

interface User {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  role: string;
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;

  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<string | null>;
  initialize: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  isLoading: true,

  initialize: async () => {
    // On app start, check if we have stored tokens
    const tokens = await window.electronAPI.getStoredTokens();
    if (tokens) {
      set({ accessToken: tokens.accessToken, isAuthenticated: true, isLoading: false });
      // Fetch user profile
    } else {
      set({ isLoading: false });
    }
  },

  login: async (email, password) => {
    // Call API, store tokens via IPC, set state
  },

  logout: async () => {
    await window.electronAPI.clearTokens();
    set({ user: null, accessToken: null, isAuthenticated: false });
  },

  refreshToken: async () => {
    // Call refresh endpoint, update stored tokens via IPC
    return null;
  },
}));
```

### TanStack Query Provider with Offline Persistence

```typescript
// Source: TanStack Query docs + persist-client docs
// src/renderer/src/main.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PersistQueryClientProvider } from '@tanstack/react-query-persist-client';
import { createSyncStoragePersister } from '@tanstack/query-sync-storage-persister';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      gcTime: 1000 * 60 * 60 * 24, // 24 hours (for offline)
      retry: 2,
      refetchOnWindowFocus: true,
    },
  },
});

const persister = createSyncStoragePersister({
  storage: window.localStorage,
  key: 'kmuhub-query-cache',
});

export function App() {
  return (
    <PersistQueryClientProvider
      client={queryClient}
      persistOptions={{ persister, maxAge: 1000 * 60 * 60 * 24 }}
    >
      {/* Router, etc. */}
    </PersistQueryClientProvider>
  );
}
```

### Native Notification Integration (Main Process)

```typescript
// Source: Electron Notification API docs
// src/main/ipc/notifications.ts
import { ipcMain, Notification } from 'electron';

export function registerNotificationHandlers(): void {
  ipcMain.on('notification:show', (_event, { title, body }) => {
    if (Notification.isSupported()) {
      const notification = new Notification({
        title,
        body,
        silent: false,
      });
      notification.on('click', () => {
        // Focus the app window and navigate to relevant content
        // mainWindow.show(); mainWindow.focus();
      });
      notification.show();
    }
  });
}
```

### CORS Configuration Note

The backend CORS config (`CORSAllowedOrigins`) defaults to `http://localhost:3000`. For electron-vite dev server (port 5173) and production file:// protocol, update:
```bash
CORS_ALLOWED_ORIGINS="http://localhost:3000;http://localhost:5173;file://"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Create React App + Electron | electron-vite (Vite-based) | 2023-2024 | 10x faster HMR, proper main/preload/renderer separation |
| tailwind.config.js + PostCSS | @tailwindcss/vite plugin + CSS @import | Tailwind v4 (2025) | Zero config files, faster builds, CSS-first theming |
| nodeIntegration: true | contextIsolation + contextBridge | Electron 12+ (2021) | Mandatory for security; old tutorials are dangerous |
| Redux + redux-thunk | TanStack Query (server) + Zustand (client) | 2023-2025 | Less boilerplate, automatic caching, better offline support |
| Axios + manual types | openapi-fetch + openapi-typescript | 2024-2025 | Zero-runtime types, spec-driven, no drift |
| react-grid-layout 1.x (Flow) | react-grid-layout 2.x (TypeScript rewrite) | 2024 | Native TS, hooks API (useGridLayout), tree-shakeable |
| Component libraries (MUI) | shadcn/ui (copy-paste) | 2023-2025 | Full source ownership, Tailwind-native, smaller bundles |
| react-router v6 | react-router v7 | 2024-2025 | Lazy route discovery, framework mode, data loading |

**Deprecated/outdated:**
- `electron-webpack`: Unmaintained, replaced by electron-vite
- `@electron/remote`: Security risk, removed; use IPC instead
- `nodeIntegration: true`: Major security vulnerability
- `tailwind.config.js` in v4: Replaced by CSS-first `@theme` configuration
- `@types/react-grid-layout`: No longer needed with react-grid-layout v2 (bundled types)

## Electron-Specific Security Checklist

These must be enforced in every plan:

1. `contextIsolation: true` (default, never disable)
2. `nodeIntegration: false` (default, never enable)
3. `sandbox: true` (enable for additional isolation)
4. `webSecurity: true` (default, never disable)
5. CSP meta tag with strict policy (no unsafe-eval, no unsafe-inline)
6. All external links open via `shell.openExternal`, not in-app navigation
7. Auth tokens stored via `safeStorage` in main process, not localStorage
8. IPC channels expose specific operations, not raw ipcRenderer
9. Validate all IPC arguments in main process handlers
10. `setWindowOpenHandler` to prevent navigation to untrusted URLs

## Role-Based Dashboard Defaults (DESK-03)

The admin dashboard configuration needs a backend storage component. Approach:

- Default dashboard layouts are JSON configurations stored server-side per role
- Admin can modify role defaults via a settings page
- Users can personalize (override) their dashboard, stored in their user preferences
- Layout format: react-grid-layout Layout[] serialized as JSON
- Backend endpoint needed: `GET/PUT /api/v1/dashboard/defaults/{role}` and `GET/PUT /api/v1/dashboard/layout` (user's personal)
- This requires a small backend addition -- a `dashboard_layouts` table in the database
- Consider adding to existing CRM service or as a gateway-level feature with direct DB access

## Offline Caching Strategy (DESK-04)

For "basic offline functionality with local caching for recently accessed data":

1. **TanStack Query persistence** handles the heavy lifting: all queried data is serialized to localStorage automatically
2. **gcTime of 24 hours** means recently accessed contacts/deals/messages remain available offline for a full day
3. **Online/offline detection** via `navigator.onLine` + `window.addEventListener('online'/'offline')`
4. **Read-only when offline**: Users can VIEW cached data but mutations are blocked with a UI indicator
5. **No writes-while-offline queue** in v1: Complexity of conflict resolution is not worth it for "basic offline functionality"
6. **Cache size management**: localStorage has a ~5-10MB limit; monitor usage and evict oldest entries
7. **Future upgrade path**: If heavier offline is needed later, swap localStorage persister for IndexedDB persister (same TanStack Query API)

## Open Questions

Things that could not be fully resolved:

1. **react-grid-layout v2 + React 19 compatibility**
   - What we know: v2 was released recently with TypeScript rewrite and hooks API
   - What's unclear: Explicit React 19 peer dependency verification was not found
   - Recommendation: Test compatibility early in plan 05-01; if issues arise, fall back to v1.4 with @types/react-grid-layout

2. **Dashboard backend storage**
   - What we know: Need to store role-based default layouts and per-user overrides
   - What's unclear: Whether to add endpoints to existing CRM service or create a lightweight "preferences" service
   - Recommendation: Add to gateway with direct DB access (it already has a pool connection); simple key-value store in a `user_preferences` / `dashboard_defaults` table

3. **Electron auto-update strategy**
   - What we know: electron-builder supports auto-update via electron-updater
   - What's unclear: Whether to use GitHub releases, S3, or custom update server
   - Recommendation: Defer auto-update to a later phase; focus on manual distribution via electron-builder for now. The package.json already references electron-vite for builds.

4. **Custom window chrome (frameless window)**
   - What we know: Many modern Electron apps use frameless windows with custom title bars
   - What's unclear: Whether user wants native OS title bar or custom
   - Recommendation: Use native title bar for v1 (simpler, better OS integration). Custom chrome can be added later.

## Sources

### Primary (HIGH confidence)
- [electron-vite.org/guide](https://electron-vite.org/guide/) - Project structure, entry points, features
- [electron-vite.org/config](https://electron-vite.org/config/) - Configuration file documentation
- [electronjs.org/docs/latest/tutorial/context-isolation](https://www.electronjs.org/docs/latest/tutorial/context-isolation) - Context isolation security
- [electronjs.org/docs/latest/tutorial/ipc](https://www.electronjs.org/docs/latest/tutorial/ipc) - IPC patterns and best practices
- [electronjs.org/docs/latest/tutorial/performance](https://www.electronjs.org/docs/latest/tutorial/performance) - Performance optimization
- [electronjs.org/docs/latest/tutorial/security](https://www.electronjs.org/docs/latest/tutorial/security) - Security checklist
- [tailwindcss.com/blog/tailwindcss-v4](https://tailwindcss.com/blog/tailwindcss-v4) - Tailwind v4 changes
- [react.dev/reference/react/lazy](https://react.dev/reference/react/lazy) - React lazy loading
- [tanstack.com/query/latest](https://tanstack.com/query/latest) - TanStack Query features
- [openapi-ts.dev](https://openapi-ts.dev/) - openapi-typescript + openapi-fetch
- [zustand.docs.pmnd.rs](https://zustand.docs.pmnd.rs/integrations/persisting-store-data) - Zustand persist middleware
- [react-grid-layout GitHub](https://github.com/react-grid-layout/react-grid-layout) - v2 TypeScript rewrite

### Secondary (MEDIUM confidence)
- [blog.mohitnagaraj.in - electron-vite + shadcn setup](https://blog.mohitnagaraj.in/blog/202505/Electron_Shadcn_Guide) - Practical electron-vite + Tailwind v4 + shadcn/ui setup
- [remix.run/blog/faster-lazy-loading](https://remix.run/blog/faster-lazy-loading) - React Router v7.5 lazy loading improvements
- [ilert.com blog - react-grid-layout choice](https://www.ilert.com/blog/building-interactive-dashboards-why-react-grid-layout-was-our-best-choice) - Real-world dashboard implementation
- [ui.shadcn.com](https://ui.shadcn.com/) - shadcn/ui component library docs

### Tertiary (LOW confidence)
- [blog.logrocket.com - offline-first 2025](https://blog.logrocket.com/offline-first-frontend-apps-2025-indexeddb-sqlite/) - Offline storage comparison
- Various Medium articles on state management comparisons

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries verified against official docs, versions confirmed, existing package.json already declares core stack
- Architecture: HIGH - electron-vite project structure is well-documented, Electron IPC patterns are stable and official
- Pitfalls: HIGH - CSP/Tailwind issue specifically called out in project's CLAUDE.md, memory optimization documented by Electron officially
- Widget system: MEDIUM - react-grid-layout v2 is recent, React 19 compatibility needs runtime verification
- Offline strategy: MEDIUM - TanStack Query persistence is well-documented but localStorage limits for complex CRM data need real-world testing

**Research date:** 2026-02-07
**Valid until:** 2026-03-07 (30 days - Electron and React ecosystem stable)
