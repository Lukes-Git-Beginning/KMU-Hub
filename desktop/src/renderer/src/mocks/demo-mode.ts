/**
 * Demo Mode Bootstrap — intercepts all API calls with realistic mock data.
 * Activated via RENDERER_VITE_DEMO_MODE=true.
 *
 * Uses a direct fetch interceptor instead of MSW Service Worker because
 * Electron production builds use file:// protocol which doesn't support
 * Service Workers.
 *
 * Handlers are dynamically imported so Vite code-splits them into a
 * separate chunk. When DEMO_MODE is false, the dynamic import() is
 * dead-code-eliminated and the mock data never enters the prod bundle.
 */

export const DEMO_MODE = import.meta.env.RENDERER_VITE_DEMO_MODE === 'true'

// ── Intercept fetch at module scope ──────────────────────────────────
// Must run BEFORE any other module captures globalThis.fetch (e.g.
// openapi-fetch's createClient). Import this module first in main.tsx.

type Handler = {
  run(info: { request: Request; requestId: string }): Promise<{ response: Response } | undefined>
}

let _handlers: Handler[] = []
let _loadPromise: Promise<void> | null = null

if (DEMO_MODE) {
  _loadPromise = import('./handlers').then((m) => {
    _handlers = m.handlers
  })

  const originalFetch = window.fetch.bind(window)

  window.fetch = async function interceptedFetch(
    input: RequestInfo | URL,
    init?: RequestInit,
  ): Promise<Response> {
    if (_loadPromise) {
      await _loadPromise
      _loadPromise = null
    }

    const request = new Request(
      input instanceof URL ? input.href : input,
      init,
    )

    for (const handler of _handlers) {
      const requestId = Math.random().toString(36).slice(2, 9)
      try {
        const result = await handler.run({ request: request.clone(), requestId })
        if (result?.response) {
          return result.response
        }
      } catch {
        // Handler didn't match — try next
      }
    }

    // No handler matched — pass through to real fetch
    return originalFetch(input, init)
  }

  console.info('[Demo Mode] Fetch interceptor active — all API calls are mocked')
}

export function startDemoMode(): void {
  if (!DEMO_MODE) return

  // Clear stale query cache (now in IndexedDB) so fresh mock data is loaded
  import('idb-keyval').then(({ del }) => del('cosmi-query-cache')).catch(() => {})
}
