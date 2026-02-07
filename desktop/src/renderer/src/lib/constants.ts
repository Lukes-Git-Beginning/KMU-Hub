/** Application-wide constants */

/** Base URL for the backend API */
export const API_BASE_URL =
  import.meta.env.RENDERER_VITE_API_URL || 'http://localhost:8080'

/** WebSocket reconnection delays in ms (exponential backoff, max 60s) */
export const WS_RECONNECT_DELAYS = [3000, 6000, 12000, 24000, 48000, 60000]

/** Maximum number of WebSocket reconnection attempts before giving up */
export const MAX_RECONNECT_ATTEMPTS = 10

/** React Query stale time: 5 minutes */
export const STALE_TIME = 5 * 60 * 1000

/** React Query garbage collection time: 24 hours (long for offline support) */
export const GC_TIME = 24 * 60 * 60 * 1000
