import { describe, it, expect } from 'vitest'
import { loginErrorKey } from '../auth'

// Every one of these statuses used to produce the same hardcoded English
// sentence, which made a wrong password, a rate limit and a dead backend
// indistinguishable — both for the user and for anyone debugging a failed login.
describe('loginErrorKey', () => {
  it.each([
    [401, 'auth.invalidCredentials'],
    [403, 'auth.accountDisabled'],
    [409, 'auth.twoFactorRequired'],
    [429, 'auth.rateLimited'],
    [500, 'auth.serverError'],
    [502, 'auth.serverError'],
    [503, 'auth.serverError'],
  ])('maps %i to %s', (status, expected) => {
    expect(loginErrorKey(status)).toBe(expected)
  })

  it('falls back to the generic key for statuses it does not know', () => {
    expect(loginErrorKey(418)).toBe('auth.loginFailed')
  })

  it('gives every status its own message', () => {
    const keys = [401, 403, 409, 429].map(loginErrorKey)
    expect(new Set(keys).size).toBe(keys.length)
  })
})
