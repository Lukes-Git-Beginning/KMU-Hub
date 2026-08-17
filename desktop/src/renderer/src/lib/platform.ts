/**
 * Platform detection and the storage backends that differ between the Electron
 * shell and the plain web build.
 *
 * The same renderer ships twice: inside Electron, where a preload script exposes
 * window.electronAPI, and as a web app served over HTTP, where it does not. Every
 * feature that only one of them can do belongs behind an accessor here, so call
 * sites stay free of `window.electronAPI?.` chains that silently no-op.
 */
import type { ElectronAPI, TokenPair } from '../../../preload/types'

/**
 * True when running inside the Electron shell, i.e. the preload bridge is present.
 *
 * Checks the bridge rather than a user-agent string: the bridge is what call sites
 * actually need, and contextIsolation can be configured such that a UA sniff and
 * the real capability disagree.
 */
export function isElectron(): boolean {
  return typeof window !== 'undefined' && window.electronAPI !== undefined
}

/** localStorage key holding the token pair in the web build. */
const WEB_TOKEN_KEY = 'cosmi-auth-tokens'

/**
 * Token persistence across restarts.
 *
 * Electron encrypts the pair with safeStorage and writes it to userData. The web
 * build has no equivalent and uses localStorage, which is readable by any script
 * that runs on the origin -- the strict CSP in index.html is what keeps foreign
 * script out, so treat that CSP as part of this decision, not as unrelated
 * hardening.
 *
 * Both tokens are stored, not just the refresh token. Keeping the access token in
 * memory only sounds stricter but buys nothing: an attacker holding the refresh
 * token mints a fresh access token whenever they like. It would only add a
 * guaranteed 401 on every cold start, since initialize() would always begin with
 * an empty access token.
 *
 * Typed as ElectronAPI['auth'] on purpose -- the compiler then rejects any drift
 * between the two backends.
 */
export const tokenStore: ElectronAPI['auth'] = {
  async getStoredTokens(): Promise<TokenPair | null> {
    const bridge = window.electronAPI?.auth
    if (bridge) return bridge.getStoredTokens()

    try {
      const raw = localStorage.getItem(WEB_TOKEN_KEY)
      if (!raw) return null
      // localStorage is user-writable, so treat the payload as untrusted: a
      // half-written or hand-edited entry must read as "not logged in", never as
      // a token pair with undefined members.
      const parsed = JSON.parse(raw) as Partial<TokenPair>
      if (typeof parsed?.accessToken !== 'string' || typeof parsed?.refreshToken !== 'string') {
        return null
      }
      return { accessToken: parsed.accessToken, refreshToken: parsed.refreshToken }
    } catch {
      // Malformed JSON, or storage blocked entirely (private mode, cookies off).
      return null
    }
  },

  async storeTokens(tokens: TokenPair): Promise<void> {
    const bridge = window.electronAPI?.auth
    if (bridge) return bridge.storeTokens(tokens)

    try {
      localStorage.setItem(WEB_TOKEN_KEY, JSON.stringify(tokens))
    } catch {
      // Quota exceeded or storage unavailable. The session still works from
      // memory; it just will not survive a reload.
    }
  },

  async clearTokens(): Promise<void> {
    const bridge = window.electronAPI?.auth
    if (bridge) return bridge.clearTokens()

    try {
      localStorage.removeItem(WEB_TOKEN_KEY)
    } catch {
      // Non-critical -- the in-memory state is cleared by the caller regardless.
    }
  },
}
