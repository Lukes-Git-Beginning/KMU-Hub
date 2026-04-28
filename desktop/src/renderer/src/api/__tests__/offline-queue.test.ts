/**
 * Tests for the IndexedDB-backed offline mutation queue.
 *
 * Uses fake-indexeddb to simulate IndexedDB in jsdom without a real browser.
 */
import 'fake-indexeddb/auto'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { enqueue, drain, peek, clear, getDeadLetter } from '../offline-queue'

// Reset IndexedDB state between tests via the queue's own clear() helper.
beforeEach(async () => {
  await clear()
  vi.useRealTimers()
})

describe('enqueue', () => {
  it('adds an item to the queue', async () => {
    await enqueue('POST', '/api/v1/crm/contacts', '{"name":"Test"}')
    const queue = await peek()
    expect(queue).toHaveLength(1)
    expect(queue[0].method).toBe('POST')
    expect(queue[0].path).toBe('/api/v1/crm/contacts')
    expect(queue[0].body).toBe('{"name":"Test"}')
    expect(queue[0].retryCount).toBe(0)
  })

  it('assigns a unique idempotency key per item', async () => {
    await enqueue('POST', '/api/v1/crm/contacts', null)
    await enqueue('POST', '/api/v1/crm/contacts', null)
    const queue = await peek()
    expect(queue[0].idempotencyKey).not.toBe(queue[1].idempotencyKey)
  })

  it('uses provided idempotency key when given', async () => {
    await enqueue('POST', '/api/v1/test', null, {}, 'my-stable-key')
    const queue = await peek()
    expect(queue[0].idempotencyKey).toBe('my-stable-key')
  })

  it('persists multiple items', async () => {
    await enqueue('POST', '/api/v1/a', null)
    await enqueue('PUT', '/api/v1/b', '{"x":1}')
    await enqueue('DELETE', '/api/v1/c', null)
    const queue = await peek()
    expect(queue).toHaveLength(3)
  })
})

describe('drain', () => {
  it('successfully replays and removes items from queue', async () => {
    await enqueue('POST', '/api/v1/crm/contacts', '{"name":"Test"}')

    const mockFetch = vi.fn().mockResolvedValue({ ok: true, status: 201 })
    vi.stubGlobal('fetch', mockFetch)

    const result = await drain('https://app.zentria.tech', () => 'token-abc')

    expect(result.replayed).toBe(1)
    expect(result.failed).toBe(0)
    expect(result.deadLettered).toBe(0)

    const remaining = await peek()
    expect(remaining).toHaveLength(0)

    vi.unstubAllGlobals()
  })

  it('keeps items in queue on transient server error', async () => {
    await enqueue('POST', '/api/v1/test', null)

    const mockFetch = vi.fn().mockResolvedValue({ ok: false, status: 503 })
    vi.stubGlobal('fetch', mockFetch)

    const result = await drain('https://app.zentria.tech', () => 'token')

    expect(result.failed).toBe(1)
    expect(result.replayed).toBe(0)

    const remaining = await peek()
    expect(remaining).toHaveLength(1)
    expect(remaining[0].retryCount).toBe(1)
    expect(remaining[0].lastError).toContain('503')

    vi.unstubAllGlobals()
  })

  it('retries on 409 (idempotency in-flight, not success) without dropping the item', async () => {
    // 409 from the idempotency middleware means a parallel request holding the
    // same key is still in-flight. The drain MUST retry, not silently drop:
    // the cached response (after the in-flight call completes) returns 2xx.
    await enqueue('POST', '/api/v1/test', null)

    const mockFetch = vi.fn().mockResolvedValue({ ok: false, status: 409 })
    vi.stubGlobal('fetch', mockFetch)

    const result = await drain('https://app.zentria.tech', () => 'token')
    expect(result.replayed).toBe(0)
    expect(result.failed).toBe(1)

    // Item remains queued for the next drain cycle.
    const remaining = await peek()
    expect(remaining).toHaveLength(1)
    expect(remaining[0].retryCount).toBe(1)

    vi.unstubAllGlobals()
  })

  it('sends Idempotency-Key header on replay', async () => {
    await enqueue('POST', '/api/v1/test', null, {}, 'stable-key-123')

    const capturedHeaders: Headers[] = []
    const mockFetch = vi.fn().mockImplementation((_url: string, opts: RequestInit) => {
      capturedHeaders.push(opts.headers as Headers)
      return Promise.resolve({ ok: true, status: 200 })
    })
    vi.stubGlobal('fetch', mockFetch)

    await drain('https://app.zentria.tech', () => 'token')

    expect(capturedHeaders).toHaveLength(1)
    const headers = capturedHeaders[0] as Headers
    expect(headers.get('Idempotency-Key')).toBe('stable-key-123')

    vi.unstubAllGlobals()
  })
})

describe('dead-letter', () => {
  it('moves item to dead-letter after MAX_RETRIES failures', async () => {
    const { set } = await import('idb-keyval')

    // Pre-populate at retryCount=4 (one away from dead-letter).
    // Stub setTimeout so the 16s backoffMs(4) resolves instantly.
    vi.stubGlobal('setTimeout', (fn: () => void) => { fn(); return 0 })

    await enqueue('POST', '/api/v1/test', null, {}, 'dl-key')
    let queue = await peek()
    queue[0].retryCount = 4
    await set('cosmi-mutation-queue', queue)

    const mockFetch = vi.fn().mockResolvedValue({ ok: false, status: 500 })
    vi.stubGlobal('fetch', mockFetch)

    const result = await drain('https://app.zentria.tech', () => 'token')

    expect(result.deadLettered).toBe(1)
    expect(result.failed).toBe(0)

    const remaining = await peek()
    expect(remaining).toHaveLength(0)

    const deadLetter = await getDeadLetter()
    expect(deadLetter).toHaveLength(1)
    expect(deadLetter[0].idempotencyKey).toBe('dl-key')
    expect(deadLetter[0].retryCount).toBe(5)

    vi.unstubAllGlobals()
  })

  it('accumulates dead-letter items across multiple drain calls', async () => {
    const { set } = await import('idb-keyval')
    vi.stubGlobal('setTimeout', (fn: () => void) => { fn(); return 0 })

    for (let i = 0; i < 2; i++) {
      await enqueue('POST', `/api/v1/test${i}`, null, {}, `dl-key-${i}`)
      let queue = await peek()
      queue = queue.map((item) => ({ ...item, retryCount: 4 }))
      await set('cosmi-mutation-queue', queue)

      vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status: 500 }))
      await drain('https://app.zentria.tech', () => 'token')
    }

    vi.unstubAllGlobals()

    const deadLetter = await getDeadLetter()
    expect(deadLetter).toHaveLength(2)
  })
})

describe('clear', () => {
  it('empties both queue and dead-letter', async () => {
    await enqueue('POST', '/api/v1/test', null)
    await clear()
    expect(await peek()).toHaveLength(0)
    expect(await getDeadLetter()).toHaveLength(0)
  })
})
