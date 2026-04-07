/**
 * Integration Registry — all integration definitions, types, and field configs.
 *
 * Central source of truth for the integration grid and panels.
 * Custom panels (DATEV, Bexio) are rendered by their own components.
 * Generic panels use FieldDefinition[] to render fields automatically.
 */
import {
  Building2,
  MessageSquare,
  Hash,
  Plug,
  PenTool,
  FileText,
  Video,
  Radio,
  Phone,
  Send,
  type LucideIcon,
} from 'lucide-react'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type IntegrationCategory =
  | 'buchhaltung'
  | 'kommunikation'
  | 'dokumente'
  | 'video'
  | 'marketing'

export type FieldType = 'text' | 'password' | 'url' | 'select' | 'switch' | 'readonly'

export type AuthMethod = 'api_key' | 'oauth2' | 'server' | 'existing'

export type PanelType = 'generic' | 'custom'

export interface FieldDefinition {
  id: string
  label: string
  type: FieldType
  placeholder?: string
  helpText?: string
  options?: { value: string; label: string }[]
  defaultValue?: string | boolean
  required?: boolean
  section?: string
  monospace?: boolean
}

export interface IntegrationDefinition {
  id: string
  name: string
  description: string
  category: IntegrationCategory
  icon: LucideIcon
  iconColor: string
  authMethod: AuthMethod
  panelType: PanelType
  fields?: FieldDefinition[]
  comingSoon?: boolean
}

// ---------------------------------------------------------------------------
// Category display config
// ---------------------------------------------------------------------------

export const CATEGORY_ORDER: IntegrationCategory[] = [
  'buchhaltung',
  'kommunikation',
  'dokumente',
  'video',
  'marketing',
]

export const CATEGORY_LABEL_KEYS: Record<IntegrationCategory, string> = {
  buchhaltung: 'settings.integrations.category.accounting',
  kommunikation: 'settings.integrations.category.communication',
  dokumente: 'settings.integrations.category.documents',
  video: 'settings.integrations.category.video',
  marketing: 'settings.integrations.category.marketing',
}

// ---------------------------------------------------------------------------
// Platform map (existing API integrations → platform key)
// ---------------------------------------------------------------------------

export const PLATFORM_MAP: Record<string, string> = {
  slack: 'slack',
  'teams-webhook': 'teams',
  'custom-webhook': 'custom_webhook',
}

// ---------------------------------------------------------------------------
// All 12 integration definitions
// ---------------------------------------------------------------------------

