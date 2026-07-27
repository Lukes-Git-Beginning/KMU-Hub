import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface Ticket {
  id: string
  ticketNr: string
  subject: string
  description: string
  priority: 'low' | 'medium' | 'high' | 'critical'
  status: 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'
  assignedTo: string
  contactName: string
  slaDueAt: string
  slaOverdue: boolean
  /** Remaining (or, when overdue, elapsed) time split into days + hours. Label via slaLabel(). */
  slaDays: number
  slaHours: number
  createdAt: string
  updatedAt: string
  category?: string
  customFields?: Record<string, string | number | boolean>
  csatRating?: number
  csatComment?: string
  autoRouted?: boolean
}

// ---------------------------------------------------------------------------
// Categories (5.10)
// ---------------------------------------------------------------------------

export const MOCK_CATEGORIES = [
  'Netzwerk',
  'Hardware',
  'Software',
  'Zugang',
  'E-Mail',
  'Telefonie',
  'Sicherheit',
  'Sonstiges',
] as const

export type TicketCategory = (typeof MOCK_CATEGORIES)[number]

// ---------------------------------------------------------------------------
// Custom Field Definitions (5.13)
// ---------------------------------------------------------------------------

export interface CustomFieldDef {
  id: string
  name: string
  type: 'text' | 'number' | 'dropdown' | 'checkbox'
  options?: string[]
}

/** @deprecated Legacy 5.13 mock set — superseded by the customization-layer
 *  `helpdesk_ticket` custom fields (mocks/data/custom-fields.ts), which the editor
 *  edits and the detail now renders via `useModuleCustomFields`. Kept only for any
 *  remaining references; do not use for the ticket detail. */
export const MOCK_CUSTOM_FIELD_DEFS: CustomFieldDef[] = [
  { id: 'cf-1', name: 'Gerätetyp', type: 'dropdown', options: ['Laptop', 'Desktop', 'Drucker', 'Telefon', 'Monitor', 'Netzwerk', 'Server', 'Sonstiges'] },
  { id: 'cf-2', name: 'Raumnummer', type: 'text' },
  { id: 'cf-3', name: 'Remotezugriff erlaubt', type: 'checkbox' },
  { id: 'cf-4', name: 'Geschätzter Aufwand (h)', type: 'number' },
]

// Demo custom-field values per ticket now live on the wire seeds
// (mocks/handlers/helpdesk.ts → SEED_CUSTOM_FIELDS) so the module reads them
// through the adapter (intake P1b). The former display-layer overlay was removed.

// ---------------------------------------------------------------------------
// Canned Responses (5.6)
// ---------------------------------------------------------------------------

export interface CannedResponse {
  id: string
  title: string
  content: string
  category: string
  shortcut: string
}

export const MOCK_CANNED_RESPONSES: CannedResponse[] = [
  { id: 'cr-1', title: 'Begrüßung', content: '<p>Guten Tag,</p><p>vielen Dank für Ihre Anfrage. Wir haben Ihr Ticket erhalten und werden uns so schnell wie möglich darum kümmern.</p><p>Freundliche Grüße,<br/>IT Helpdesk</p>', category: 'Allgemein', shortcut: '/gruss' },
  { id: 'cr-2', title: 'Ticket geschlossen', content: '<p>Guten Tag,</p><p>Ihr Ticket wurde bearbeitet und geschlossen. Sollte das Problem erneut auftreten, eröffnen Sie bitte ein neues Ticket mit Verweis auf diese Ticketnummer.</p><p>Freundliche Grüße,<br/>IT Helpdesk</p>', category: 'Allgemein', shortcut: '/close' },
  { id: 'cr-3', title: 'VPN Troubleshooting', content: '<p>Bitte führen Sie folgende Schritte aus:</p><ol><li>VPN-Client komplett beenden (Taskleiste prüfen)</li><li>Internetverbindung prüfen</li><li>VPN-Client neu starten</li><li>Server-Adresse: <strong>vpn.firma.de</strong></li></ol><p>Falls das Problem weiterhin besteht, senden Sie bitte die Logdateien.</p>', category: 'Netzwerk', shortcut: '/vpn' },
  { id: 'cr-4', title: 'Passwort-Reset Anleitung', content: '<p>Sie können Ihr Passwort selbst zurücksetzen:</p><ol><li>Besuchen Sie <strong>https://password.firma.de</strong></li><li>Geben Sie Ihren Benutzernamen ein</li><li>Bestätigen Sie per SMS-Code</li><li>Setzen Sie ein neues Passwort (mind. 12 Zeichen)</li></ol>', category: 'Zugang', shortcut: '/pw' },
  { id: 'cr-5', title: 'Hardware-Bestellung', content: '<p>Vielen Dank für die Anfrage. Für Hardware-Bestellungen benötigen wir:</p><ul><li>Genaue Gerätebezeichnung</li><li>Kostenstelle / Budget-Freigabe</li><li>Gewünschtes Lieferdatum</li><li>Genehmigung des Vorgesetzten</li></ul>', category: 'Hardware', shortcut: '/hw' },
  { id: 'cr-6', title: 'Rückfrage', content: '<p>Guten Tag,</p><p>Für die weitere Bearbeitung benötigen wir noch folgende Informationen:</p><ul><li>[Bitte ergänzen]</li></ul><p>Bitte antworten Sie direkt auf dieses Ticket.</p>', category: 'Allgemein', shortcut: '/frage' },
]

