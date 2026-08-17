/**
 * Covers the web build's half of the platform accessors.
 *
 * test/setup.ts installs a window.electronAPI mock globally, so every other test
 * in the suite implicitly runs as if it were inside Electron. These tests delete
 * it first -- otherwise the browser paths added for the web build would never be
 * exercised anywhere.
 */
import { describe, it, expect, beforeEach, afterAll, vi } from 'vitest'
import { isElectron, tokenStore } from '../platform'

const originalBridge = window.electronAPI

function runningInBrowser() {
  // Assign undefined rather than delete: test/setup.ts defines the property
  // without `configurable`, so it cannot be removed. Equivalent for our checks,
  // which all go through `!== undefined` or optional chaining.
  ;(window as { electronAPI?: unknown }).electronAPI = undefined
}

beforeEach(() => {
  localStorage.clear()
  runningInBrowser()
})

afterAll(() => {
  ;(window as { electronAPI?: unknown }).electronAPI = originalBridge
})

describe('isElectron', () => {
  it('is false when no preload bridge is present', () => {
    expect(isElectron()).toBe(false)
  })

  it('is true once the bridge exists', () => {
    ;(window as { electronAPI?: unknown }).electronAPI = { auth: {} }
    expect(isElectron()).toBe(true)
  })
})

describe('tokenStore in the web build', () => {
  it('returns null when nothing was stored', async () => {
    await expect(tokenStore.getStoredTokens()).resolves.toBeNull()
  })

  it('round-trips a token pair', async () => {
    await tokenStore.storeTokens({ accessToken: 'a', refreshToken: 'r' })
    await expect(tokenStore.getStoredTokens()).resolves.toEqual({
      accessToken: 'a',
      refreshToken: 'r',
    })
  })

  it('forgets the pair after clearTokens', async () => {
    await tokenStore.storeTokens({ accessToken: 'a', refreshToken: 'r' })
    await tokenStore.clearTokens()
    await expect(tokenStore.getStoredTokens()).resolves.toBeNull()
  })

  // localStorage is user-writable. A tampered entry has to read as "logged out",
  // never as a pair whose members are undefined -- the API client would then send
  // "Bearer undefined" instead of redirecting to the login screen.
  it.each([
    ['not JSON at all', '}{'],
    ['a JSON primitive', '"hello"'],
    ['an object without tokens', '{}'],
    ['a missing refresh token', '{"accessToken":"a"}'],
    ['a missing access token', '{"refreshToken":"r"}'],
    ['non-string tokens', '{"accessToken":1,"refreshToken":2}'],
    ['null tokens', '{"accessToken":null,"refreshToken":null}'],
  ])('rejects %s', async (_label, raw) => {
    localStorage.setItem('cosmi-auth-tokens', raw)
    await expect(tokenStore.getStoredTokens()).resolves.toBeNull()
  })

  it('stays usable when localStorage throws', async () => {
    const setItem = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError')
    })
    // Must not reject: a full or disabled storage may cost persistence across a
    // reload, but it may not break the login that just succeeded.
    await expect(
      tokenStore.storeTokens({ accessToken: 'a', refreshToken: 'r' }),
    ).resolves.toBeUndefined()
    setItem.mockRestore()
  })
})

describe('tokenStore inside Electron', () => {
  it('delegates to the preload bridge and never touches localStorage', async () => {
    const bridge = {
      getStoredTokens: vi.fn().mockResolvedValue({ accessToken: 'x', refreshToken: 'y' }),
      storeTokens: vi.fn().mockResolvedValue(undefined),
      clearTokens: vi.fn().mockResolvedValue(undefined),
    }
    ;(window as { electronAPI?: unknown }).electronAPI = { auth: bridge }

    await tokenStore.storeTokens({ accessToken: 'x', refreshToken: 'y' })
    await expect(tokenStore.getStoredTokens()).resolves.toEqual({
      accessToken: 'x',
      refreshToken: 'y',
    })
    await tokenStore.clearTokens()

    expect(bridge.storeTokens).toHaveBeenCalledOnce()
    expect(bridge.getStoredTokens).toHaveBeenCalledOnce()
    expect(bridge.clearTokens).toHaveBeenCalledOnce()
    expect(localStorage.getItem('cosmi-auth-tokens')).toBeNull()
  })
})
