import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'

const API = API_BASE_URL

// ---------------------------------------------------------------------------
// Integration configs
// ---------------------------------------------------------------------------

const integrationConfigs = [
  { id: 'int-bexio', name: 'Bexio', type: 'accounting', status: 'disconnected', icon: 'bexio', description: 'Schweizer Buchhaltungssoftware — Kontakte, Rechnungen, Zeiterfassung synchronisieren.' },
  { id: 'int-lexware', name: 'Lexware', type: 'accounting', status: 'disconnected', icon: 'lexware', description: 'Deutsche Buchhaltung & Lohn — DATEV-Export, Rechnungsimport.' },
  { id: 'int-datev', name: 'DATEV', type: 'accounting', status: 'disconnected', icon: 'datev', description: 'Buchungsdaten an den Steuerberater übermitteln.' },
  { id: 'int-caldav', name: 'CalDAV', type: 'calendar', status: 'disabled', icon: 'calendar', description: 'Kalender-Synchronisation mit externen Anbietern (Google, Outlook, Nextcloud).' },
  { id: 'int-carddav', name: 'CardDAV', type: 'contacts', status: 'disabled', icon: 'contacts', description: 'Kontakt-Synchronisation mit externen Adressbuechern.' },
  { id: 'int-slack', name: 'Slack', type: 'messaging', status: 'disconnected', icon: 'slack', description: 'Benachrichtigungen und Aktionen in Slack-Channels.' },
  { id: 'int-teams', name: 'Microsoft Teams', type: 'messaging', status: 'disconnected', icon: 'teams', description: 'Meetings und Benachrichtigungen via Teams.' },
  { id: 'int-smtp', name: 'E-Mail (SMTP)', type: 'email', status: 'disconnected', icon: 'mail', description: 'Ausgehende E-Mails ueber eigenen Mailserver versenden.' },
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

  // Bexio status — disconnected
  http.get(`${API}/api/v1/integrations/bexio/status`, () => {
    return HttpResponse.json({ connected: false, last_sync: null, error: null })
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