// ---------------------------------------------------------------------------
// Routing Rules (5.11)
// ---------------------------------------------------------------------------

export interface RoutingRule {
  id: string
  category: string
  assignTo: string
  priorityOverride?: Ticket['priority']
  active: boolean
}

export const MOCK_ROUTING_RULES: RoutingRule[] = [
  { id: 'rr-1', category: 'Netzwerk', assignTo: 'Marco Hartmann', active: true },
  { id: 'rr-2', category: 'Hardware', assignTo: 'Sandra Bürki', active: true },
  { id: 'rr-3', category: 'Software', assignTo: 'Marco Hartmann', active: true },
  { id: 'rr-4', category: 'Zugang', assignTo: 'Sandra Bürki', active: true },
  { id: 'rr-5', category: 'E-Mail', assignTo: 'Marco Hartmann', active: true },
  { id: 'rr-6', category: 'Telefonie', assignTo: 'Sandra Bürki', active: true },
  { id: 'rr-7', category: 'Sicherheit', assignTo: 'Marco Hartmann', priorityOverride: 'high', active: true },
  { id: 'rr-8', category: 'Sonstiges', assignTo: 'Marco Hartmann', active: true },
]

// ---------------------------------------------------------------------------
// Business Hours (5.8)
// ---------------------------------------------------------------------------

export interface BusinessDay {
  day: string
  active: boolean
  start: string
  end: string
}

export interface Holiday {
  id: string
  date: string
  name: string
}

export const MOCK_BUSINESS_HOURS: BusinessDay[] = [
  { day: 'Montag', active: true, start: '08:00', end: '17:30' },
  { day: 'Dienstag', active: true, start: '08:00', end: '17:30' },
  { day: 'Mittwoch', active: true, start: '08:00', end: '17:30' },
  { day: 'Donnerstag', active: true, start: '08:00', end: '17:30' },
  { day: 'Freitag', active: true, start: '08:00', end: '17:30' },
  { day: 'Samstag', active: false, start: '09:00', end: '13:00' },
  { day: 'Sonntag', active: false, start: '09:00', end: '13:00' },
]

export const MOCK_HOLIDAYS: Holiday[] = [
  { id: 'h-1', date: '2026-01-01', name: 'Neujahr' },
  { id: 'h-2', date: '2026-01-06', name: 'Heilige Drei Könige' },
  { id: 'h-3', date: '2026-04-03', name: 'Karfreitag' },
  { id: 'h-4', date: '2026-04-06', name: 'Ostermontag' },
  { id: 'h-5', date: '2026-05-01', name: 'Tag der Arbeit' },
  { id: 'h-6', date: '2026-10-03', name: 'Tag der Deutschen Einheit' },
  { id: 'h-7', date: '2026-12-25', name: 'Weihnachtstag' },
  { id: 'h-8', date: '2026-12-26', name: 'Stephanstag' },
]

// ---------------------------------------------------------------------------
// Message Threads (5.7) — one conversation per ticket, lives in the store so
// replies/internal notes/escalations persist (was in-file MOCK_THREADS before).
// ---------------------------------------------------------------------------

export interface ThreadMessage {
  id: string
  author: string
  role: 'customer' | 'agent'
  text: string
  timestamp: string
  isInternal?: boolean
}

export interface KBArticle {
  id: string
  title: string
  category: string
  excerpt: string
  views: number
  published: boolean
  updatedAt: string
}

export interface HelpdeskStats {
  openTickets: number
  avgResponseTime: string
  resolvedThisWeek: number
  customerSatisfaction: string
  weeklyBreakdown: { label: string; count: number }[]
}

/** Payload for creating a ticket (the rest is derived in addTicket). */
export interface NewTicketInput {
  subject: string
  description: string
  priority: Ticket['priority']
  assignedTo: string
  contactName: string
  category?: string
}

const PRIORITY_LADDER: Ticket['priority'][] = ['low', 'medium', 'high', 'critical']

interface HelpdeskStore {
  tickets: Ticket[]
  threads: Record<string, ThreadMessage[]>
  kbArticles: KBArticle[]
  stats: HelpdeskStats
  cannedResponses: CannedResponse[]
  routingRules: RoutingRule[]
  businessHours: BusinessDay[]
  holidays: Holiday[]
  /** CSAT survey config (tenant): auto-send after close + the delay before it goes out. */
  csatEnabled: boolean
  csatDelayHours: number
  /**
   * Intake-channel config (tenant, Ticket-Intake P6). The three ways a ticket can
   * be created are individually on/off; the module's origin tabs and the
   * self-service / public entry points read this. `intakeFormId` binds the form
   * that the self-service + external channels render (a form with
   * intakeTargetId 'helpdesk_ticket'). Functional toggle — configured in the
   * module editor's Kanäle panel, applied directly (like csatEnabled), not via
   * the draft/deploy overlay used for content customization.
   */
  intakeChannels: { agent: boolean; selfservice: boolean; external: boolean }
  intakeFormId: string

  // ── Ticket actions ──
  /** Create a ticket with auto number (HD-YYYY-NNNN), SLA + timestamps; prepends it. Returns the new id. */
  addTicket: (input: NewTicketInput) => string
  updateTicketStatus: (id: string, status: Ticket['status']) => void
  assignTicket: (id: string, agent: string) => void
  /** Bump priority one step (capped at critical) + system thread note. */
  escalateTicket: (id: string, by: string) => void
  /** Merge a ticket into another: appends source thread + closes source with a note. */
  mergeTicket: (sourceId: string, targetId: string) => void
  addReply: (id: string, msg: { author: string; body: string; internal?: boolean }) => void
  saveCsat: (id: string, rating: number, comment?: string) => void