export const INTEGRATION_REGISTRY: IntegrationDefinition[] = [
  // ── Buchhaltung & Finanzen ──────────────────────────────────────────────
  {
    id: 'datev-rechnungswesen',
    name: 'DATEV Rechnungswesen',
    description: 'settings.integrations.datev-rechnungswesen.description',
    category: 'buchhaltung',
    icon: Building2,
    iconColor: 'text-green-600',
    authMethod: 'api_key',
    panelType: 'custom',
  },
  {
    id: 'bexio',
    name: 'Bexio',
    description: 'settings.integrations.bexio.description',
    category: 'buchhaltung',
    icon: Building2,
    iconColor: 'text-blue-600',
    authMethod: 'oauth2',
    panelType: 'custom',
  },

  // ── Kommunikation ──────────────────────────────────────────────────────
  {
    id: 'slack',
    name: 'Slack',
    description: 'settings.integrations.slack.description',
    category: 'kommunikation',
    icon: Hash,
    iconColor: 'text-purple-600',
    authMethod: 'existing',
    panelType: 'generic',
  },
  {
    id: 'teams-webhook',
    name: 'Teams Webhook',
    description: 'settings.integrations.teams-webhook.description',
    category: 'kommunikation',
    icon: MessageSquare,
    iconColor: 'text-blue-500',
    authMethod: 'existing',
    panelType: 'generic',
  },
  {
    id: 'custom-webhook',
    name: 'Custom Webhook',
    description: 'settings.integrations.custom-webhook.description',
    category: 'kommunikation',
    icon: Plug,
    iconColor: 'text-orange-500',
    authMethod: 'existing',
    panelType: 'generic',
  },
  {
    id: 'teams-graph',
    name: 'Microsoft Teams (Graph)',
    description: 'settings.integrations.teams-graph.description',
    category: 'kommunikation',
    icon: MessageSquare,
    iconColor: 'text-blue-600',
    authMethod: 'api_key',
    panelType: 'generic',
    fields: [
      { id: 'tenantId', label: 'settings.integrations.teams-graph.fields.tenantId.label', type: 'text', placeholder: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx', required: true, section: 'settings.integrations.common.section.azureAd', monospace: true },
      { id: 'clientId', label: 'settings.integrations.teams-graph.fields.clientId.label', type: 'text', placeholder: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx', required: true, section: 'settings.integrations.common.section.azureAd', monospace: true },
      { id: 'clientSecret', label: 'settings.integrations.teams-graph.fields.clientSecret.label', type: 'password', placeholder: '***', required: true, section: 'settings.integrations.common.section.azureAd', monospace: true },
      { id: 'botEnabled', label: 'settings.integrations.teams-graph.fields.botEnabled.label', type: 'switch', defaultValue: false, section: 'settings.integrations.common.section.botConfig', helpText: 'settings.integrations.teams-graph.fields.botEnabled.helpText' },
      { id: 'autoCreateChannels', label: 'settings.integrations.teams-graph.fields.autoCreateChannels.label', type: 'switch', defaultValue: false, section: 'settings.integrations.common.section.botConfig', helpText: 'settings.integrations.teams-graph.fields.autoCreateChannels.helpText' },
    ],
  },
  {
    id: 'whatsapp',
    name: 'WhatsApp Business',
    description: 'settings.integrations.whatsapp.description',
    category: 'kommunikation',
    icon: Phone,
    iconColor: 'text-green-500',
    authMethod: 'server',
    panelType: 'generic',
    fields: [
      { id: 'apiUrl', label: 'settings.integrations.whatsapp.fields.apiUrl.label', type: 'url', placeholder: 'https://graph.facebook.com/v18.0/', required: true, section: 'settings.integrations.common.section.apiConnection', monospace: true },
      { id: 'phoneNumber', label: 'settings.integrations.whatsapp.fields.phoneNumber.label', type: 'text', placeholder: '+49 170 1234567', required: true, section: 'settings.integrations.common.section.apiConnection' },
      { id: 'accessToken', label: 'settings.integrations.whatsapp.fields.accessToken.label', type: 'password', placeholder: 'EAAx...', required: true, section: 'settings.integrations.common.section.apiConnection', monospace: true },
      { id: 'webhookUrl', label: 'settings.integrations.whatsapp.fields.webhookUrl.label', type: 'url', placeholder: 'https://kmuhub.app/api/v1/webhooks/whatsapp', section: 'settings.integrations.common.section.webhook', monospace: true },
      { id: 'verifyToken', label: 'settings.integrations.whatsapp.fields.verifyToken.label', type: 'password', placeholder: 'settings.integrations.whatsapp.fields.verifyToken.placeholder', section: 'settings.integrations.common.section.webhook', monospace: true },
    ],
  },

  // ── Dokumente & Signatur ────────────────────────────────────────────────
  {
    id: 'skribble',
    name: 'Skribble',
    description: 'settings.integrations.skribble.description',
    category: 'dokumente',
    icon: PenTool,
    iconColor: 'text-indigo-600',
    authMethod: 'api_key',
    panelType: 'generic',
    fields: [
      { id: 'apiKey', label: 'settings.integrations.common.fields.apiKey.label', type: 'password', placeholder: 'sk_live_...', required: true, section: 'settings.integrations.common.section.authentication', monospace: true },
      { id: 'signatureLevel', label: 'settings.integrations.skribble.fields.signatureLevel.label', type: 'select', options: [{ value: 'ses', label: 'settings.integrations.skribble.fields.signatureLevel.option.ses' }, { value: 'aes', label: 'settings.integrations.skribble.fields.signatureLevel.option.aes' }, { value: 'qes', label: 'settings.integrations.skribble.fields.signatureLevel.option.qes' }], defaultValue: 'aes', section: 'settings.integrations.common.section.settings' },
    ],
  },
  {
    id: 'collabora',
    name: 'Collabora Online',
    description: 'settings.integrations.collabora.description',
    category: 'dokumente',
    icon: FileText,
    iconColor: 'text-teal-600',
    authMethod: 'server',
    panelType: 'generic',
    fields: [
      { id: 'serverUrl', label: 'settings.integrations.common.fields.serverUrl.label', type: 'url', placeholder: 'https://collabora.example.com', required: true, section: 'settings.integrations.common.section.serverConfig', monospace: true },
      { id: 'wopiDiscoveryUrl', label: 'settings.integrations.collabora.fields.wopiDiscoveryUrl.label', type: 'url', placeholder: 'https://collabora.example.com/hosting/discovery', section: 'settings.integrations.common.section.serverConfig', monospace: true },
      { id: 'licenseStatus', label: 'settings.integrations.collabora.fields.licenseStatus.label', type: 'readonly', defaultValue: 'Community Edition', section: 'settings.integrations.common.section.license' },
    ],
  },

  // ── Video & Meetings ────────────────────────────────────────────────────
  {
    id: 'zoom',
    name: 'Zoom',
    description: 'settings.integrations.zoom.description',
    category: 'video',
    icon: Video,
    iconColor: 'text-blue-500',
    authMethod: 'oauth2',
    panelType: 'generic',
    fields: [
      { id: 'autoRecord', label: 'settings.integrations.zoom.fields.autoRecord.label', type: 'switch', defaultValue: false, section: 'settings.integrations.common.section.meetingDefaults', helpText: 'settings.integrations.zoom.fields.autoRecord.helpText' },
      { id: 'waitingRoom', label: 'settings.integrations.zoom.fields.waitingRoom.label', type: 'switch', defaultValue: true, section: 'settings.integrations.common.section.meetingDefaults', helpText: 'settings.integrations.zoom.fields.waitingRoom.helpText' },
      { id: 'defaultDuration', label: 'settings.integrations.zoom.fields.defaultDuration.label', type: 'select', options: [{ value: '30', label: 'settings.integrations.zoom.fields.defaultDuration.option.30' }, { value: '60', label: 'settings.integrations.zoom.fields.defaultDuration.option.60' }, { value: '90', label: 'settings.integrations.zoom.fields.defaultDuration.option.90' }], defaultValue: '60', section: 'settings.integrations.common.section.meetingDefaults' },
    ],
  },
  {
    id: 'livekit',
    name: 'LiveKit',
    description: 'settings.integrations.livekit.description',
    category: 'video',
    icon: Radio,
    iconColor: 'text-rose-500',
    authMethod: 'server',
    panelType: 'generic',
    fields: [
      { id: 'serverUrl', label: 'settings.integrations.common.fields.serverUrl.label', type: 'url', placeholder: 'wss://livekit.example.com', required: true, section: 'settings.integrations.common.section.server', monospace: true },
      { id: 'apiKey', label: 'settings.integrations.common.fields.apiKey.label', type: 'password', placeholder: 'API...', required: true, section: 'settings.integrations.common.section.credentials', monospace: true },
      { id: 'apiSecret', label: 'settings.integrations.livekit.fields.apiSecret.label', type: 'password', placeholder: '***', required: true, section: 'settings.integrations.common.section.credentials', monospace: true },
    ],
  },

  // ── Marketing ───────────────────────────────────────────────────────────
  {
    id: 'brevo',
    name: 'Brevo',
    description: 'settings.integrations.brevo.description',
    category: 'marketing',
    icon: Send,
    iconColor: 'text-sky-600',
    authMethod: 'api_key',
    panelType: 'generic',
    fields: [
      { id: 'apiKey', label: 'settings.integrations.common.fields.apiKey.label', type: 'password', placeholder: 'xkeysib-...', required: true, section: 'settings.integrations.common.section.authentication', monospace: true },
      { id: 'listMapping', label: 'settings.integrations.brevo.fields.listMapping.label', type: 'switch', defaultValue: false, section: 'settings.integrations.common.section.synchronisation', helpText: 'settings.integrations.brevo.fields.listMapping.helpText' },
      { id: 'syncEnabled', label: 'settings.integrations.brevo.fields.syncEnabled.label', type: 'switch', defaultValue: false, section: 'settings.integrations.common.section.synchronisation', helpText: 'settings.integrations.brevo.fields.syncEnabled.helpText' },
    ],
  },
]
