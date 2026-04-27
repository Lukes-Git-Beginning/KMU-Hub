/**
 * Authentication store (Zustand).
 *
 * Manages user session, token persistence via Electron safeStorage,
 * and WebSocket connectivity lifecycle.
 *
 * Offline behavior:
 * - initialize: loads cached user from localStorage when offline (skip API call)
 * - login: rejects immediately when offline (requires network)
 * - refreshToken: skips refresh when offline (use cached access token until online)
 */
import { create } from 'zustand'
import { apiClient } from '@/api/client'
import { wsManager } from '@/api/websocket'
import { validate2FALogin } from '@/api/security-client'
import type { components } from '@/api/types'

type UserInfo = components['schemas']['UserInfo']

/** localStorage key for cached user data (offline fallback). */
const CACHED_USER_KEY = 'cosmi-cached-user'

/** localStorage key for tenant-mode override (mock persistence until backend). */
const TENANT_MODE_KEY = 'cosmi:tenant:mode'

export type TenantMode = 'centralized' | 'distributed'

export interface Tenant {
  planType: 'cosmi' | 'orbit'
  mode: TenantMode
  employeeCount: number
}

export interface User {
  id: string
  email: string
  firstName: string
  lastName: string
  roles: string[]
}

interface AuthState {
  user: User | null
  tenant: Tenant
  accessToken: string | null
  refreshTokenValue: string | null
  isAuthenticated: boolean
  isLoading: boolean

  /** Pending 2FA token (set when login requires TOTP verification). */
  pendingToken: string | null

  /** Set tenant mode and persist to localStorage. */
  setTenantMode: (mode: TenantMode) => void

  /** Initialize auth state from stored tokens (call on app startup). */
  initialize: () => Promise<void>

  /** Register a new account. Automatically logs in on success. */
  register: (email: string, password: string, firstName: string, lastName: string) => Promise<void>

  /** Log in with email and password. Throws '2FA_REQUIRED' if 2FA needed. */
  login: (email: string, password: string) => Promise<void>

  /** Complete login by validating 2FA code with pending token. */
  complete2FALogin: (pendingToken: string, code: string) => Promise<void>

  /** Log out: revoke token, clear storage, disconnect WebSocket. */
  logout: () => Promise<void>

  /** Refresh the access token. Returns the new token or null on failure. */
  refreshToken: () => Promise<string | null>
}

/** Convert backend UserInfo to our User type. */
function toUser(info: UserInfo): User {
  return {
    id: info.id ?? '',
    email: info.email ?? '',
    firstName: info.first_name ?? '',
    lastName: info.last_name ?? '',
    roles: info.roles ?? [],
  }
}

/** Cache user data in localStorage for offline fallback. */
function cacheUser(user: User): void {
  try {
    localStorage.setItem(CACHED_USER_KEY, JSON.stringify(user))
  } catch {
    // QuotaExceededError or other storage error -- non-critical
  }
}

