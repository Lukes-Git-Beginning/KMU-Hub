/** Application-wide constants */

/**
 * Resolve the backend address.
 *
 * Order matters:
 * 1. An explicit RENDERER_VITE_API_URL always wins. The Electron builds set it
 *    at build time (.env.demo, .env.production) and must keep doing so -- they
 *    load from file://, so there is no origin to derive anything from.
 * 2. Otherwise, when the app was served over http(s), it came *from* the server
 *    it needs to talk to. Deriving the address instead of baking it in is what
 *    makes one build work for every customer server, rather than one build per
 *    customer -- see .planning/auslieferungsmodell-2026-08-12.md §3.1.
 * 3. Local fallback for anything else.
 */
function resolveApiBaseUrl(): string {
  const configured = import.meta.env.RENDERER_VITE_API_URL
  if (configured) return configured

  if (typeof window !== 'undefined' && window.location.protocol.startsWith('http')) {
    return window.location.origin
  }

  return 'http://localhost:8080'
}

/** Base URL for the backend API */
export const API_BASE_URL = resolveApiBaseUrl()

/**
 * Base URL under which public booking pages are served. These live on the
 * marketing site (zentria-website, `src/pages/book/[slug].astro`), not in the
 * product — so the address is configurable per installation rather than tied
 * to API_BASE_URL.
 */
export const BOOKING_BASE_URL =
  import.meta.env.RENDERER_VITE_BOOKING_URL || 'https://zentria.tech/book'

/** WebSocket reconnection delays in ms (exponential backoff, max 60s) */
export const WS_RECONNECT_DELAYS = [3000, 6000, 12000, 24000, 48000, 60000]

/** Maximum number of WebSocket reconnection attempts before giving up */
export const MAX_RECONNECT_ATTEMPTS = 10

/** React Query stale time: 5 minutes */
export const STALE_TIME = 5 * 60 * 1000

/** React Query garbage collection time: 24 hours (long for offline support) */
export const GC_TIME = 24 * 60 * 60 * 1000
