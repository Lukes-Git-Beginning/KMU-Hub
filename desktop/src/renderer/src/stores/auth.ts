/**
 * Authentication store (Zustand).
 *
 * Manages user session, token persistence via Electron safeStorage,
 * and WebSocket connectivity lifecycle.
 */
import { create } from 'zustand'
import { apiClient } from '@/api/client'
import { wsManager } from '@/api/websocket'
import type { components } from '@/api/types'

type UserInfo = components['schemas']['UserInfo']

export interface User {
  id: string
  email: string
  firstName: string
  lastName: string
  roles: string[]
}

interface AuthState {
  user: User | null
  accessToken: string | null
  refreshTokenValue: string | null
  isAuthenticated: boolean
  isLoading: boolean

  /** Initialize auth state from stored tokens (call on app startup). */
  initialize: () => Promise<void>

  /** Log in with email and password. */
  login: (email: string, password: string) => Promise<void>

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
    firstName: info.firstName ?? '',
    lastName: info.lastName ?? '',
    roles: info.roles ?? [],
  }
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  accessToken: null,
  refreshTokenValue: null,
  isAuthenticated: false,
  isLoading: true,

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

      // Fetch user profile to validate the token is still valid
      const { data, error } = await apiClient.GET('/api/v1/auth/me')

      if (error || !data?.user) {
        // Token expired -- try refresh
        const newToken = await get().refreshToken()
        if (!newToken) {
          // Refresh also failed -- clean up
          await window.electronAPI.auth.clearTokens()
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
          set({
            user: null,
            accessToken: null,
            refreshTokenValue: null,
            isAuthenticated: false,
            isLoading: false,
          })
          return
        }

        set({ user: toUser(retryResult.data.user), isLoading: false })
      } else {
        set({ user: toUser(data.user), isLoading: false })
      }

      // Connect WebSocket after successful auth restoration
      const { accessToken } = get()
      if (accessToken) {
        wsManager.connect(accessToken)
      }
    } catch {
      set({
        user: null,
        accessToken: null,
        refreshTokenValue: null,
        isAuthenticated: false,
        isLoading: false,
      })
    }
  },

  async login(email: string, password: string) {
    const { data, error } = await apiClient.POST('/api/v1/auth/login', {
      body: { email, password },
    })

    if (error || !data) {
      throw new Error('Login failed. Please check your credentials.')
    }

    const accessToken = data.accessToken ?? null
    const refreshToken = data.refreshToken ?? null
    const user = data.user ? toUser(data.user) : null

    if (!accessToken || !refreshToken || !user) {
      throw new Error('Invalid server response.')
    }

    // Persist tokens via Electron safeStorage
    await window.electronAPI.auth.storeTokens({
      accessToken,
      refreshToken,
    })

    set({
      user,
      accessToken,
      refreshTokenValue: refreshToken,
      isAuthenticated: true,
      isLoading: false,
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

    try {
      const { data, error } = await apiClient.POST('/api/v1/auth/refresh', {
        body: { refresh_token: refreshTokenValue },
      })

      if (error || !data?.accessToken || !data?.refreshToken) {
        return null
      }

      const newAccessToken = data.accessToken
      const newRefreshToken = data.refreshToken

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
