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

import { DEMO_MODE } from './demo-mode-flag'
export { DEMO_MODE }

// ── Intercept fetch at module scope ──────────────────────────────────
// Must run BEFORE any other module captures globalThis.fetch (e.g.
// openapi-fetch's createClient). Import this module first in main.tsx.

// Structurally compatible with msw's HttpHandler.run result.
type Handler = {
  run(info: { request: Request; requestId: string }): Promise<{ response?: Response } | null | undefined>
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

  // ── Intercept XMLHttpRequest as well ───────────────────────────────
  // Multipart uploads (documents, chat) use XHR for upload progress and
  // would bypass the fetch interceptor — route API XHRs through the same
  // handlers and synthesize the XHR lifecycle events.
  const origOpen = XMLHttpRequest.prototype.open
  const origSend = XMLHttpRequest.prototype.send
  const origSetHeader = XMLHttpRequest.prototype.setRequestHeader

  interface DemoXHR extends XMLHttpRequest {
    __demoMethod?: string
    __demoUrl?: string
    __demoHeaders?: Record<string, string>
  }

  XMLHttpRequest.prototype.open = function open(
    this: DemoXHR,
    method: string,
    url: string | URL,
    ...rest: [boolean?, (string | null)?, (string | null)?]
  ) {
    this.__demoMethod = method
    this.__demoUrl = typeof url === 'string' ? url : url.href
    return origOpen.call(this, method, url, ...(rest as [boolean]))
  } as typeof XMLHttpRequest.prototype.open

  XMLHttpRequest.prototype.setRequestHeader = function setRequestHeader(
    this: DemoXHR,
    name: string,
    value: string,
  ) {
    ;(this.__demoHeaders ??= {})[name] = value
    return origSetHeader.call(this, name, value)
  }

  XMLHttpRequest.prototype.send = function send(
    this: DemoXHR,
    body?: Document | XMLHttpRequestBodyInit | null,
  ) {
    const url = this.__demoUrl
    if (!url || !url.includes('/api/v1/')) {
      return origSend.call(this, body as XMLHttpRequestBodyInit | null)
    }
    // eslint-disable-next-line @typescript-eslint/no-this-alias -- capture the XHR instance for the detached async replay below
    const xhr = this
    void (async () => {
      if (_loadPromise) {
        await _loadPromise
        _loadPromise = null
      }
      const request = new Request(url, {
        method: xhr.__demoMethod ?? 'GET',
        headers: xhr.__demoHeaders,
        body: body as BodyInit | null,
      })
      for (const handler of _handlers) {
        const requestId = Math.random().toString(36).slice(2, 9)
        try {
          const result = await handler.run({ request: request.clone(), requestId })
          if (result?.response) {
            const text = await result.response.text()
            Object.defineProperty(xhr, 'readyState', { value: 4 })
            Object.defineProperty(xhr, 'status', { value: result.response.status })
            Object.defineProperty(xhr, 'statusText', { value: result.response.statusText })
            Object.defineProperty(xhr, 'responseText', { value: text })
            Object.defineProperty(xhr, 'response', { value: text })
            xhr.upload.dispatchEvent(
              new ProgressEvent('progress', { lengthComputable: true, loaded: 100, total: 100 }),
            )
            xhr.dispatchEvent(new Event('readystatechange'))
            xhr.dispatchEvent(new ProgressEvent('load'))
            xhr.dispatchEvent(new ProgressEvent('loadend'))
            return
          }
        } catch {
          // Handler didn't match — try next
        }
      }
      // No handler matched — pass through to the real network
      origSend.call(xhr, body as XMLHttpRequestBodyInit | null)
    })()
  }

  console.info('[Demo Mode] Fetch + XHR interceptors active — all API calls are mocked')
}

export function startDemoMode(): void {
  if (!DEMO_MODE) return

  // Clear stale query cache (now in IndexedDB) so fresh mock data is loaded
  import('idb-keyval').then(({ del }) => del('cosmi-query-cache')).catch(() => {})
}
