import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { DEFAULT_CONTACT_MAPPINGS } from '@/api/bexio-types'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Bexio integration — stateful demo mock (invoice-pull review, Session #8)
// ---------------------------------------------------------------------------
// Starts DISCONNECTED so the setup wizard is reviewable (card → wizard). The
// OAuth authorize call flips it to connected (simulating the completed OAuth
// round-trip), which unlocks the sync dashboard (card → dashboard) and lets the
// wizard advance past step 1. Disconnect resets it. This lets a reviewer walk
// wizard → dashboard → read-only invoice without any code change.
let bexioConnected = false

// Timestamps computed once at module load so the "vor X" relative labels in the
// dashboard stay stable across a QA session.
const bxNow = Date.now()
const bxMinsAgo = (m: number) => new Date(bxNow - m * 60_000).toISOString()
const bxHoursAgo = (h: number) => new Date(bxNow - h * 3_600_000).toISOString()
const bxDaysAgo = (d: number) => new Date(bxNow - d * 86_400_000).toISOString()

// Shape: BexioSyncStatus (api/bexio-types.ts)
const bexioSyncStatus = {
  contact_sync_enabled: true,
  invoice_push_enabled: false,
  quote_push_enabled: false,
  payment_poll_enabled: true,
  invoice_pull_enabled: true,
  total_contacts_mapped: 128,
  total_invoices_mapped: 34,
  total_quotes_mapped: 0,
  last_contact_sync_at: bxMinsAgo(12),
  last_payment_poll_at: bxHoursAgo(2),
  last_invoice_pull_at: bxMinsAgo(4),
}

// Shape: BexioSyncLogEntry[] (api/bexio-types.ts)
const bexioSyncLogs = [
  { id: 'bxlog-1', sync_type: 'invoice_pull', status: 'completed', items_processed: 12, items_created: 3, items_updated: 9, items_failed: 0, started_at: bxMinsAgo(4), completed_at: bxMinsAgo(4) },
  { id: 'bxlog-2', sync_type: 'contact_delta', status: 'completed', items_processed: 47, items_created: 5, items_updated: 42, items_failed: 0, started_at: bxMinsAgo(12), completed_at: bxMinsAgo(11) },
  { id: 'bxlog-3', sync_type: 'payment_poll', status: 'partial', items_processed: 8, items_created: 0, items_updated: 7, items_failed: 1, error_message: 'Zahlung #4821 ohne zugeordnete Rechnung — übersprungen.', started_at: bxHoursAgo(2), completed_at: bxHoursAgo(2) },
  { id: 'bxlog-4', sync_type: 'contact_full', status: 'completed', items_processed: 128, items_created: 128, items_updated: 0, items_failed: 0, started_at: bxDaysAgo(3), completed_at: bxDaysAgo(3) },
]

// ---------------------------------------------------------------------------
// Integration configs
// ---------------------------------------------------------------------------

const integrationConfigs = [
  { id: 'int-bexio', name: 'Bexio', type: 'accounting', status: 'disconnected', icon: 'bexio', description: 'Schweizer Buchhaltungssoftware — Kontakte, Rechnungen, Zeiterfassung synchronisieren.' },
  { id: 'int-lexware', name: 'Lexware', type: 'accounting', status: 'disconnected', icon: 'lexware', description: 'Deutsche Buchhaltung & Lohn — DATEV-Export, Rechnungsimport.' },
  { id: 'int-datev', name: 'DATEV', type: 'accounting', status: 'disconnected', icon: 'datev', description: 'Buchungsdaten an den Steuerberater übermitteln.' },
  { id: 'int-caldav', name: 'CalDAV', type: 'calendar', status: 'disabled', icon: 'calendar', description: 'Kalender-Synchronisation mit externen Anbietern (Google, Outlook, Nextcloud).' },
  { id: 'int-carddav', name: 'CardDAV', type: 'contacts', status: 'disabled', icon: 'contacts', description: 'Kontakt-Synchronisation mit externen Adressbüchern.' },
  { id: 'int-slack', name: 'Slack', type: 'messaging', status: 'disconnected', icon: 'slack', description: 'Benachrichtigungen und Aktionen in Slack-Channels.' },
  { id: 'int-teams', name: 'Microsoft Teams', type: 'messaging', status: 'disconnected', icon: 'teams', description: 'Meetings und Benachrichtigungen via Teams.' },
  { id: 'int-smtp', name: 'E-Mail (SMTP)', type: 'email', status: 'disconnected', icon: 'mail', description: 'Ausgehende E-Mails über eigenen Mailserver versenden.' },
  { id: 'int-imap', name: 'E-Mail (IMAP)', type: 'email', status: 'disconnected', icon: 'mail', description: 'E-Mail-Postfach einbinden und synchronisieren.' },
]

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

export const settingsHandlers = [
  // Integration configs
  http.get(`${API}/api/v1/integrations/configs`, () => {
    return HttpResponse.json({ integrations: integrationConfigs, total: integrationConfigs.length })
  }),

  // Bexio — connection status (stateful, see block above)
  http.get(`${API}/api/v1/integrations/bexio/status`, () => {
    return bexioConnected
      ? HttpResponse.json({ connected: true, org_name: 'Zentria Demo AG', connected_at: bxDaysAgo(3) })
      : HttpResponse.json({ connected: false })
  }),

  // Bexio — OAuth authorize: returns the real Bexio consent URL and, for the
  // demo, marks the connection established so the wizard can advance.
  http.get(`${API}/api/v1/integrations/bexio/oauth/authorize`, () => {
    bexioConnected = true
    return HttpResponse.json({
      authorization_url:
        'https://office.bexio.com/oauth/authorize?client_id=cosmi-demo&response_type=code&scope=accounting',
    })
  }),

  // Bexio — disconnect: resets to the wizard flow
  http.post(`${API}/api/v1/integrations/bexio/disconnect`, () => {
    bexioConnected = false
    return HttpResponse.json({ success: true })
  }),

  // Bexio — sync status (4 dashboard cards)
  http.get(`${API}/api/v1/integrations/bexio/sync/status`, () => {
    return HttpResponse.json(bexioSyncStatus)
  }),

  // Bexio — sync history log
  http.get(`${API}/api/v1/integrations/bexio/sync/logs`, () => {
    return HttpResponse.json(bexioSyncLogs)
  }),

  // Bexio — manual sync trigger
  http.post(`${API}/api/v1/integrations/bexio/sync/trigger`, () => {
    return HttpResponse.json({ sync_id: 'bxsync-demo-1' })
  }),

  // Bexio — sync config persist (wizard finish)
  http.put(`${API}/api/v1/integrations/bexio/sync/config`, () => {
    return HttpResponse.json({ success: true })
  }),

  // Bexio — field mappings (contact/invoice/quote all share the contact set in demo)
  http.get(`${API}/api/v1/integrations/bexio/mappings/:entity`, () => {
    return HttpResponse.json({ mappings: DEFAULT_CONTACT_MAPPINGS })
  }),
  http.put(`${API}/api/v1/integrations/bexio/mappings/:entity`, () => {
    return HttpResponse.json({ success: true })
  }),

  // Lexware status — disconnected
  http.get(`${API}/api/v1/integrations/lexware/status`, () => {
    return HttpResponse.json({ connected: false, last_sync: null, error: null })
  }),

  // DATEV status — disconnected
  http.get(`${API}/api/v1/finance/datev/status`, () => {
    return HttpResponse.json({ connected: false, last_export: null, error: null })
  }),

  // CalDAV status — disabled
  http.get(`${API}/api/v1/caldav/status`, () => {
    return HttpResponse.json({ enabled: false, accounts: [], error: null })
  }),

  // Plugin manifests — empty
  http.get(`${API}/api/v1/plugins/manifests`, () => {
    return HttpResponse.json({ plugins: [], total: 0 })
  }),

  // Plugin installations — empty
  http.get(`${API}/api/v1/plugins/installations`, () => {
    return HttpResponse.json({ installations: [], total: 0 })
  }),
]
