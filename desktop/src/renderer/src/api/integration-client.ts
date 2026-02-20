/**
 * Integration API client for external platform connections.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  IntegrationConfig,
  CreateIntegrationRequest,
  UpdateIntegrationRequest,
  AccountLink,
  ChannelMapping,
  CreateChannelMappingRequest,
} from './integration-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

let refreshPromise: Promise<string | null> | null = null

async function getToken(): Promise<string | undefined> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

async function refreshTokenFn(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  const store = useAuthStore.getState()
  if (!refreshPromise) {
    refreshPromise = store.refreshToken().finally(() => { refreshPromise = null })
  }
  return refreshPromise
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const method = options.method ?? 'GET'
  if (!navigator.onLine && MUTATION_METHODS.has(method)) {
    throw new Error('Aenderungen sind offline nicht moeglich.')
  }

  const token = await getToken()
  const headers = new Headers(options.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }

  let res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })

  if (res.status === 401 && !path.includes('/auth/')) {
    const newToken = await refreshTokenFn()
    if (!newToken) {
      const { useAuthStore } = await import('@/stores/auth')
      useAuthStore.getState().logout()
      throw new Error('Session abgelaufen')
    }
    headers.set('Authorization', `Bearer ${newToken}`)
    res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error ?? body.message ?? `HTTP ${res.status}`)
  }

  if (res.status === 204) return {} as T
  return res.json() as Promise<T>
}

// ---------------------------------------------------------------------------
// Integration Configs
// ---------------------------------------------------------------------------

export const integrationConfigApi = {
  list() {
    return request<{ configs: IntegrationConfig[] }>('/api/v1/integrations/configs')
  },
  create(data: CreateIntegrationRequest) {
    return request<{ config: IntegrationConfig }>('/api/v1/integrations/configs', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  },
  update(id: string, data: UpdateIntegrationRequest) {
    return request<{ config: IntegrationConfig }>(`/api/v1/integrations/configs/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  },
  delete(id: string) {
    return request<Record<string, never>>(`/api/v1/integrations/configs/${id}`, {
      method: 'DELETE',
    })
  },
  test(id: string) {
    return request<{ success: boolean; message?: string }>(`/api/v1/integrations/configs/${id}/test`, {
      method: 'POST',
    })
  },
}

// ---------------------------------------------------------------------------
// Account Linking
// ---------------------------------------------------------------------------

export const accountLinkApi = {
  list() {
    return request<{ links: AccountLink[] }>('/api/v1/integrations/link')
  },
  link(platform: string, token: string) {
    return request<{ link: AccountLink }>('/api/v1/integrations/link', {
      method: 'POST',
      body: JSON.stringify({ platform, token }),
    })
  },
  unlink(platform: string) {
    return request<Record<string, never>>(`/api/v1/integrations/link/${platform}`, {
      method: 'DELETE',
    })
  },
}

// ---------------------------------------------------------------------------
// Channel Mapping
// ---------------------------------------------------------------------------

export const channelMappingApi = {
  list(integrationId: string) {
    return request<{ mappings: ChannelMapping[] }>(
      `/api/v1/integrations/configs/${integrationId}/mappings`,
    )
  },
  create(integrationId: string, data: CreateChannelMappingRequest) {
    return request<{ mapping: ChannelMapping }>(
      `/api/v1/integrations/configs/${integrationId}/mappings`,
      { method: 'POST', body: JSON.stringify(data) },
    )
  },
  delete(integrationId: string, mappingId: string) {
    return request<Record<string, never>>(
      `/api/v1/integrations/configs/${integrationId}/mappings/${mappingId}`,
      { method: 'DELETE' },
    )
  },
  toggleActive(integrationId: string, mappingId: string, isActive: boolean) {
    return request<{ mapping: ChannelMapping }>(
      `/api/v1/integrations/configs/${integrationId}/mappings/${mappingId}`,
      { method: 'PUT', body: JSON.stringify({ is_active: isActive }) },
    )
  },
}
