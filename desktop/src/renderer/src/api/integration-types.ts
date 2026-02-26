/**
 * TypeScript types for the Integration module (Teams, Slack, future platforms).
 *
 * Covers: integration configs, channel mappings, account linking, and test
 * notification requests. Matches the backend gateway endpoints defined in
 * route_integration.go.
 */

// ---------------------------------------------------------------------------
// Platform type
// ---------------------------------------------------------------------------

export type Platform = 'teams' | 'slack'

// ---------------------------------------------------------------------------
// Integration config
// ---------------------------------------------------------------------------

export interface IntegrationConfig {
  id: string
  platform: Platform
  is_active: boolean
  credentials_vault_key: string
  metadata: string // JSON string from backend
  created_by: string
  created_at: string
  updated_at: string
  // UI-layer fields (optional, not in backend response)
  name?: string
  status?: string
  last_tested_at?: string
  webhook_url?: string
}

export interface CreateIntegrationConfigRequest {
  platform: Platform
  credentials_vault_key: string
  metadata?: string
}

export interface UpdateIntegrationConfigRequest {
  is_active?: boolean
  credentials_vault_key?: string
  metadata?: string
}

export interface IntegrationConfigResponse {
  config: IntegrationConfig
}

export interface IntegrationConfigListResponse {
  configs: IntegrationConfig[]
}

// ---------------------------------------------------------------------------
// Channel mappings
// ---------------------------------------------------------------------------

export interface ChannelMapping {
  id: string
  config_id: string
  channel_id: string
  channel_name: string
  modules: string[]
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateChannelMappingRequest {
  channel_id: string
  channel_name: string
  modules: string[]
}

export interface UpdateChannelMappingRequest {
  channel_id?: string
  channel_name?: string
  modules?: string[]
  is_active?: boolean
}

export interface ChannelMappingResponse {
  mapping: ChannelMapping
}

export interface ChannelMappingListResponse {
  mappings: ChannelMapping[]
}

// ---------------------------------------------------------------------------
// Account linking
// ---------------------------------------------------------------------------

export interface AccountLinkStatus {
  linked: boolean
  platform: Platform
  external_display_name?: string
  linked_at?: string
}

export interface LinkAccountRequest {
  token: string
}

export interface LinkAccountResponse {
  status: string
  platform: string
  external_id: string
}

// ---------------------------------------------------------------------------
// Test notification
// ---------------------------------------------------------------------------

export interface TestNotificationRequest {
  channel_id: string
}

export interface TestNotificationResponse {
  success: boolean
  message: string
}

// ---------------------------------------------------------------------------
// Module definitions for the mapping editor
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Backward-compatible aliases for design-integration hooks
// ---------------------------------------------------------------------------

export type IntegrationPlatform = Platform

export interface CreateIntegrationRequest {
  platform: Platform
  name: string
  webhook_url?: string
}

export interface UpdateIntegrationRequest {
  name?: string
  webhook_url?: string
  is_active?: boolean
}

// ---------------------------------------------------------------------------
// Module definitions for the mapping editor
// ---------------------------------------------------------------------------

export const INTEGRATION_MODULES = [
  { id: 'crm', label: 'CRM', color: '#3b82f6' },
  { id: 'work', label: 'Aufgaben', color: '#8b5cf6' },
  { id: 'hr', label: 'Personal', color: '#22c55e' },
  { id: 'biz', label: 'Finanzen', color: '#f59e0b' },
  { id: 'email', label: 'E-Mail', color: '#0ea5e9' },
  { id: 'chat', label: 'Chat', color: '#10b981' },
  { id: 'calendar', label: 'Kalender', color: '#f97316' },
  { id: 'document', label: 'Dokumente', color: '#64748b' },
  { id: 'system', label: 'System', color: '#ef4444' },
  { id: 'automation', label: 'Automation', color: '#a855f7' },
] as const