  // ── Canned responses ──
  addCannedResponse: (input: Omit<CannedResponse, 'id'>) => void
  updateCannedResponse: (id: string, patch: Partial<Omit<CannedResponse, 'id'>>) => void
  deleteCannedResponse: (id: string) => void

  // ── Config ──
  saveBusinessHours: (hours: BusinessDay[], holidays: Holiday[]) => void
  saveRoutingRules: (rules: RoutingRule[]) => void
  setCsatEnabled: (enabled: boolean) => void
  setCsatDelayHours: (hours: number) => void
  setIntakeChannel: (channel: 'agent' | 'selfservice' | 'external', enabled: boolean) => void
  setIntakeFormId: (id: string) => void

  // ── Knowledge base ──
  /** Edited KB article bodies (HTML), overriding the static fallback. */
  kbBodies: Record<string, string>
  saveKbBody: (id: string, html: string) => void
}

/** Next ticket number HD-YYYY-NNNN: current year, sequence = highest existing + 1. */
function nextTicketNr(tickets: Ticket[]): string {
  const year = new Date().getFullYear()
  let max = 0
  for (const t of tickets) {
    const m = /^HD-\d{4}-(\d+)$/.exec(t.ticketNr)
    if (m) max = Math.max(max, parseInt(m[1], 10))
  }
  return `HD-${year}-${String(max + 1).padStart(4, '0')}`
}

export interface SlaState {
  overdue: boolean
  days: number
  hours: number
}

/** Structured SLA against the real clock (no German strings — label via slaLabel). */
function computeSla(slaDueAt: string): SlaState {
  const due = new Date(slaDueAt)
  const now = new Date()
  const diffMs = due.getTime() - now.getTime()
  const overdue = diffMs < 0
  const totalHours = Math.abs(Math.floor(diffMs / 3600000))
  return { overdue, days: Math.floor(totalHours / 24), hours: totalHours % 24 }
}

/**
 * Localised SLA label from structured state. Single source of truth for every
 * SLA consumer (ticket list, detail modal, dashboard widget) — keeps the unit
 * strings in i18n instead of hardcoded German in the store.
 */
export function slaLabel(
  t: (key: string, opts?: Record<string, unknown>) => string,
  sla: Pick<Ticket, 'slaOverdue' | 'slaDays' | 'slaHours'>,
): string {
  const { slaOverdue: overdue, slaDays: days, slaHours: hours } = sla
  if (overdue) {
    return days > 0
      ? t('helpdesk.sla.overdueDays', { days, hours })
      : t('helpdesk.sla.overdueHours', { hours })
  }
  return days > 0
    ? t('helpdesk.sla.remainingDays', { days, hours })
    : t('helpdesk.sla.remainingHours', { hours })
}

function makeTicket(
  id: string, ticketNr: string, subject: string, description: string,
  priority: Ticket['priority'], status: Ticket['status'],
  assignedTo: string, contactName: string,
  slaDueAt: string, createdAt: string, updatedAt: string,
  category?: string,
  customFields?: Record<string, string | number | boolean>,
  csatRating?: number,
  csatComment?: string,
  autoRouted?: boolean,
): Ticket {
  const sla = computeSla(slaDueAt)
  return { id, ticketNr, subject, description, priority, status, assignedTo, contactName, slaDueAt, slaOverdue: sla.overdue, slaDays: sla.days, slaHours: sla.hours, createdAt, updatedAt, category, customFields, csatRating, csatComment, autoRouted }
}

