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

export const CATEGORY_LABELS: Record<IntegrationCategory, string> = {
  buchhaltung: 'Buchhaltung & Finanzen',
  kommunikation: 'Kommunikation',
  dokumente: 'Dokumente & Signatur',
  video: 'Video & Meetings',
  marketing: 'Marketing',
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
    description: 'Konten-Mapping, BWA/SuSa-Export, DATEV-Connect oder Dateiexport',
    category: 'buchhaltung',
    icon: Building2,
    iconColor: 'text-green-600',
    authMethod: 'api_key',
    panelType: 'custom',
  },
  {
    id: 'bexio',
    name: 'Bexio',
    description: 'Buchhaltungssoftware mit Kontakt-, Rechnungs- und Produkt-Sync',
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
    description: 'Team-Nachrichten, Channel-Mapping und Benachrichtigungen',
    category: 'kommunikation',
    icon: Hash,
    iconColor: 'text-purple-600',
    authMethod: 'existing',
    panelType: 'generic',
  },
  {
    id: 'teams-webhook',
    name: 'Teams Webhook',
    description: 'Microsoft Teams Webhook-Benachrichtigungen',
    category: 'kommunikation',
    icon: MessageSquare,
    iconColor: 'text-blue-500',
    authMethod: 'existing',
    panelType: 'generic',
  },
  {
    id: 'custom-webhook',
    name: 'Custom Webhook',
    description: 'Benutzerdefinierte Webhook-Integration',
    category: 'kommunikation',
    icon: Plug,
    iconColor: 'text-orange-500',
    authMethod: 'existing',
    panelType: 'generic',
  },
  {
    id: 'teams-graph',
    name: 'Microsoft Teams (Graph)',
    description: 'MS Graph API fuer Teams-Channels, Bot-Konfiguration und Praesenz',
    category: 'kommunikation',
    icon: MessageSquare,
    iconColor: 'text-blue-600',
    authMethod: 'api_key',
    panelType: 'generic',
    fields: [
      { id: 'tenantId', label: 'Tenant-ID', type: 'text', placeholder: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx', required: true, section: 'Azure AD', monospace: true },
      { id: 'clientId', label: 'Client-ID', type: 'text', placeholder: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx', required: true, section: 'Azure AD', monospace: true },
      { id: 'clientSecret', label: 'Client-Secret', type: 'password', placeholder: '***', required: true, section: 'Azure AD', monospace: true },
      { id: 'botEnabled', label: 'Bot aktiviert', type: 'switch', defaultValue: false, section: 'Bot-Konfiguration', helpText: 'Automatische Nachrichten in Teams-Channels senden' },
      { id: 'autoCreateChannels', label: 'Channels automatisch erstellen', type: 'switch', defaultValue: false, section: 'Bot-Konfiguration', helpText: 'Channels fuer neue Projekte automatisch anlegen' },
    ],
  },
  {
    id: 'whatsapp',
    name: 'WhatsApp Business',
    description: 'Kunden-Kommunikation ueber WhatsApp Business API',
    category: 'kommunikation',
    icon: Phone,
    iconColor: 'text-green-500',
    authMethod: 'server',
    panelType: 'generic',
    fields: [
      { id: 'apiUrl', label: 'API-URL', type: 'url', placeholder: 'https://graph.facebook.com/v18.0/', required: true, section: 'API-Verbindung', monospace: true },
      { id: 'phoneNumber', label: 'Telefonnummer', type: 'text', placeholder: '+49 170 1234567', required: true, section: 'API-Verbindung' },
      { id: 'accessToken', label: 'Access-Token', type: 'password', placeholder: 'EAAx...', required: true, section: 'API-Verbindung', monospace: true },
      { id: 'webhookUrl', label: 'Webhook-URL', type: 'url', placeholder: 'https://kmuhub.app/api/v1/webhooks/whatsapp', section: 'Webhook', monospace: true },
      { id: 'verifyToken', label: 'Verify-Token', type: 'password', placeholder: 'Bel. Token fuer Webhook-Verifizierung', section: 'Webhook', monospace: true },
    ],
  },

  // ── Dokumente & Signatur ────────────────────────────────────────────────
  {
    id: 'skribble',
    name: 'Skribble',
    description: 'Elektronische Signaturen (SES, AES, QES) fuer Vertraege und Dokumente',
    category: 'dokumente',
    icon: PenTool,
    iconColor: 'text-indigo-600',
    authMethod: 'api_key',
    panelType: 'generic',
    fields: [
      { id: 'apiKey', label: 'API-Schluessel', type: 'password', placeholder: 'sk_live_...', required: true, section: 'Authentifizierung', monospace: true },
      { id: 'signatureLevel', label: 'Standard-Signatur-Level', type: 'select', options: [{ value: 'ses', label: 'SES (Einfach)' }, { value: 'aes', label: 'AES (Fortgeschritten)' }, { value: 'qes', label: 'QES (Qualifiziert)' }], defaultValue: 'aes', section: 'Einstellungen' },
    ],
  },
  {
    id: 'collabora',
    name: 'Collabora Online',
    description: 'Online-Office-Editor fuer Dokumente direkt im Browser',
    category: 'dokumente',
    icon: FileText,
    iconColor: 'text-teal-600',
    authMethod: 'server',
    panelType: 'generic',
    fields: [
      { id: 'serverUrl', label: 'Server-URL', type: 'url', placeholder: 'https://collabora.example.com', required: true, section: 'Server-Konfiguration', monospace: true },
      { id: 'wopiDiscoveryUrl', label: 'WOPI-Discovery-URL', type: 'url', placeholder: 'https://collabora.example.com/hosting/discovery', section: 'Server-Konfiguration', monospace: true },
      { id: 'licenseStatus', label: 'Lizenz-Status', type: 'readonly', defaultValue: 'Community Edition', section: 'Lizenz' },
    ],
  },

  // ── Video & Meetings ────────────────────────────────────────────────────
  {
    id: 'zoom',
    name: 'Zoom',
    description: 'Video-Meetings erstellen und Kalender-Integration',
    category: 'video',
    icon: Video,
    iconColor: 'text-blue-500',
    authMethod: 'oauth2',
    panelType: 'generic',
    fields: [
      { id: 'autoRecord', label: 'Automatisch aufzeichnen', type: 'switch', defaultValue: false, section: 'Meeting-Defaults', helpText: 'Meetings automatisch in der Cloud aufzeichnen' },
      { id: 'waitingRoom', label: 'Warteraum aktiviert', type: 'switch', defaultValue: true, section: 'Meeting-Defaults', helpText: 'Teilnehmer muessen vom Host zugelassen werden' },
      { id: 'defaultDuration', label: 'Standard-Dauer', type: 'select', options: [{ value: '30', label: '30 Minuten' }, { value: '60', label: '60 Minuten' }, { value: '90', label: '90 Minuten' }], defaultValue: '60', section: 'Meeting-Defaults' },
    ],
  },
  {
    id: 'livekit',
    name: 'LiveKit',
    description: 'Self-Hosted Video/Audio-Infrastruktur fuer Meetings und Anrufe',
    category: 'video',
    icon: Radio,
    iconColor: 'text-rose-500',
    authMethod: 'server',
    panelType: 'generic',
    fields: [
      { id: 'serverUrl', label: 'Server-URL', type: 'url', placeholder: 'wss://livekit.example.com', required: true, section: 'Server', monospace: true },
      { id: 'apiKey', label: 'API-Key', type: 'password', placeholder: 'API...', required: true, section: 'Zugangsdaten', monospace: true },
      { id: 'apiSecret', label: 'API-Secret', type: 'password', placeholder: '***', required: true, section: 'Zugangsdaten', monospace: true },
    ],
  },

  // ── Marketing ───────────────────────────────────────────────────────────
  {
    id: 'brevo',
    name: 'Brevo',
    description: 'E-Mail-Marketing, Newsletter-Versand und Kontakt-Synchronisation',
    category: 'marketing',
    icon: Send,
    iconColor: 'text-sky-600',
    authMethod: 'api_key',
    panelType: 'generic',
    fields: [
      { id: 'apiKey', label: 'API-Schluessel', type: 'password', placeholder: 'xkeysib-...', required: true, section: 'Authentifizierung', monospace: true },
      { id: 'listMapping', label: 'Listen-Zuordnung', type: 'switch', defaultValue: false, section: 'Synchronisation', helpText: 'Kontaktgruppen automatisch als Brevo-Listen synchronisieren' },
      { id: 'syncEnabled', label: 'Sync aktiviert', type: 'switch', defaultValue: false, section: 'Synchronisation', helpText: 'Neue Kontakte automatisch an Brevo uebertragen' },
    ],
  },
]
