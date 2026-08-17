/**
 * Compose-draft handover.
 *
 * This replaced a Zustand store that could not do the job: the pop-out compose
 * window is a separate renderer process, so module state never crossed over and
 * the draft was silently dropped. These tests pin the behaviour that fixes it.
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  stashComposeDraft,
  takeComposeDraft,
  clearComposeDraft,
  type ComposeDraft,
} from '../mails'

const KEY = 'cosmi:mails:compose-handover'

function draft(overrides: Partial<ComposeDraft> = {}): ComposeDraft {
  return {
    to: ['a@example.com'],
    cc: [],
    bcc: [],
    subject: 'Angebot',
    body: '<p>Hallo</p>',
    mode: 'compose',
    ...overrides,
  }
}

beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('compose draft handover', () => {
  it('hands a draft to the window that opens next', () => {
    stashComposeDraft(draft({ subject: 'Rechnung' }))
    expect(takeComposeDraft()?.subject).toBe('Rechnung')
  })

  it('preserves reply threading and the sending account', () => {
    stashComposeDraft(draft({ mode: 'reply', replyToMessageId: 'msg-1', accountId: 'acc-1' }))
    const taken = takeComposeDraft()
    expect(taken?.mode).toBe('reply')
    expect(taken?.replyToMessageId).toBe('msg-1')
    expect(taken?.accountId).toBe('acc-1')
  })

  // Consuming on read is what keeps a reload of the compose window from
  // resurrecting a draft the user already sent.
  it('can only be taken once', () => {
    stashComposeDraft(draft())
    expect(takeComposeDraft()).not.toBeNull()
    expect(takeComposeDraft()).toBeNull()
  })

  it('returns null when nothing was stashed', () => {
    expect(takeComposeDraft()).toBeNull()
  })

  it('drops a draft older than the TTL', () => {
    vi.useFakeTimers()
    stashComposeDraft(draft())
    vi.advanceTimersByTime(31_000)
    expect(takeComposeDraft()).toBeNull()
  })

  it('still accepts a draft within the TTL', () => {
    vi.useFakeTimers()
    stashComposeDraft(draft())
    vi.advanceTimersByTime(5_000)
    expect(takeComposeDraft()).not.toBeNull()
  })

  it('clearComposeDraft discards without handing anything over', () => {
    stashComposeDraft(draft())
    clearComposeDraft()
    expect(takeComposeDraft()).toBeNull()
  })

  // The compose form calls .length on to/cc/bcc immediately, so a tampered entry
  // must read as "no draft" rather than reaching the form with undefined arrays.
  it.each([
    ['not JSON', '}{'],
    ['no envelope timestamp', '{"draft":{"to":[],"cc":[],"bcc":[],"subject":"","body":""}}'],
    ['no draft at all', '{"at":' + Date.now() + '}'],
    ['recipients that are not arrays', '{"at":' + Date.now() + ',"draft":{"to":"x","cc":[],"bcc":[],"subject":"","body":""}}'],
    ['a missing body', '{"at":' + Date.now() + ',"draft":{"to":[],"cc":[],"bcc":[],"subject":""}}'],
  ])('rejects %s', (_label, raw) => {
    localStorage.setItem(KEY, raw)
    expect(takeComposeDraft()).toBeNull()
  })
})