const MOCK_TICKETS_RAW: Ticket[] = [
  makeTicket('tk-1', 'HD-2026-0301', 'Drucker im 2. OG druckt nicht', 'Seit heute Morgen funktioniert der Netzwerkdrucker HP LaserJet im 2. OG nicht mehr. Fehlermeldung: Offline.', 'high', 'in_progress', 'Marco Hartmann', 'Brigitte Schärer', '2026-02-15T16:00:00', '2026-02-15T08:30:00', '2026-02-15T09:15:00', 'Hardware', { 'Gerätetyp': 'Drucker', 'Raumnummer': '2.04', 'Remotezugriff erlaubt': true, 'Geschätzter Aufwand (h)': 1 }, undefined, undefined, true),
  makeTicket('tk-2', 'HD-2026-0302', 'VPN-Verbindung bricht ab', 'VPN-Verbindung zum Firmennetz trennt sich alle 10 Minuten. Betrifft Home-Office Zugang.', 'critical', 'in_progress', 'Marco Hartmann', 'Stefan Wenger', '2026-02-15T12:00:00', '2026-02-14T17:45:00', '2026-02-15T08:00:00', 'Netzwerk', { 'Gerätetyp': 'Laptop', 'Remotezugriff erlaubt': true, 'Geschätzter Aufwand (h)': 3 }, undefined, undefined, true),
  makeTicket('tk-3', 'HD-2026-0303', 'Neuer Mitarbeiter - Zugänge einrichten', 'Ab 01.03.2026 neuer Mitarbeiter: Lukas Meier, Abteilung Buchhaltung. Bitte alle Standardzugänge einrichten.', 'medium', 'open', 'Sandra Bürki', 'Karin Pfister', '2026-02-28T17:00:00', '2026-02-14T14:00:00', '2026-02-14T14:00:00', 'Zugang', { 'Raumnummer': '3.12' }, undefined, undefined, true),
  makeTicket('tk-4', 'HD-2026-0304', 'Outlook synchronisiert Kalender nicht', 'Kalendereinträge werden nicht zwischen Outlook Desktop und Mobile synchronisiert.', 'medium', 'waiting', 'Marco Hartmann', 'Andreas Müller', '2026-02-17T12:00:00', '2026-02-14T11:30:00', '2026-02-14T16:20:00', 'E-Mail', { 'Gerätetyp': 'Laptop', 'Remotezugriff erlaubt': false }),
  makeTicket('tk-5', 'HD-2026-0305', 'Bildschirm flackert', 'Der externe Monitor (Dell 27") an Arbeitsplatz B-12 flackert sporadisch.', 'low', 'open', 'Sandra Bürki', 'Regula Vogt', '2026-02-20T17:00:00', '2026-02-14T09:00:00', '2026-02-14T09:00:00', 'Hardware', { 'Gerätetyp': 'Monitor', 'Raumnummer': 'B-12', 'Geschätzter Aufwand (h)': 0.5 }),
  makeTicket('tk-6', 'HD-2026-0306', 'ERP-System Fehlermeldung bei Rechnungserstellung', 'Beim Erstellen von Rechnungen erscheint Fehler 500. Betrifft alle Mitarbeiter in der Buchhaltung.', 'critical', 'resolved', 'Marco Hartmann', 'Thomas Kunz', '2026-02-14T14:00:00', '2026-02-14T08:00:00', '2026-02-14T12:30:00', 'Software', { 'Geschätzter Aufwand (h)': 4 }, 5, 'Sehr schnell gelöst, danke!'),
  makeTicket('tk-7', 'HD-2026-0307', 'WLAN im Sitzungszimmer zu schwach', 'Im Sitzungszimmer "Pilatus" ist das WLAN-Signal zu schwach für Videokonferenzen.', 'medium', 'in_progress', 'Sandra Bürki', 'Daniel Roth', '2026-02-18T17:00:00', '2026-02-13T15:30:00', '2026-02-15T10:00:00', 'Netzwerk', { 'Raumnummer': 'Pilatus', 'Geschätzter Aufwand (h)': 2 }),
  makeTicket('tk-8', 'HD-2026-0308', 'Software-Lizenz Adobe CC abgelaufen', 'Die Adobe Creative Cloud Lizenz für Marketing ist seit gestern abgelaufen.', 'high', 'waiting', 'Sandra Bürki', 'Nicole Berger', '2026-02-16T12:00:00', '2026-02-13T11:00:00', '2026-02-14T10:00:00', 'Software', { 'Gerätetyp': 'Desktop' }),
  makeTicket('tk-9', 'HD-2026-0309', 'Backup-Fehler Fileserver', 'Nächtliches Backup des Fileservers ist seit 3 Tagen fehlgeschlagen. Fehlermeldung: Speicherplatz unzureichend.', 'critical', 'resolved', 'Marco Hartmann', 'System Alert', '2026-02-13T10:00:00', '2026-02-12T06:00:00', '2026-02-13T09:00:00', 'Sicherheit', { 'Gerätetyp': 'Server', 'Geschätzter Aufwand (h)': 6 }, 4, 'Gut gelöst, hätte aber etwas schneller sein können.'),
  makeTicket('tk-10', 'HD-2026-0310', 'Zutrittskarte funktioniert nicht', 'Zutrittskarte für Büro 3. OG funktioniert seit heute Morgen nicht mehr. Badge-Nr: Z-4412.', 'medium', 'open', 'Sandra Bürki', 'Eveline Stauffer', '2026-02-16T17:00:00', '2026-02-15T07:50:00', '2026-02-15T07:50:00', 'Sicherheit', { 'Raumnummer': '3. OG' }),
  makeTicket('tk-11', 'HD-2026-0311', 'Telefon-Anlage: Weiterleitung einrichten', 'Neue Rufweiterleitung für Empfang auf +49 89 555 12 99 einrichten (Urlaubsabwesenheit).', 'low', 'closed', 'Marco Hartmann', 'Ruth Eberle', '2026-02-14T17:00:00', '2026-02-12T14:00:00', '2026-02-13T11:00:00', 'Telefonie', {}, 5, 'Perfekt, vielen Dank!'),
  makeTicket('tk-12', 'HD-2026-0312', 'Laptop für Aussendienst bestellen', 'Neuer Laptop (ThinkPad T14) für Aussendienst-Mitarbeiter Reto Graf benötigt. Inkl. Dockingstation.', 'low', 'open', 'Sandra Bürki', 'Beat Kuhn', '2026-02-25T17:00:00', '2026-02-11T16:00:00', '2026-02-11T16:00:00', 'Hardware', { 'Gerätetyp': 'Laptop' }),
  makeTicket('tk-13', 'HD-2026-0313', 'Sharepoint-Berechtigung für Projekt X', 'Team Projekt X braucht Zugriff auf Sharepoint-Ordner /Projekte/X. Benutzer: Meier, Huber, Keller.', 'medium', 'resolved', 'Sandra Bürki', 'Patricia Hofer', '2026-02-15T12:00:00', '2026-02-13T09:00:00', '2026-02-14T15:30:00', 'Zugang', {}, 4, ''),
  makeTicket('tk-14', 'HD-2026-0314', 'Virenwarnung auf Arbeitsplatz C-03', 'Sophos meldet verdächtige Datei auf Arbeitsplatz C-03 (Abt. Einkauf). Quarantäne aktiv.', 'high', 'in_progress', 'Marco Hartmann', 'System Alert', '2026-02-15T14:00:00', '2026-02-15T06:30:00', '2026-02-15T08:45:00', 'Sicherheit', { 'Gerätetyp': 'Desktop', 'Raumnummer': 'C-03', 'Remotezugriff erlaubt': false, 'Geschätzter Aufwand (h)': 2 }, undefined, undefined, true),
  makeTicket('tk-15', 'HD-2026-0315', 'Teams-Raum "Säntis" Kamera defekt', 'Die Konferenzkamera im Teams-Raum Säntis zeigt kein Bild mehr. Modell: Logitech Rally.', 'medium', 'open', 'Marco Hartmann', 'Yvonne Stocker', '2026-02-19T17:00:00', '2026-02-14T13:00:00', '2026-02-14T13:00:00', 'Hardware', { 'Gerätetyp': 'Sonstiges', 'Raumnummer': 'Säntis' }),
]

