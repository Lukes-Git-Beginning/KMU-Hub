/**
 * Integration API client for Teams/Slack configuration, channel mappings,
 * and account linking.
 *
 * Follows the caldav-client.ts / automation-client.ts pattern: typed fetch
 * helper with auth header injection and 401 retry.
 *
 * Gateway routes: /api/v1/integrations/*
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  Platform,
  IntegrationConfigResponse,
  IntegrationConfigListResponse,
  CreateIntegrationConfigRequest,
  UpdateIntegrationConfigRequest,
  ChannelMappingResponse,
  ChannelMappingListResponse,
  CreateChannelMappingRequest,
  UpdateChannelMappingRequest,
  LinkAccountRequest,
  LinkAccountResponse,
  AccountLinkStatus,
  TestNotificationResponse,
} from './integration-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

class OfflineError extends Error {
  constructor() {
    super('Änderungen sind offline nicht möglich.')
    this.name = 'OfflineError'
  }
}

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new OfflineError()
  }

  const url = `${API_BASE_URL}${opts.path}`
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = {
    method: opts.method,
    headers,
  }

  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const res = await fetch(url, init)

  if (!res.ok) {
    if (res.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryRes = await fetch(url, { ...init, headers })
        if (!retryRes.ok) {
          const err = await retryRes.json().catch(() => ({}))
          throw new Error(
            (err as Record<string, string>).error ||
              `Request failed: ${retryRes.status}`,
          )
        }
        if (retryRes.status === 204) return {} as T
        return retryRes.json() as Promise<T>
      }

      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await res.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${res.status}`,
    )
  }

  if (res.status === 204) return {} as T
  return res.json() as Promise<T>
}

// ---------------------------------------------------------------------------
// Integration configs (admin)
// ---------------------------------------------------------------------------

export function listConfigs() {
  return request<IntegrationConfigListResponse>({
    method: 'GET',
    path: '/api/v1/integrations/configs',
  })
}

export function getConfig(platform: Platform) {
  return request<IntegrationConfigResponse>({
    method: 'GET',
    path: `/api/v1/integrations/configs/${platform}`,
  })
}

export function createConfig(data: CreateIntegrationConfigRequest) {
  return request<IntegrationConfigResponse>({
    method: 'POST',
    path: '/api/v1/integrations/configs',
    body: data,
  })
}

export function updateConfig(
  platform: Platform,
  data: UpdateIntegrationConfigRequest,
) {
  return request<IntegrationConfigResponse>({
    method: 'PUT',
    path: `/api/v1/integrations/configs/${platform}`,
    body: data,
  })
}

export function deleteConfig(platform: Platform) {
  return request<{ status: string }>({
    method: 'DELETE',
    path: `/api/v1/integrations/configs/${platform}`,
  })
}

export function testConfig(platform: Platform) {
  return request<TestNotificationResponse>({
    method: 'POST',
    path: `/api/v1/integrations/configs/${platform}/test`,
  })
}

// ---------------------------------------------------------------------------
// Channel mappings (admin)
// ---------------------------------------------------------------------------

export function listMappings(platform: Platform) {
  return request<ChannelMappingListResponse>({
    method: 'GET',
    path: `/api/v1/integrations/configs/${platform}/mappings`,
  })
}

export function createMapping(
  platform: Platform,
  data: CreateChannelMappingRequest,
) {
  return request<ChannelMappingResponse>({
    method: 'POST',
    path: `/api/v1/integrations/configs/${platform}/mappings`,
    body: data,
  })
}

export function updateMapping(
  id: string,
  data: UpdateChannelMappingRequest,
) {
  return request<ChannelMappingResponse>({
    method: 'PUT',
    path: `/api/v1/integrations/mappings/${id}`,
    body: data,
  })
}

export function deleteMapping(id: string) {
  return request<{ status: string }>({
    method: 'DELETE',
    path: `/api/v1/integrations/mappings/${id}`,
  })
}

// ---------------------------------------------------------------------------
// Account linking (any user)
// ---------------------------------------------------------------------------

export function getLinkStatus(platform: Platform) {
  return request<AccountLinkStatus>({
    method: 'GET',
    path: `/api/v1/integrations/link/${platform}/status`,
  })
}

export function linkAccount(data: LinkAccountRequest) {
  return request<LinkAccountResponse>({
    method: 'POST',
    path: '/api/v1/integrations/link',
    body: data,
  })
}

export function unlinkAccount(platform: Platform) {
  return request<{ status: string }>({
    method: 'DELETE',
    path: `/api/v1/integrations/link/${platform}`,
  })
}

// ---------------------------------------------------------------------------
// Namespace re-exports for useIntegrations hooks (backward compat)
// ---------------------------------------------------------------------------

export const integrationConfigApi = {
  list: listConfigs,
  create: (data: CreateIntegrationConfigRequest) => createConfig(data),
  update: (id: string, data: UpdateIntegrationConfigRequest) =>
    updateConfig(id as Platform, data),
  delete: (id: string) => deleteConfig(id as Platform),
  test: (id: string) => testConfig(id as Platform),
}

export const accountLinkApi = {
  list: (): Promise<{ links: AccountLinkStatus[] }> =>
    Promise.resolve({ links: [] }),
  link: (platform: string, token: string) =>
    linkAccount({ token } as LinkAccountRequest),
  unlink: (platform: string) => unlinkAccount(platform as Platform),
}

export const channelMappingApi = {
  list: (integrationId: string) => listMappings(integrationId as Platform),
  create: (integrationId: string, data: CreateChannelMappingRequest) =>
    createMapping(integrationId as Platform, data),
  delete: (_integrationId: string, mappingId: string) =>
    deleteMapping(mappingId),
}
