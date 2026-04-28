/**
 * Idempotency key helpers for mutation requests.
 *
 * Each mutation that could be retried (offline queue, network retry) must
 * carry a stable Idempotency-Key so the backend can deduplicate it.
 */

/**
 * Generates a fresh idempotency key using the Web Crypto API.
 * The key is a UUID v4 prefixed with a timestamp so it is sortable and
 * gives the backend a hint about creation time without DB overhead.
 */
export function generateIdempotencyKey(): string {
  const ts = Date.now().toString(36)
  const rand = crypto.randomUUID()
  return `${ts}-${rand}`
}

/**
 * Returns a new Headers object with the Idempotency-Key header set.
 * If the headers already contain an Idempotency-Key it is left unchanged
 * (callers that pre-generate keys for the offline queue must not be overwritten).
 */
export function withIdempotencyKey(headers: HeadersInit = {}): Headers {
  const h = new Headers(headers)
  if (!h.has('Idempotency-Key')) {
    h.set('Idempotency-Key', generateIdempotencyKey())
  }
  return h
}