// ---------------------------------------------------------------------------
// Re-base seed timestamps relative to "now" so SLAs read believably against the
// real clock (computeSla uses new Date()). Per-ticket offsets in hours: created
// ago / updated ago / SLA due-in (negative = overdue) — a deliberate mix of
// overdue, soon-due (<4h = yellow) and comfortable.
// ---------------------------------------------------------------------------

const _SLA_H = 3600000
const _NOW = Date.now()
const SLA_SEED: Record<string, { createdAgoH: number; updatedAgoH: number; dueInH: number }> = {
  'tk-1': { createdAgoH: 30, updatedAgoH: 5, dueInH: 3 },     // soon (yellow)
  'tk-2': { createdAgoH: 50, updatedAgoH: 6, dueInH: -4 },    // overdue
  'tk-3': { createdAgoH: 48, updatedAgoH: 48, dueInH: 60 },   // comfortable
  'tk-4': { createdAgoH: 70, updatedAgoH: 20, dueInH: 30 },
  'tk-5': { createdAgoH: 80, updatedAgoH: 80, dueInH: 100 },
  'tk-6': { createdAgoH: 54, updatedAgoH: 50, dueInH: -40 },  // resolved, was overdue
  'tk-7': { createdAgoH: 96, updatedAgoH: 8, dueInH: 20 },
  'tk-8': { createdAgoH: 100, updatedAgoH: 30, dueInH: 2 },   // soon (yellow)
  'tk-9': { createdAgoH: 130, updatedAgoH: 120, dueInH: -72 },
  'tk-10': { createdAgoH: 10, updatedAgoH: 10, dueInH: 28 },
  'tk-11': { createdAgoH: 120, updatedAgoH: 100, dueInH: -90 },
  'tk-12': { createdAgoH: 140, updatedAgoH: 140, dueInH: 120 },
  'tk-13': { createdAgoH: 60, updatedAgoH: 30, dueInH: -20 },
  'tk-14': { createdAgoH: 30, updatedAgoH: 6, dueInH: 1 },    // very soon (yellow)
  'tk-15': { createdAgoH: 44, updatedAgoH: 44, dueInH: 50 },
}

const MOCK_TICKETS: Ticket[] = MOCK_TICKETS_RAW.map((t) => {
  const s = SLA_SEED[t.id]
  if (!s) return t
  const slaDueAt = new Date(_NOW + s.dueInH * _SLA_H).toISOString()
  const sla = computeSla(slaDueAt)
  return {
    ...t,
    createdAt: new Date(_NOW - s.createdAgoH * _SLA_H).toISOString(),
    updatedAt: new Date(_NOW - s.updatedAgoH * _SLA_H).toISOString(),
    slaDueAt,
    slaOverdue: sla.overdue,
    slaDays: sla.days,
    slaHours: sla.hours,
  }
})

// ---------------------------------------------------------------------------
// Seed threads — one believable conversation per ticket (all 15). Agent vs.
// customer turns, a couple of internal notes. Replies/notes append here.
// ---------------------------------------------------------------------------

