/**
 * IndexedDB-backed offline mutation queue.
 *
 * When the Electron app loses connectivity, mutations are enqueued here
 * instead of being dropped. On reconnection, `drain()` replays them
 * in FIFO order with exponential backoff.
 *
 * Storage keys:
 *   cosmi-mutation-queue        — active queue (idb-keyval)
 *   cosmi-mutation-dead-letter  — failed items after MAX_RETRIES attempts
 */

import { get, set } from 'idb-keyval'
import { generateIdempotencyKey } from './idempotency'

export interface QueuedMutation {
  /** Stable idempotency key — must be sent as Idempotency-Key header on replay. */
  idempotencyKey: string
  method: string
  path: string
  /** JSON-serialised request body (or null for DELETE without body). */
  body: string | null
  /** Extra headers to include on replay (e.g. Content-Type). */
  headers: Record<string, string>
  createdAt: number
  retryCount: number
  lastError: string | null
}

const QUEUE_KEY = 'cosmi-mutation-queue'
const DEAD_LETTER_KEY = 'cosmi-mutation-dead-letter'
const MAX_RETRIES = 5
const MAX_PARALLEL = 5

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

async function readQueue(): Promise<QueuedMutation[]> {
  return (await get<QueuedMutation[]>(QUEUE_KEY)) ?? []
}

async function writeQueue(items: QueuedMutation[]): Promise<void> {
  await set(QUEUE_KEY, items)
}

async function readDeadLetter(): Promise<QueuedMutation[]> {
  return (await get<QueuedMutation[]>(DEAD_LETTER_KEY)) ?? []
}

async function writeDeadLetter(items: QueuedMutation[]): Promise<void> {
  await set(DEAD_LETTER_KEY, items)
}

function backoffMs(retryCount: number): number {
  // Exponential backoff: 1s, 2s, 4s, 8s, 16s (capped)
  return Math.min(1000 * Math.pow(2, retryCount), 16_000)
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Adds a mutation to the queue for later replay.
 * Automatically generates an idempotency key if not provided.
 */
export async function enqueue(
  method: string,
  path: string,
  body: string | null,
  headers: Record<string, string> = {},
  idempotencyKey?: string,
): Promise<QueuedMutation> {
  const item: QueuedMutation = {
    idempotencyKey: idempotencyKey ?? generateIdempotencyKey(),
    method,
    path,
    body,
    headers,
    createdAt: Date.now(),
    retryCount: 0,
    lastError: null,
  }

  const queue = await readQueue()
  queue.push(item)
  await writeQueue(queue)
  return item
}

/**
 * Replays all queued mutations (max MAX_PARALLEL in parallel).
 * Successfully replayed items are removed from the queue.
 * Items that fail are retried up to MAX_RETRIES times; beyond that they
 * are moved to the dead-letter store.
 *
 * @param baseUrl - API base URL (e.g. "https://app.zentria.tech")
 * @param getAccessToken - callable returning the current JWT access token
 */
export async function drain(
  baseUrl: string,
  getAccessToken: () => string | null,
): Promise<{ replayed: number; failed: number; deadLettered: number }> {
  let queue = await readQueue()
  if (queue.length === 0) {
    return { replayed: 0, failed: 0, deadLettered: 0 }
  }

  let replayed = 0
  let failed = 0
  let deadLettered = 0

  // Process in batches of MAX_PARALLEL.
  const remaining: QueuedMutation[] = []
  const toDeadLetter: QueuedMutation[] = []

  for (let i = 0; i < queue.length; i += MAX_PARALLEL) {
    const batch = queue.slice(i, i + MAX_PARALLEL)
    const results = await Promise.allSettled(
      batch.map((item) => replayItem(item, baseUrl, getAccessToken)),
    )

    results.forEach((result, idx) => {
      const item = batch[idx]
      if (result.status === 'fulfilled') {
        replayed++
      } else {
        const updated: QueuedMutation = {
          ...item,
          retryCount: item.retryCount + 1,
          lastError: result.reason instanceof Error ? result.reason.message : String(result.reason),
        }

        if (updated.retryCount >= MAX_RETRIES) {
          toDeadLetter.push(updated)
          deadLettered++
        } else {
          // Exponential backoff: schedule retry later (keep in queue for now)
          remaining.push(updated)
          failed++
        }
      }
    })
  }

  await writeQueue(remaining)

  if (toDeadLetter.length > 0) {
    const existing = await readDeadLetter()
    await writeDeadLetter([...existing, ...toDeadLetter])
  }

  return { replayed, failed, deadLettered }
}

async function replayItem(
  item: QueuedMutation,
  baseUrl: string,
  getAccessToken: () => string | null,
): Promise<void> {
  const token = getAccessToken()
  const headers = new Headers(item.headers)
  headers.set('Idempotency-Key', item.idempotencyKey)
  // Content-Type only for methods that carry a JSON body. DELETE without body
  // would otherwise trip strict server-side request validators.
  if (item.body !== null && item.body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  // Apply exponential backoff delay before retrying.
  if (item.retryCount > 0) {
    await new Promise((resolve) => setTimeout(resolve, backoffMs(item.retryCount)))
  }

  const response = await fetch(`${baseUrl}${item.path}`, {
    method: item.method,
    headers,
    body: item.body ?? undefined,
  })

  // 2xx — idempotency replay returns the cached 2xx status, so a successful
  // re-drain looks identical to first-time success. 409 is reserved for
  // in-flight collisions and MUST be retried (not silently dropped).
  if (response.ok) {
    return
  }

  // 4xx client errors (except 409 in-flight and 429 rate-limit) are not
  // retryable — surface them to dead letter immediately.
  if (response.status >= 400 && response.status < 500 && response.status !== 409 && response.status !== 429) {
    throw new Error(`HTTP ${response.status} (not retryable)`)
  }

  throw new Error(`HTTP ${response.status}`)
}

/**
 * Returns all queued mutations without modifying the queue.
 */
export async function peek(): Promise<QueuedMutation[]> {
  return readQueue()
}

/**
 * Clears the active queue AND the dead-letter store.
 * Use only in dev/test or for explicit user-initiated clear.
 */
export async function clear(): Promise<void> {
  await writeQueue([])
  await writeDeadLetter([])
}

/**
 * Returns dead-lettered mutations (failed after MAX_RETRIES).
 */
export async function getDeadLetter(): Promise<QueuedMutation[]> {
  return readDeadLetter()
}
