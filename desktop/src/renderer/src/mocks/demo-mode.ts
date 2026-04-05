/**
 * Demo Mode Bootstrap — intercepts all API calls with realistic mock data.
 * Activated via RENDERER_VITE_DEMO_MODE=true.
 *
 * Uses a direct fetch interceptor instead of MSW Service Worker because
 * Electron production builds use file:// protocol which doesn't support
 * Service Workers or dynamic import().
 *
 * Handlers are statically imported so they work without dynamic import().
 * In non-demo production builds, Vite dead-code eliminates everything
 * after the early return.
 */

import { handlers } from './handlers'

export const DEMO_MODE = import.meta.env.RENDERER_VITE_DEMO_MODE === 'true'

// ── Intercept fetch at module scope ──────────────────────────────────
// Must run BEFORE any other module captures globalThis.fetch (e.g.
// openapi-fetch's createClient). Import this module first in main.tsx.
if (DEMO_MODE) {
  const originalFetch = window.fetch.bind(window)

  window.fetch = async function interceptedFetch(
    input: RequestInfo | URL,
    init?: RequestInit,
  ): Promise<Response> {
    const request = new Request(
      input instanceof URL ? input.href : input,
      init,
    )

    for (const handler of handlers) {
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

  // Clear stale query cache so fresh mock data is loaded
  try {
    localStorage.removeItem('kmuhub-query-cache')
  } catch {
    // localStorage may be unavailable
  }
}