const MOCK_THREADS: Record<string, ThreadMessage[]> = {
  'tk-1': [
    { id: 'tk-1-m1', author: 'Brigitte Schärer', role: 'customer', text: 'Der Drucker im 2. OG zeigt seit heute Morgen "Offline" an. Mehrere Kollegen haben das gleiche Problem.', timestamp: '2026-02-15T08:30:00' },
    { id: 'tk-1-m2', author: 'Marco Hartmann', role: 'agent', text: 'Danke für die Meldung. Ich prüfe den Druckserver und die Netzwerkverbindung. Bitte kurz Geduld.', timestamp: '2026-02-15T09:15:00' },
    { id: 'tk-1-m3', author: 'Marco Hartmann', role: 'agent', text: 'Der Druckspooler-Dienst wurde neu gestartet. Bitte testen Sie erneut.', timestamp: '2026-02-15T09:45:00' },
  ],
  'tk-2': [
    { id: 'tk-2-m1', author: 'Stefan Wenger', role: 'customer', text: 'Meine VPN-Verbindung trennt sich alle 10 Minuten. Arbeiten im Home-Office ist so nicht möglich.', timestamp: '2026-02-14T17:45:00' },
    { id: 'tk-2-m2', author: 'Marco Hartmann', role: 'agent', text: 'Welchen VPN-Client nutzen Sie? Bitte senden Sie mir die Logdatei.', timestamp: '2026-02-15T08:00:00' },
    { id: 'tk-2-m3', author: 'Stefan Wenger', role: 'customer', text: 'Cisco AnyConnect 4.10. Logs sind im Anhang.', timestamp: '2026-02-15T08:30:00' },
  ],
  'tk-3': [
    { id: 'tk-3-m1', author: 'Karin Pfister', role: 'customer', text: 'Neuer Mitarbeiter Lukas Meier startet am 01.03. in der Buchhaltung. Bitte alle Standardzugänge einrichten.', timestamp: '2026-02-14T14:00:00' },
    { id: 'tk-3-m2', author: 'Sandra Bürki', role: 'agent', text: 'Wird vorbereitet. AD-Konto, E-Mail, ERP und Zeiterfassung.', timestamp: '2026-02-14T15:00:00', isInternal: true },
  ],
  'tk-4': [
    { id: 'tk-4-m1', author: 'Andreas Müller', role: 'customer', text: 'Termine aus Outlook Desktop tauchen auf dem iPhone nicht auf. Andersrum genauso.', timestamp: '2026-02-14T11:30:00' },
    { id: 'tk-4-m2', author: 'Marco Hartmann', role: 'agent', text: 'Das klingt nach einem Profilfehler in der Exchange-Synchronisation. Ich brauche das Modell Ihres iPhones.', timestamp: '2026-02-14T16:20:00' },
  ],
  'tk-5': [
    { id: 'tk-5-m1', author: 'Regula Vogt', role: 'customer', text: 'Der Dell-Monitor an B-12 flackert sporadisch, etwa alle paar Minuten kurz.', timestamp: '2026-02-14T09:00:00' },
  ],
  'tk-6': [
    { id: 'tk-6-m1', author: 'Thomas Kunz', role: 'customer', text: 'Beim Erstellen von Rechnungen kommt Fehler 500. Die ganze Buchhaltung steht.', timestamp: '2026-02-14T08:00:00' },
    { id: 'tk-6-m2', author: 'Marco Hartmann', role: 'agent', text: 'Der ERP-Applikationsdienst hatte sich aufgehängt. Neustart durchgeführt, Rechnungslauf läuft wieder.', timestamp: '2026-02-14T12:30:00' },
    { id: 'tk-6-m3', author: 'Thomas Kunz', role: 'customer', text: 'Bestätigt, funktioniert wieder. Vielen Dank für die schnelle Hilfe!', timestamp: '2026-02-14T13:00:00' },
  ],
  'tk-7': [
    { id: 'tk-7-m1', author: 'Daniel Roth', role: 'customer', text: 'Im Sitzungszimmer Pilatus bricht das WLAN bei Videokonferenzen ständig ab.', timestamp: '2026-02-13T15:30:00' },
    { id: 'tk-7-m2', author: 'Sandra Bürki', role: 'agent', text: 'Wir prüfen die Access-Point-Abdeckung. Eventuell ist ein zusätzlicher AP nötig.', timestamp: '2026-02-15T10:00:00', isInternal: true },
  ],
  'tk-8': [
    { id: 'tk-8-m1', author: 'Nicole Berger', role: 'customer', text: 'Die Adobe-CC-Lizenz fürs Marketing ist abgelaufen, niemand kann mehr arbeiten.', timestamp: '2026-02-13T11:00:00' },
    { id: 'tk-8-m2', author: 'Sandra Bürki', role: 'agent', text: 'Verlängerung ist bei der Beschaffung angefragt. Ich warte auf die Budget-Freigabe.', timestamp: '2026-02-14T10:00:00' },
  ],
  'tk-9': [
    { id: 'tk-9-m1', author: 'System Alert', role: 'customer', text: 'Automatische Meldung: Nächtliches Backup des Fileservers seit 3 Tagen fehlgeschlagen (Speicherplatz unzureichend).', timestamp: '2026-02-12T06:00:00' },
    { id: 'tk-9-m2', author: 'Marco Hartmann', role: 'agent', text: 'Alte Snapshots bereinigt, Backup-Ziel erweitert. Nächster Lauf erfolgreich verifiziert.', timestamp: '2026-02-13T09:00:00' },
  ],
  'tk-10': [
    { id: 'tk-10-m1', author: 'Eveline Stauffer', role: 'customer', text: 'Meine Zutrittskarte (Z-4412) öffnet das Büro im 3. OG seit heute Morgen nicht mehr.', timestamp: '2026-02-15T07:50:00' },
  ],
  'tk-11': [
    { id: 'tk-11-m1', author: 'Ruth Eberle', role: 'customer', text: 'Bitte eine Rufweiterleitung für den Empfang auf +49 89 555 12 99 einrichten (Urlaub).', timestamp: '2026-02-12T14:00:00' },
    { id: 'tk-11-m2', author: 'Marco Hartmann', role: 'agent', text: 'Weiterleitung ist aktiv und getestet. Gilt bis auf Widerruf.', timestamp: '2026-02-13T11:00:00' },
  ],
  'tk-12': [
    { id: 'tk-12-m1', author: 'Beat Kuhn', role: 'customer', text: 'Für den Aussendienst-Kollegen Reto Graf brauchen wir ein ThinkPad T14 inkl. Dockingstation.', timestamp: '2026-02-11T16:00:00' },
  ],
  'tk-13': [
    { id: 'tk-13-m1', author: 'Patricia Hofer', role: 'customer', text: 'Das Team von Projekt X braucht Zugriff auf den Sharepoint-Ordner /Projekte/X.', timestamp: '2026-02-13T09:00:00' },
    { id: 'tk-13-m2', author: 'Sandra Bürki', role: 'agent', text: 'Berechtigungen für Meier, Huber und Keller gesetzt. Bitte einmal abmelden und neu anmelden.', timestamp: '2026-02-14T15:30:00' },
  ],
  'tk-14': [
    { id: 'tk-14-m1', author: 'System Alert', role: 'customer', text: 'Sophos meldet eine verdächtige Datei auf Arbeitsplatz C-03 (Einkauf). Quarantäne ist aktiv.', timestamp: '2026-02-15T06:30:00' },
    { id: 'tk-14-m2', author: 'Marco Hartmann', role: 'agent', text: 'Datei analysiert — False Positive durch ein Makro. Ich whiteliste die Signatur nach Rücksprache.', timestamp: '2026-02-15T08:45:00', isInternal: true },
  ],
  'tk-15': [
    { id: 'tk-15-m1', author: 'Yvonne Stocker', role: 'customer', text: 'Die Logitech-Rally-Kamera im Teams-Raum Säntis zeigt kein Bild mehr.', timestamp: '2026-02-14T13:00:00' },
  ],
}

