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
  slaRemaining: string
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

export const MOCK_CUSTOM_FIELD_DEFS: CustomFieldDef[] = [
  { id: 'cf-1', name: 'Geraetetyp', type: 'dropdown', options: ['Laptop', 'Desktop', 'Drucker', 'Telefon', 'Monitor', 'Netzwerk', 'Server', 'Sonstiges'] },
  { id: 'cf-2', name: 'Raumnummer', type: 'text' },
  { id: 'cf-3', name: 'Remotezugriff erlaubt', type: 'checkbox' },
  { id: 'cf-4', name: 'Geschaetzter Aufwand (h)', type: 'number' },
]

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
  { id: 'cr-1', title: 'Begruessung', content: '<p>Guten Tag,</p><p>vielen Dank fuer Ihre Anfrage. Wir haben Ihr Ticket erhalten und werden uns so schnell wie moeglich darum kuemmern.</p><p>Freundliche Gruesse,<br/>IT Helpdesk</p>', category: 'Allgemein', shortcut: '/gruss' },
  { id: 'cr-2', title: 'Ticket geschlossen', content: '<p>Guten Tag,</p><p>Ihr Ticket wurde bearbeitet und geschlossen. Sollte das Problem erneut auftreten, eroeffnen Sie bitte ein neues Ticket mit Verweis auf diese Ticketnummer.</p><p>Freundliche Gruesse,<br/>IT Helpdesk</p>', category: 'Allgemein', shortcut: '/close' },
  { id: 'cr-3', title: 'VPN Troubleshooting', content: '<p>Bitte fuehren Sie folgende Schritte aus:</p><ol><li>VPN-Client komplett beenden (Taskleiste pruefen)</li><li>Internetverbindung pruefen</li><li>VPN-Client neu starten</li><li>Server-Adresse: <strong>vpn.firma.ch</strong></li></ol><p>Falls das Problem weiterhin besteht, senden Sie bitte die Logdateien.</p>', category: 'Netzwerk', shortcut: '/vpn' },
  { id: 'cr-4', title: 'Passwort-Reset Anleitung', content: '<p>Sie koennen Ihr Passwort selbst zuruecksetzen:</p><ol><li>Besuchen Sie <strong>https://password.firma.ch</strong></li><li>Geben Sie Ihren Benutzernamen ein</li><li>Bestaetigen Sie per SMS-Code</li><li>Setzen Sie ein neues Passwort (mind. 12 Zeichen)</li></ol>', category: 'Zugang', shortcut: '/pw' },
  { id: 'cr-5', title: 'Hardware-Bestellung', content: '<p>Vielen Dank fuer die Anfrage. Fuer Hardware-Bestellungen benoetigen wir:</p><ul><li>Genaue Geraetebezeichnung</li><li>Kostenstelle / Budget-Freigabe</li><li>Gewuenschtes Lieferdatum</li><li>Genehmigung des Vorgesetzten</li></ul>', category: 'Hardware', shortcut: '/hw' },
  { id: 'cr-6', title: 'Rueckfrage', content: '<p>Guten Tag,</p><p>Fuer die weitere Bearbeitung benoetigen wir noch folgende Informationen:</p><ul><li>[Bitte ergaenzen]</li></ul><p>Bitte antworten Sie direkt auf dieses Ticket.</p>', category: 'Allgemein', shortcut: '/frage' },
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
  { id: 'rr-2', category: 'Hardware', assignTo: 'Sandra Buerki', active: true },
  { id: 'rr-3', category: 'Software', assignTo: 'Marco Hartmann', active: true },
  { id: 'rr-4', category: 'Zugang', assignTo: 'Sandra Buerki', active: true },
  { id: 'rr-5', category: 'E-Mail', assignTo: 'Marco Hartmann', active: true },
  { id: 'rr-6', category: 'Telefonie', assignTo: 'Sandra Buerki', active: true },
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
  { id: 'h-2', date: '2026-01-02', name: 'Berchtoldstag' },
  { id: 'h-3', date: '2026-04-03', name: 'Karfreitag' },
  { id: 'h-4', date: '2026-04-06', name: 'Ostermontag' },
  { id: 'h-5', date: '2026-05-01', name: 'Tag der Arbeit' },
  { id: 'h-6', date: '2026-08-01', name: 'Bundesfeiertag' },
  { id: 'h-7', date: '2026-12-25', name: 'Weihnachtstag' },
  { id: 'h-8', date: '2026-12-26', name: 'Stephanstag' },
]

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

