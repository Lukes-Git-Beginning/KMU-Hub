/**
 * Integration module types for external platform connections
 * (Teams, Slack, etc.)
 */

export type IntegrationPlatform = 'teams' | 'slack' | 'custom_webhook'
export type IntegrationStatus = 'active' | 'inactive' | 'error'

export interface IntegrationConfig {
  id: string
  platform: IntegrationPlatform
  name: string
  webhook_url?: string
  status: IntegrationStatus
  last_tested_at?: string
  created_at: string
  updated_at: string
}

export interface CreateIntegrationRequest {
  platform: IntegrationPlatform
  name: string
  webhook_url?: string
  credentials?: Record<string, string>
}

export interface UpdateIntegrationRequest {
  name?: string
  webhook_url?: string
  status?: IntegrationStatus
}

export interface AccountLink {
  id: string
  platform: IntegrationPlatform
  external_user_id?: string
  linked: boolean
  linked_at?: string
}

export interface ChannelMapping {
  id: string
  integration_id: string
  channel_id: string
  channel_name: string
  modules: string[]
  is_active: boolean
}

export interface CreateChannelMappingRequest {
  channel_id: string
  channel_name: string
  modules: string[]
}