const MOCK_KB_ARTICLES: KBArticle[] = [
  { id: 'kb-1', title: 'VPN-Verbindung einrichten (Windows)', category: 'Netzwerk', excerpt: 'Schritt-für-Schritt Anleitung zur Einrichtung der VPN-Verbindung unter Windows 10/11 mit dem Cisco AnyConnect Client.', views: 342, published: true, updatedAt: '2026-01-20T10:00:00' },
  { id: 'kb-2', title: 'Drucker hinzufügen im Netzwerk', category: 'Hardware', excerpt: 'So fügen Sie einen Netzwerkdrucker unter Windows oder macOS hinzu. Inkl. Treiber-Download Links.', views: 218, published: true, updatedAt: '2026-01-15T14:00:00' },
  { id: 'kb-3', title: 'Passwort zurücksetzen (Self-Service)', category: 'Sicherheit', excerpt: 'Anleitung zum selbstständigen Zurücksetzen des Active-Directory Passworts über das Self-Service Portal.', views: 567, published: true, updatedAt: '2026-02-01T09:00:00' },
  { id: 'kb-4', title: 'Outlook E-Mail Signatur einrichten', category: 'E-Mail', excerpt: 'Einheitliche E-Mail Signatur gemäß CI/CD-Richtlinien einrichten. Vorlagen und Anleitungen.', views: 189, published: true, updatedAt: '2025-12-10T11:00:00' },
  { id: 'kb-5', title: 'Home-Office Checkliste IT', category: 'Allgemein', excerpt: 'Checkliste für die IT-Ausstattung im Home-Office: VPN, Telefonie, Hardware-Anforderungen.', views: 445, published: true, updatedAt: '2026-01-08T16:00:00' },
]

const MOCK_STATS: HelpdeskStats = {
  openTickets: 8,
  avgResponseTime: '2.4 Std.',
  resolvedThisWeek: 5,
  customerSatisfaction: '4.6 / 5.0',
  weeklyBreakdown: [
    { label: 'Mo', count: 12 },
    { label: 'Di', count: 8 },
    { label: 'Mi', count: 15 },
    { label: 'Do', count: 10 },
    { label: 'Fr', count: 7 },
    { label: 'Sa', count: 2 },
    { label: 'So', count: 1 },
  ],
}

/** Monotonic id suffix for store-created entities (avoids Date.now collisions in a tick). */
let _seq = 0
function uid(prefix: string): string {
  _seq += 1
  return `${prefix}-${Date.now().toString(36)}${_seq.toString(36)}`
}