interface HelpdeskStore {
  tickets: Ticket[]
  kbArticles: KBArticle[]
  stats: HelpdeskStats
  cannedResponses: CannedResponse[]
  routingRules: RoutingRule[]
  businessHours: BusinessDay[]
  holidays: Holiday[]
}

function computeSla(slaDueAt: string): { overdue: boolean; remaining: string } {
  const due = new Date(slaDueAt)
  const now = new Date('2026-02-15T11:00:00')
  const diffMs = due.getTime() - now.getTime()
  if (diffMs < 0) {
    const hours = Math.abs(Math.floor(diffMs / 3600000))
    return { overdue: true, remaining: `${hours}h ueberfaellig` }
  }
  const hours = Math.floor(diffMs / 3600000)
  if (hours < 24) return { overdue: false, remaining: `${hours}h uebrig` }
  const days = Math.floor(hours / 24)
  return { overdue: false, remaining: `${days}d ${hours % 24}h uebrig` }
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
  return { id, ticketNr, subject, description, priority, status, assignedTo, contactName, slaDueAt, slaOverdue: sla.overdue, slaRemaining: sla.remaining, createdAt, updatedAt, category, customFields, csatRating, csatComment, autoRouted }
}

const MOCK_TICKETS: Ticket[] = [
  makeTicket('tk-1', 'HD-2026-0301', 'Drucker im 2. OG druckt nicht', 'Seit heute Morgen funktioniert der Netzwerkdrucker HP LaserJet im 2. OG nicht mehr. Fehlermeldung: Offline.', 'high', 'in_progress', 'Marco Hartmann', 'Brigitte Schaerer', '2026-02-15T16:00:00', '2026-02-15T08:30:00', '2026-02-15T09:15:00', 'Hardware', { 'Geraetetyp': 'Drucker', 'Raumnummer': '2.04', 'Remotezugriff erlaubt': true, 'Geschaetzter Aufwand (h)': 1 }, undefined, undefined, true),
  makeTicket('tk-2', 'HD-2026-0302', 'VPN-Verbindung bricht ab', 'VPN-Verbindung zum Firmennetz trennt sich alle 10 Minuten. Betrifft Home-Office Zugang.', 'critical', 'in_progress', 'Marco Hartmann', 'Stefan Wenger', '2026-02-15T12:00:00', '2026-02-14T17:45:00', '2026-02-15T08:00:00', 'Netzwerk', { 'Geraetetyp': 'Laptop', 'Remotezugriff erlaubt': true, 'Geschaetzter Aufwand (h)': 3 }, undefined, undefined, true),
  makeTicket('tk-3', 'HD-2026-0303', 'Neuer Mitarbeiter - Zugaenge einrichten', 'Ab 01.03.2026 neuer Mitarbeiter: Lukas Meier, Abteilung Buchhaltung. Bitte alle Standardzugaenge einrichten.', 'medium', 'open', 'Sandra Buerki', 'Karin Pfister', '2026-02-28T17:00:00', '2026-02-14T14:00:00', '2026-02-14T14:00:00', 'Zugang', { 'Raumnummer': '3.12' }, undefined, undefined, true),
  makeTicket('tk-4', 'HD-2026-0304', 'Outlook synchronisiert Kalender nicht', 'Kalendereintraege werden nicht zwischen Outlook Desktop und Mobile synchronisiert.', 'medium', 'waiting', 'Marco Hartmann', 'Andreas Mueller', '2026-02-17T12:00:00', '2026-02-14T11:30:00', '2026-02-14T16:20:00', 'E-Mail', { 'Geraetetyp': 'Laptop', 'Remotezugriff erlaubt': false }),
  makeTicket('tk-5', 'HD-2026-0305', 'Bildschirm flackert', 'Der externe Monitor (Dell 27") an Arbeitsplatz B-12 flackert sporadisch.', 'low', 'open', 'Sandra Buerki', 'Regula Vogt', '2026-02-20T17:00:00', '2026-02-14T09:00:00', '2026-02-14T09:00:00', 'Hardware', { 'Geraetetyp': 'Monitor', 'Raumnummer': 'B-12', 'Geschaetzter Aufwand (h)': 0.5 }),
  makeTicket('tk-6', 'HD-2026-0306', 'ERP-System Fehlermeldung bei Rechnungserstellung', 'Beim Erstellen von Rechnungen erscheint Fehler 500. Betrifft alle Mitarbeiter in der Buchhaltung.', 'critical', 'resolved', 'Marco Hartmann', 'Thomas Kunz', '2026-02-14T14:00:00', '2026-02-14T08:00:00', '2026-02-14T12:30:00', 'Software', { 'Geschaetzter Aufwand (h)': 4 }, 5, 'Sehr schnell geloest, danke!'),
  makeTicket('tk-7', 'HD-2026-0307', 'WLAN im Sitzungszimmer zu schwach', 'Im Sitzungszimmer "Pilatus" ist das WLAN-Signal zu schwach fuer Videokonferenzen.', 'medium', 'in_progress', 'Sandra Buerki', 'Daniel Roth', '2026-02-18T17:00:00', '2026-02-13T15:30:00', '2026-02-15T10:00:00', 'Netzwerk', { 'Raumnummer': 'Pilatus', 'Geschaetzter Aufwand (h)': 2 }),
  makeTicket('tk-8', 'HD-2026-0308', 'Software-Lizenz Adobe CC abgelaufen', 'Die Adobe Creative Cloud Lizenz fuer Marketing ist seit gestern abgelaufen.', 'high', 'waiting', 'Sandra Buerki', 'Nicole Berger', '2026-02-16T12:00:00', '2026-02-13T11:00:00', '2026-02-14T10:00:00', 'Software', { 'Geraetetyp': 'Desktop' }),
  makeTicket('tk-9', 'HD-2026-0309', 'Backup-Fehler Fileserver', 'Naechtliches Backup des Fileservers ist seit 3 Tagen fehlgeschlagen. Fehlermeldung: Speicherplatz unzureichend.', 'critical', 'resolved', 'Marco Hartmann', 'System Alert', '2026-02-13T10:00:00', '2026-02-12T06:00:00', '2026-02-13T09:00:00', 'Sicherheit', { 'Geraetetyp': 'Server', 'Geschaetzter Aufwand (h)': 6 }, 4, 'Gut geloest, haette aber etwas schneller sein koennen.'),
  makeTicket('tk-10', 'HD-2026-0310', 'Zutrittskarte funktioniert nicht', 'Zutrittskarte fuer Buero 3. OG funktioniert seit heute Morgen nicht mehr. Badge-Nr: Z-4412.', 'medium', 'open', 'Sandra Buerki', 'Eveline Stauffer', '2026-02-16T17:00:00', '2026-02-15T07:50:00', '2026-02-15T07:50:00', 'Sicherheit', { 'Raumnummer': '3. OG' }),
  makeTicket('tk-11', 'HD-2026-0311', 'Telefon-Anlage: Weiterleitung einrichten', 'Neue Rufweiterleitung fuer Empfang auf +41 44 555 12 99 einrichten (Ferienabwesenheit).', 'low', 'closed', 'Marco Hartmann', 'Ruth Eberle', '2026-02-14T17:00:00', '2026-02-12T14:00:00', '2026-02-13T11:00:00', 'Telefonie', {}, 5, 'Perfekt, vielen Dank!'),
  makeTicket('tk-12', 'HD-2026-0312', 'Laptop fuer Aussendienst bestellen', 'Neuer Laptop (ThinkPad T14) fuer Aussendienst-Mitarbeiter Reto Graf benoetigt. Inkl. Dockingstation.', 'low', 'open', 'Sandra Buerki', 'Beat Kuhn', '2026-02-25T17:00:00', '2026-02-11T16:00:00', '2026-02-11T16:00:00', 'Hardware', { 'Geraetetyp': 'Laptop' }),
  makeTicket('tk-13', 'HD-2026-0313', 'Sharepoint-Berechtigung fuer Projekt X', 'Team Projekt X braucht Zugriff auf Sharepoint-Ordner /Projekte/X. Benutzer: Meier, Huber, Keller.', 'medium', 'resolved', 'Sandra Buerki', 'Patricia Hofer', '2026-02-15T12:00:00', '2026-02-13T09:00:00', '2026-02-14T15:30:00', 'Zugang', {}, 4, ''),
  makeTicket('tk-14', 'HD-2026-0314', 'Virenwarnung auf Arbeitsplatz C-03', 'Sophos meldet verdaechtige Datei auf Arbeitsplatz C-03 (Abt. Einkauf). Quarantaene aktiv.', 'high', 'in_progress', 'Marco Hartmann', 'System Alert', '2026-02-15T14:00:00', '2026-02-15T06:30:00', '2026-02-15T08:45:00', 'Sicherheit', { 'Geraetetyp': 'Desktop', 'Raumnummer': 'C-03', 'Remotezugriff erlaubt': false, 'Geschaetzter Aufwand (h)': 2 }, undefined, undefined, true),
  makeTicket('tk-15', 'HD-2026-0315', 'Teams-Raum "Saentis" Kamera defekt', 'Die Konferenzkamera im Teams-Raum Saentis zeigt kein Bild mehr. Modell: Logitech Rally.', 'medium', 'open', 'Marco Hartmann', 'Yvonne Stocker', '2026-02-19T17:00:00', '2026-02-14T13:00:00', '2026-02-14T13:00:00', 'Hardware', { 'Geraetetyp': 'Sonstiges', 'Raumnummer': 'Saentis' }),
]

