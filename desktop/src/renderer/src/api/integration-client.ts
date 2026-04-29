/**
 * Integration API client for Teams/Slack configuration, channel mappings,
 * and account linking.
 *
 * Follows the caldav-client.ts / automation-client.ts pattern: typed fetch
 * helper with auth header injection and 401 retry.
 *
 * Gateway routes: /api/v1/integrations/*
 */
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
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function request<T>(opts: { method: string; path: string; body?: unknown }): Promise<T> {
  return authenticatedRequest<T>(opts)
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