export const useHelpdeskStore = create<HelpdeskStore>()(
  persist(
    (set, get) => ({
      tickets: MOCK_TICKETS,
      threads: MOCK_THREADS,
      kbArticles: MOCK_KB_ARTICLES,
      stats: MOCK_STATS,
      cannedResponses: MOCK_CANNED_RESPONSES,
      routingRules: MOCK_ROUTING_RULES,
      businessHours: MOCK_BUSINESS_HOURS,
      holidays: MOCK_HOLIDAYS,
      csatEnabled: true,
      csatDelayHours: 24,
      intakeChannels: { agent: true, selfservice: true, external: false },
      intakeFormId: 'tmpl-ticket-intake',
      kbBodies: {},

      addTicket: (input) => {
        const now = new Date()
        const id = uid('tk')
        // SLA: 8 business-ish hours from now (demo); concrete dueAt drives the badge.
        const slaDueAt = new Date(now.getTime() + 8 * 3600000).toISOString()
        const ticket = makeTicket(
          id,
          nextTicketNr(get().tickets),
          input.subject,
          input.description,
          input.priority,
          'open',
          input.assignedTo,
          input.contactName || '—',
          slaDueAt,
          now.toISOString(),
          now.toISOString(),
          input.category,
        )
        set((s) => ({
          tickets: [ticket, ...s.tickets],
          threads: {
            ...s.threads,
            [id]: [
              {
                id: uid('m'),
                author: 'System',
                role: 'agent',
                text: input.description || 'Ticket erstellt.',
                timestamp: now.toISOString(),
              },
            ],
          },
        }))
        return id
      },

      updateTicketStatus: (id, status) =>
        set((s) => ({
          tickets: s.tickets.map((t) =>
            t.id === id ? { ...t, status, updatedAt: new Date().toISOString() } : t,
          ),
        })),

      assignTicket: (id, agent) =>
        set((s) => {
          const t = s.tickets.find((x) => x.id === id)
          if (!t || t.assignedTo === agent) {
            return {
              tickets: s.tickets.map((x) =>
                x.id === id ? { ...x, assignedTo: agent, updatedAt: new Date().toISOString() } : x,
              ),
            }
          }
          const note: ThreadMessage = {
            id: uid('m'),
            author: 'System',
            role: 'agent',
            text: `Ticket zugewiesen an ${agent}.`,
            timestamp: new Date().toISOString(),
            isInternal: true,
          }
          return {
            tickets: s.tickets.map((x) =>
              x.id === id ? { ...x, assignedTo: agent, updatedAt: new Date().toISOString() } : x,
            ),
            threads: { ...s.threads, [id]: [...(s.threads[id] ?? []), note] },
          }
        }),

      escalateTicket: (id, by) =>
        set((s) => {
          const t = s.tickets.find((x) => x.id === id)
          if (!t) return {}
          const idx = PRIORITY_LADDER.indexOf(t.priority)
          const next = PRIORITY_LADDER[Math.min(idx + 1, PRIORITY_LADDER.length - 1)]
          const note: ThreadMessage = {
            id: uid('m'),
            author: 'System',
            role: 'agent',
            text:
              next === t.priority
                ? `Eskalation durch ${by} — bereits höchste Priorität.`
                : `Eskaliert von ${by}: Priorität ${t.priority} → ${next}.`,
            timestamp: new Date().toISOString(),
            isInternal: true,
          }
          return {
            tickets: s.tickets.map((x) =>
              x.id === id ? { ...x, priority: next, updatedAt: new Date().toISOString() } : x,
            ),
            threads: { ...s.threads, [id]: [...(s.threads[id] ?? []), note] },
          }
        }),

      mergeTicket: (sourceId, targetId) =>
        set((s) => {
          const source = s.tickets.find((x) => x.id === sourceId)
          const target = s.tickets.find((x) => x.id === targetId)
          if (!source || !target || sourceId === targetId) return {}
          const now = new Date().toISOString()
          const movedThread = (s.threads[sourceId] ?? []).map((m) => ({ ...m, id: uid('m') }))
          const targetNote: ThreadMessage = {
            id: uid('m'),
            author: 'System',
            role: 'agent',
            text: `Ticket ${source.ticketNr} wurde hier zusammengeführt.`,
            timestamp: now,
            isInternal: true,
          }
          const sourceNote: ThreadMessage = {
            id: uid('m'),
            author: 'System',
            role: 'agent',
            text: `Zusammengeführt mit ${target.ticketNr}.`,
            timestamp: now,
            isInternal: true,
          }
          return {
            tickets: s.tickets.map((x) =>
              x.id === sourceId ? { ...x, status: 'closed', updatedAt: now } : x,
            ),
            threads: {
              ...s.threads,
              [targetId]: [...(s.threads[targetId] ?? []), targetNote, ...movedThread],
              [sourceId]: [...(s.threads[sourceId] ?? []), sourceNote],
            },
          }
        }),

      addReply: (id, msg) =>
        set((s) => {
          const entry: ThreadMessage = {
            id: uid('m'),
            author: msg.author,
            role: 'agent',
            text: msg.body,
            timestamp: new Date().toISOString(),
            isInternal: msg.internal,
          }
          return {
            threads: { ...s.threads, [id]: [...(s.threads[id] ?? []), entry] },
            tickets: s.tickets.map((t) =>
              t.id === id ? { ...t, updatedAt: new Date().toISOString() } : t,
            ),
          }
        }),

      saveCsat: (id, rating, comment) =>
        set((s) => ({
          tickets: s.tickets.map((t) =>
            t.id === id
              ? { ...t, csatRating: rating, csatComment: comment, updatedAt: new Date().toISOString() }
              : t,
          ),
        })),

      addCannedResponse: (input) =>
        set((s) => ({ cannedResponses: [{ ...input, id: uid('cr') }, ...s.cannedResponses] })),

      updateCannedResponse: (id, patch) =>
        set((s) => ({
          cannedResponses: s.cannedResponses.map((c) => (c.id === id ? { ...c, ...patch } : c)),
        })),

      deleteCannedResponse: (id) =>
        set((s) => ({ cannedResponses: s.cannedResponses.filter((c) => c.id !== id) })),

      saveBusinessHours: (hours, holidays) => set({ businessHours: hours, holidays }),

      saveRoutingRules: (rules) => set({ routingRules: rules }),

      setCsatEnabled: (enabled) => set({ csatEnabled: enabled }),
      setCsatDelayHours: (hours) => set({ csatDelayHours: hours }),

      setIntakeChannel: (channel, enabled) =>
        set((s) => ({ intakeChannels: { ...s.intakeChannels, [channel]: enabled } })),
      setIntakeFormId: (id) => set({ intakeFormId: id }),

      saveKbBody: (id, html) => set((s) => ({ kbBodies: { ...s.kbBodies, [id]: html } })),
    }),
    { name: 'cosmi-helpdesk' },
  ),
)