const MOCK_KB_ARTICLES: KBArticle[] = [
  { id: 'kb-1', title: 'VPN-Verbindung einrichten (Windows)', category: 'Netzwerk', excerpt: 'Schritt-fuer-Schritt Anleitung zur Einrichtung der VPN-Verbindung unter Windows 10/11 mit dem Cisco AnyConnect Client.', views: 342, published: true, updatedAt: '2026-01-20T10:00:00' },
  { id: 'kb-2', title: 'Drucker hinzufuegen im Netzwerk', category: 'Hardware', excerpt: 'So fuegen Sie einen Netzwerkdrucker unter Windows oder macOS hinzu. Inkl. Treiber-Download Links.', views: 218, published: true, updatedAt: '2026-01-15T14:00:00' },
  { id: 'kb-3', title: 'Passwort zuruecksetzen (Self-Service)', category: 'Sicherheit', excerpt: 'Anleitung zum selbststaendigen Zuruecksetzen des Active-Directory Passworts ueber das Self-Service Portal.', views: 567, published: true, updatedAt: '2026-02-01T09:00:00' },
  { id: 'kb-4', title: 'Outlook E-Mail Signatur einrichten', category: 'E-Mail', excerpt: 'Einheitliche E-Mail Signatur gemaess CI/CD-Richtlinien einrichten. Vorlagen und Anleitungen.', views: 189, published: true, updatedAt: '2025-12-10T11:00:00' },
  { id: 'kb-5', title: 'Home-Office Checkliste IT', category: 'Allgemein', excerpt: 'Checkliste fuer die IT-Ausstattung im Home-Office: VPN, Telefonie, Hardware-Anforderungen.', views: 445, published: true, updatedAt: '2026-01-08T16:00:00' },
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

export const useHelpdeskStore = create<HelpdeskStore>()(
  persist(
    () => ({
      tickets: MOCK_TICKETS,
      kbArticles: MOCK_KB_ARTICLES,
      stats: MOCK_STATS,
      cannedResponses: MOCK_CANNED_RESPONSES,
      routingRules: MOCK_ROUTING_RULES,
      businessHours: MOCK_BUSINESS_HOURS,
      holidays: MOCK_HOLIDAYS,
    }),
    { name: 'kmuhub-helpdesk' },
  ),
)