/** Load cached user data from localStorage. */
function loadCachedUser(): User | null {
  try {
    const raw = localStorage.getItem(CACHED_USER_KEY)
    if (!raw) return null
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

/** Clear cached user data from localStorage. */
function clearCachedUser(): void {
  try {
    localStorage.removeItem(CACHED_USER_KEY)
  } catch {
    // Non-critical
  }
}

/** Read tenant mode: localStorage override first, then default distributed.
 *  Distributed = Settings nur PERSÖNLICH, alle Admin-Funktionen im Admin-Modul links unten. */
function loadTenantMode(): TenantMode {
  try {
    const stored = localStorage.getItem(TENANT_MODE_KEY)
    if (stored === 'distributed' || stored === 'centralized') return stored
  } catch {
    // Non-critical
  }
  return 'distributed'
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  tenant: {
    planType: 'cosmi',
    mode: loadTenantMode(),
    employeeCount: 0,
  },
  accessToken: null,
  refreshTokenValue: null,
  isAuthenticated: false,
  isLoading: true,
  pendingToken: null,

  setTenantMode(mode: TenantMode) {
    try {
      localStorage.setItem(TENANT_MODE_KEY, mode)
    } catch {
      // Non-critical
    }
    set((s) => ({ tenant: { ...s.tenant, mode } }))
  },

  async initialize() {
    set({ isLoading: true })

    try {
      const stored = await window.electronAPI.auth.getStoredTokens()
      if (!stored) {
        set({ isLoading: false })
        return
      }

      // Set tokens first so the API client middleware can use them
      set({
        accessToken: stored.accessToken,
        refreshTokenValue: stored.refreshToken,
        isAuthenticated: true,
      })

      // If offline, use cached user data instead of making API call
      if (!navigator.onLine) {
        const cachedUser = loadCachedUser()
        if (cachedUser) {
          set({ user: cachedUser, isLoading: false })
          return
        }
        // No cached user but have tokens -- still mark authenticated,
        // user profile will load when back online
        set({ isLoading: false })
        return
      }

      // Fetch user profile to validate the token is still valid
      const { data, error } = await apiClient.GET('/api/v1/auth/me')

      if (error || !data?.user) {
        // Token expired -- try refresh
        const newToken = await get().refreshToken()
        if (!newToken) {
          // Refresh also failed -- clean up
          await window.electronAPI.auth.clearTokens()
          clearCachedUser()
          set({
            user: null,
            accessToken: null,
            refreshTokenValue: null,
            isAuthenticated: false,
            isLoading: false,
          })
          return
        }

        // Retry profile fetch with new token
        const retryResult = await apiClient.GET('/api/v1/auth/me')
        if (retryResult.error || !retryResult.data?.user) {
          await window.electronAPI.auth.clearTokens()
          clearCachedUser()
          set({
            user: null,
            accessToken: null,
            refreshTokenValue: null,
            isAuthenticated: false,
            isLoading: false,
          })
          return
        }

        const user = toUser(retryResult.data.user)
        cacheUser(user)
        set({ user, isLoading: false })
      } else {
        const user = toUser(data.user)
        cacheUser(user)
        set({ user, isLoading: false })
      }

      // Connect WebSocket after successful auth restoration
      const { accessToken } = get()
      if (accessToken) {
        wsManager.connect(accessToken)
      }
    } catch {
      // If offline and we have tokens, try cached user as fallback
      if (!navigator.onLine) {
        const cachedUser = loadCachedUser()
        if (cachedUser) {
          set({ user: cachedUser, isLoading: false })
          return
        }
      }

      set({
        user: null,
        accessToken: null,
        refreshTokenValue: null,
        isAuthenticated: false,
        isLoading: false,
      })
    }
  },

  async register(email: string, password: string, firstName: string, lastName: string) {
    if (!navigator.onLine) {
      throw new Error('Registrierung erfordert eine Internetverbindung.')
    }

    const { data, error } = await apiClient.POST('/api/v1/auth/register', {
      body: { email, password, first_name: firstName, last_name: lastName },
    })

    if (error || !data) {
      const msg = (error as Record<string, unknown>)?.message
      throw new Error(typeof msg === 'string' ? msg : 'Registrierung fehlgeschlagen.')
    }

    const accessToken = data.access_token ?? null
    const refreshToken = data.refresh_token ?? null
    const user = data.user ? toUser(data.user) : null

    if (!accessToken || !refreshToken || !user) {
      throw new Error('Invalid server response.')
    }

    await window.electronAPI.auth.storeTokens({ accessToken, refreshToken })
    cacheUser(user)

    set({
      user,
      accessToken,
      refreshTokenValue: refreshToken,
      isAuthenticated: true,
      isLoading: false,
      pendingToken: null,
    })

    wsManager.connect(accessToken)
  },

  async login(email: string, password: string) {
    // Login requires a network connection
    if (!navigator.onLine) {
      throw new Error('Anmeldung erfordert eine Internetverbindung.')
    }

    const { data, error } = await apiClient.POST('/api/v1/auth/login', {
      body: { email, password },
    })

    if (error || !data) {
      throw new Error('Login failed. Please check your credentials.')
    }

    // Check if 2FA is required (backend returns pending_token instead of tokens)
    const requiresTwoFactor = (data as Record<string, unknown>).requires_two_factor === true
    if (requiresTwoFactor) {
      const pendingToken = (data as Record<string, unknown>).pending_token as string | null
      set({ pendingToken })
      throw new Error('2FA_REQUIRED')
    }

    const accessToken = data.access_token ?? null
    const refreshToken = data.refresh_token ?? null
    const user = data.user ? toUser(data.user) : null

    if (!accessToken || !refreshToken || !user) {
      throw new Error('Invalid server response.')
    }

    // Persist tokens via Electron safeStorage
    await window.electronAPI.auth.storeTokens({
      accessToken,
      refreshToken,
    })

    // Cache user for offline access
    cacheUser(user)

    set({
      user,
      accessToken,
      refreshTokenValue: refreshToken,
      isAuthenticated: true,
      isLoading: false,
      pendingToken: null,
    })

    // Establish WebSocket connection
    wsManager.connect(accessToken)
  },

  async complete2FALogin(pendingToken: string, code: string) {
    if (!navigator.onLine) {
      throw new Error('Anmeldung erfordert eine Internetverbindung.')
    }

    const result = await validate2FALogin(pendingToken, code)

    const accessToken = result.access_token
    const refreshToken = result.refresh_token
    const user: User = {
      id: result.user.id,
      email: result.user.email,
      firstName: result.user.first_name,
      lastName: result.user.last_name,
      roles: result.user.roles,
    }

    // Persist tokens via Electron safeStorage
    await window.electronAPI.auth.storeTokens({
      accessToken,
      refreshToken,
    })

    // Cache user for offline access
    cacheUser(user)

    set({
      user,
      accessToken,
      refreshTokenValue: refreshToken,
      isAuthenticated: true,
      isLoading: false,
      pendingToken: null,
    })

    // Establish WebSocket connection
    wsManager.connect(accessToken)
  },

  async logout() {
    const { refreshTokenValue } = get()

    // Best-effort server-side logout
    try {
      if (refreshTokenValue) {
        await apiClient.POST('/api/v1/auth/logout', {
          body: { refresh_token: refreshTokenValue },
        })
      }
    } catch {
      // Ignore -- we're logging out anyway
    }

    // Disconnect WebSocket
    wsManager.disconnect()

    // Clear persisted tokens
    await window.electronAPI.auth.clearTokens()

    // Clear cached user
    clearCachedUser()

    set({
      user: null,
      accessToken: null,
      refreshTokenValue: null,
      isAuthenticated: false,
      isLoading: false,
    })
  },

  async refreshToken() {
    const { refreshTokenValue } = get()
    if (!refreshTokenValue) return null

    // Skip refresh when offline -- use cached access token until online
    if (!navigator.onLine) return null

    try {
      const { data, error } = await apiClient.POST('/api/v1/auth/refresh', {
        body: { refresh_token: refreshTokenValue },
      })

      if (error || !data?.access_token || !data?.refresh_token) {
        return null
      }

      const newAccessToken = data.access_token
      const newRefreshToken = data.refresh_token

      // Persist updated tokens
      await window.electronAPI.auth.storeTokens({
        accessToken: newAccessToken,
        refreshToken: newRefreshToken,
      })

      set({
        accessToken: newAccessToken,
        refreshTokenValue: newRefreshToken,
      })

      return newAccessToken
    } catch {
      return null
    }
  },
}))
