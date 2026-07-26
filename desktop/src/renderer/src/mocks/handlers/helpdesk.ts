import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import { IDS } from '../data/shared-ids'
import { getDemoSessionUserId } from '../data/rbac'
import { hoursFromNow } from '../data/date-helpers'

const API = API_BASE_URL
const BASE = `${API}/api/v1/helpdesk`

// ============================================================================
// Types (wire shapes — snake_case, aligned with helpdesk-types.ts)
// ============================================================================

type WireTicketStatus = 'open' | 'pending' | 'solved' | 'closed' | 'merged'
type WireTicketPriority = 'low' | 'normal' | 'high' | 'urgent'

interface WireTicket {
  id: string
  tenant_id: string
  subject: string
  description?: string
  status: WireTicketStatus
  priority: WireTicketPriority
  assignee_id: string | null
  requester_id: string
  queue_id: string | null
  due_at: string | null
  merged_into_id: string | null
  first_response_at: string | null
  resolved_at: string | null
  created_at: string
  updated_at: string
}

interface WireTicketMessage {
  id: string
  ticket_id: string
  author_id: string
  body: string
  internal: boolean
  attachments: string[]
  created_at: string
}

interface WireQueue {
  id: string
  tenant_id: string
  name: string
  default_assignee_id: string | null
  sla_policy_id: string | null
  created_at: string
  updated_at: string
}

interface WireCannedResponse {
  id: string
  tenant_id: string
  name: string
  body: string
  created_at: string
  updated_at: string
}

interface WireSLAPolicy {
  id: string
  tenant_id: string
  name: string
  first_response_mins: number
  resolution_mins: number
  business_hours: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

interface WireTicketSLAStatus {
  ticket_id: string
  status: 'on_track' | 'at_risk' | 'breached'
  due_at: string | null
  first_response_at: string | null
}

interface WireKBArticle {
  id: string
  tenant_id: string
  title: string
  content: string
  category: string
  status: 'draft' | 'published'
  author_id: string
  created_at: string
  updated_at: string
}

interface WireRoutingRule {
  id: string
  tenant_id: string
  name: string
  conditions: string
  target_queue_id: string | null
  priority: number
  enabled: boolean
  created_at: string
  updated_at: string
}

interface WireHelpdeskStats {
  open_tickets: number
  avg_response_time: string
  resolved_this_week: number
  customer_satisfaction: string
  weekly_breakdown: Array<{ label: string; count: number }>
}

// ============================================================================
// Mutable state
// ============================================================================

const NOW = Date.now()
const H = 3600000

// SLA due_at offsets in hours (relative to now): negative = overdue
const SLA_OFFSETS: Record<string, number> = {
  'hd-tk-001': 3,    // soon (at_risk)
  'hd-tk-002': -4,   // breached
  'hd-tk-003': 60,   // on_track
  'hd-tk-004': 30,   // on_track
  'hd-tk-005': 100,  // on_track
  'hd-tk-006': -40,  // resolved, breached
  'hd-tk-007': 20,   // on_track
  'hd-tk-008': 2,    // soon (at_risk)
  'hd-tk-009': -72,  // breached (resolved)
  'hd-tk-010': 28,   // on_track
  'hd-tk-011': -90,  // breached (closed)
  'hd-tk-012': 120,  // on_track
  'hd-tk-013': -20,  // breached (solved)
  'hd-tk-014': 1,    // very soon (at_risk)
  'hd-tk-015': 50,   // on_track
}

function slaStatus(offsetH: number): WireTicketSLAStatus['status'] {
  if (offsetH < 0) return 'breached'
  if (offsetH <= 4) return 'at_risk'
  return 'on_track'
}

const QUEUE_GENERAL = 'hd-q-001'
const QUEUE_NETWORK = 'hd-q-002'
const SLA_STANDARD = 'hd-sla-001'
const SLA_PREMIUM = 'hd-sla-002'

const queues: WireQueue[] = [
  {
    id: QUEUE_GENERAL,
    tenant_id: 'tenant-001',
    name: 'Allgemeiner IT-Support',
    default_assignee_id: IDS.users.thomas,
    sla_policy_id: SLA_STANDARD,
    created_at: new Date(NOW - 90 * 24 * H).toISOString(),
    updated_at: new Date(NOW - 90 * 24 * H).toISOString(),
  },
  {
    id: QUEUE_NETWORK,
    tenant_id: 'tenant-001',
    name: 'Netzwerk & Infrastruktur',
    default_assignee_id: IDS.users.markus,
    sla_policy_id: SLA_PREMIUM,
    created_at: new Date(NOW - 60 * 24 * H).toISOString(),
    updated_at: new Date(NOW - 60 * 24 * H).toISOString(),
  },
]

const slaPolicies: WireSLAPolicy[] = [
  {
    id: SLA_STANDARD,
    tenant_id: 'tenant-001',
    name: 'Standard SLA',
    first_response_mins: 240,   // 4 h
    resolution_mins: 2880,      // 48 h
    business_hours: null,
    created_at: new Date(NOW - 90 * 24 * H).toISOString(),
    updated_at: new Date(NOW - 90 * 24 * H).toISOString(),
  },
  {
    id: SLA_PREMIUM,
    tenant_id: 'tenant-001',
    name: 'Premium SLA (Kritisch)',
    first_response_mins: 60,    // 1 h
    resolution_mins: 480,       // 8 h
    business_hours: null,
    created_at: new Date(NOW - 60 * 24 * H).toISOString(),
    updated_at: new Date(NOW - 60 * 24 * H).toISOString(),
  },
]

const tickets: WireTicket[] = [
  {
    id: 'hd-tk-001',
    tenant_id: 'tenant-001',
    subject: 'Drucker im 2. OG druckt nicht',
    description: 'Der Drucker im 2. OG zeigt seit heute Morgen "Offline" an. Mehrere Kollegen sind betroffen und können nicht drucken.',
    status: 'pending',
    priority: 'high',
    assignee_id: IDS.users.thomas,
    requester_id: 'Brigitte Schärer',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-001'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 5 * H).toISOString(),
    resolved_at: null,
    created_at: new Date(NOW - 30 * H).toISOString(),
    updated_at: new Date(NOW - 5 * H).toISOString(),
  },
  {
    id: 'hd-tk-002',
    tenant_id: 'tenant-001',
    subject: 'VPN-Verbindung bricht ab',
    description: 'Die VPN-Verbindung trennt sich alle 10 Minuten. Konzentriertes Arbeiten im Home-Office ist so kaum möglich.',
    status: 'pending',
    priority: 'urgent',
    assignee_id: IDS.users.thomas,
    requester_id: 'Stefan Wenger',
    queue_id: QUEUE_NETWORK,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-002'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 6 * H).toISOString(),
    resolved_at: null,
    created_at: new Date(NOW - 50 * H).toISOString(),
    updated_at: new Date(NOW - 6 * H).toISOString(),
  },
  {
    id: 'hd-tk-003',
    tenant_id: 'tenant-001',
    subject: 'Neuer Mitarbeiter – Zugänge einrichten',
    status: 'open',
    priority: 'normal',
    assignee_id: IDS.users.julia,
    requester_id: IDS.users.markus, // Demo: Requester-Modell (scope=own sieht dieses Ticket als Markus)
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-003'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: null,
    resolved_at: null,
    created_at: new Date(NOW - 48 * H).toISOString(),
    updated_at: new Date(NOW - 48 * H).toISOString(),
  },
  {
    id: 'hd-tk-004',
    tenant_id: 'tenant-001',
    subject: 'Outlook synchronisiert Kalender nicht',
    description: 'Outlook synchronisiert den Kalender seit dem letzten Update nicht mehr. Termine fehlen auf dem Smartphone.',
    status: 'pending',
    priority: 'normal',
    assignee_id: IDS.users.thomas,
    requester_id: 'Andreas Müller',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-004'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 20 * H).toISOString(),
    resolved_at: null,
    created_at: new Date(NOW - 70 * H).toISOString(),
    updated_at: new Date(NOW - 20 * H).toISOString(),
  },
  {
    id: 'hd-tk-005',
    tenant_id: 'tenant-001',
    subject: 'Bildschirm flackert',
    status: 'open',
    priority: 'low',
    assignee_id: IDS.users.julia,
    requester_id: IDS.users.markus, // Demo: Requester-Modell (scope=own sieht dieses Ticket als Markus)
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-005'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: null,
    resolved_at: null,
    created_at: new Date(NOW - 80 * H).toISOString(),
    updated_at: new Date(NOW - 80 * H).toISOString(),
  },
  {
    id: 'hd-tk-006',
    tenant_id: 'tenant-001',
    subject: 'ERP-System Fehlermeldung bei Rechnungserstellung',
    description: 'Beim Erstellen einer Rechnung erscheint die Fehlermeldung "Buchungskreis nicht zugeordnet". Der Rechnungslauf ist dadurch blockiert.',
    status: 'solved',
    priority: 'urgent',
    assignee_id: IDS.users.thomas,
    requester_id: 'Thomas Kunz',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-006'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 52 * H).toISOString(),
    resolved_at: new Date(NOW - 50 * H).toISOString(),
    created_at: new Date(NOW - 54 * H).toISOString(),
    updated_at: new Date(NOW - 50 * H).toISOString(),
  },
  {
    id: 'hd-tk-007',
    tenant_id: 'tenant-001',
    subject: 'WLAN im Sitzungszimmer zu schwach',
    description: 'Das WLAN im grossen Sitzungszimmer ist zu schwach für Videocalls, die Verbindung bricht regelmässig ab.',
    status: 'pending',
    priority: 'normal',
    assignee_id: IDS.users.markus,
    requester_id: 'Daniel Roth',
    queue_id: QUEUE_NETWORK,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-007'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 8 * H).toISOString(),
    resolved_at: null,
    created_at: new Date(NOW - 96 * H).toISOString(),
    updated_at: new Date(NOW - 8 * H).toISOString(),
  },
  {
    id: 'hd-tk-008',
    tenant_id: 'tenant-001',
    subject: 'Software-Lizenz Adobe CC abgelaufen',
    description: 'Die Adobe-Creative-Cloud-Lizenz ist abgelaufen. InDesign und Photoshop lassen sich nicht mehr starten.',
    status: 'pending',
    priority: 'high',
    assignee_id: IDS.users.julia,
    requester_id: 'Nicole Berger',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-008'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 30 * H).toISOString(),
    resolved_at: null,
    created_at: new Date(NOW - 100 * H).toISOString(),
    updated_at: new Date(NOW - 30 * H).toISOString(),
  },
  {
    id: 'hd-tk-009',
    tenant_id: 'tenant-001',
    subject: 'Backup-Fehler Fileserver',
    description: 'Das nächtliche Backup des Fileservers schlägt seit drei Tagen mit einem Schreibfehler fehl.',
    status: 'solved',
    priority: 'urgent',
    assignee_id: IDS.users.thomas,
    requester_id: 'System Alert',
    queue_id: QUEUE_NETWORK,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-009'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 120 * H).toISOString(),
    resolved_at: new Date(NOW - 120 * H).toISOString(),
    created_at: new Date(NOW - 130 * H).toISOString(),
    updated_at: new Date(NOW - 120 * H).toISOString(),
  },
  {
    id: 'hd-tk-010',
    tenant_id: 'tenant-001',
    subject: 'Zutrittskarte funktioniert nicht',
    description: 'Die Zutrittskarte wird am Haupteingang nicht mehr erkannt. Der Zugang ist aktuell nur über den Empfang möglich.',
    status: 'open',
    priority: 'normal',
    assignee_id: IDS.users.julia,
    requester_id: IDS.users.markus, // Demo: Requester-Modell (scope=own sieht dieses Ticket als Markus)
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-010'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: null,
    resolved_at: null,
    created_at: new Date(NOW - 10 * H).toISOString(),
    updated_at: new Date(NOW - 10 * H).toISOString(),
  },
  {
    id: 'hd-tk-011',
    tenant_id: 'tenant-001',
    subject: 'Telefon-Anlage: Weiterleitung einrichten',
    description: 'Für die neue Empfangsnummer soll eine Weiterleitung auf die Zentrale eingerichtet werden.',
    status: 'closed',
    priority: 'low',
    assignee_id: IDS.users.thomas,
    requester_id: 'Ruth Eberle',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-011'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 110 * H).toISOString(),
    resolved_at: new Date(NOW - 100 * H).toISOString(),
    created_at: new Date(NOW - 120 * H).toISOString(),
    updated_at: new Date(NOW - 100 * H).toISOString(),
  },
  {
    id: 'hd-tk-012',
    tenant_id: 'tenant-001',
    subject: 'Laptop für Aussendienst bestellen',
    description: 'Für den neuen Aussendienst-Mitarbeiter wird ein Laptop mit Docking-Station benötigt. Das Budget ist bereits freigegeben.',
    status: 'open',
    priority: 'low',
    assignee_id: IDS.users.julia,
    requester_id: 'Beat Kuhn',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-012'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: null,
    resolved_at: null,
    created_at: new Date(NOW - 140 * H).toISOString(),
    updated_at: new Date(NOW - 140 * H).toISOString(),
  },
  {
    id: 'hd-tk-013',
    tenant_id: 'tenant-001',
    subject: 'Sharepoint-Berechtigung für Projekt X',
    description: 'Das Projektteam X braucht Lese- und Schreibrechte auf die Sharepoint-Ablage "Projekt X".',
    status: 'solved',
    priority: 'normal',
    assignee_id: IDS.users.julia,
    requester_id: 'Patricia Hofer',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-013'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 50 * H).toISOString(),
    resolved_at: new Date(NOW - 30 * H).toISOString(),
    created_at: new Date(NOW - 60 * H).toISOString(),
    updated_at: new Date(NOW - 30 * H).toISOString(),
  },
  {
    id: 'hd-tk-014',
    tenant_id: 'tenant-001',
    subject: 'Virenwarnung auf Arbeitsplatz C-03',
    description: 'Der Virenscanner meldet auf Arbeitsplatz C-03 eine Bedrohung. Das Gerät wurde vorsorglich vom Netz getrennt.',
    status: 'pending',
    priority: 'high',
    assignee_id: IDS.users.thomas,
    requester_id: 'System Alert',
    queue_id: QUEUE_NETWORK,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-014'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: new Date(NOW - 6 * H).toISOString(),
    resolved_at: null,
    created_at: new Date(NOW - 30 * H).toISOString(),
    updated_at: new Date(NOW - 6 * H).toISOString(),
  },
  {
    id: 'hd-tk-015',
    tenant_id: 'tenant-001',
    subject: 'Teams-Raum "Säntis" Kamera defekt',
    description: 'Die Deckenkamera im Teams-Raum "Säntis" zeigt kein Bild mehr. Nächste Woche stehen wichtige Kundencalls an.',
    status: 'open',
    priority: 'normal',
    assignee_id: IDS.users.markus,
    requester_id: 'Yvonne Stocker',
    queue_id: QUEUE_GENERAL,
    due_at: new Date(NOW + SLA_OFFSETS['hd-tk-015'] * H).toISOString(),
    merged_into_id: null,
    first_response_at: null,
    resolved_at: null,
    created_at: new Date(NOW - 44 * H).toISOString(),
    updated_at: new Date(NOW - 44 * H).toISOString(),
  },
]

// Ticket messages — full conversations for tk-001 and tk-002, single opening for others
const ticketMessages: Record<string, WireTicketMessage[]> = {
  'hd-tk-001': [
    { id: 'hd-msg-001-1', ticket_id: 'hd-tk-001', author_id: 'Brigitte Schärer', body: 'Der Drucker im 2. OG zeigt seit heute Morgen "Offline" an. Mehrere Kollegen haben das gleiche Problem.', internal: false, attachments: [], created_at: new Date(NOW - 30 * H).toISOString() },
    { id: 'hd-msg-001-2', ticket_id: 'hd-tk-001', author_id: IDS.users.thomas, body: 'Danke für die Meldung. Ich prüfe den Druckserver und die Netzwerkverbindung. Bitte kurz Geduld.', internal: false, attachments: [], created_at: new Date(NOW - 25 * H).toISOString() },
    { id: 'hd-msg-001-3', ticket_id: 'hd-tk-001', author_id: IDS.users.thomas, body: 'Druckspooler analysiert — Dienst war abgestürzt. Neustart läuft.', internal: true, attachments: [], created_at: new Date(NOW - 5 * H).toISOString() },
    { id: 'hd-msg-001-4', ticket_id: 'hd-tk-001', author_id: IDS.users.thomas, body: 'Der Druckspooler-Dienst wurde neu gestartet. Bitte testen Sie erneut.', internal: false, attachments: [], created_at: new Date(NOW - 5 * H).toISOString() },
  ],
  'hd-tk-002': [
    { id: 'hd-msg-002-1', ticket_id: 'hd-tk-002', author_id: 'Stefan Wenger', body: 'Meine VPN-Verbindung trennt sich alle 10 Minuten. Arbeiten im Home-Office ist so nicht möglich.', internal: false, attachments: [], created_at: new Date(NOW - 50 * H).toISOString() },
    { id: 'hd-msg-002-2', ticket_id: 'hd-tk-002', author_id: IDS.users.thomas, body: 'Welchen VPN-Client nutzen Sie? Bitte senden Sie mir die Logdatei.', internal: false, attachments: [], created_at: new Date(NOW - 44 * H).toISOString() },
    { id: 'hd-msg-002-3', ticket_id: 'hd-tk-002', author_id: 'Stefan Wenger', body: 'Cisco AnyConnect 4.10. Logs sind im Anhang.', internal: false, attachments: [], created_at: new Date(NOW - 40 * H).toISOString() },
    { id: 'hd-msg-002-4', ticket_id: 'hd-tk-002', author_id: IDS.users.thomas, body: 'Logs ausgewertet — MTU-Mismatch auf dem Gateway. Ticket an Netzwerk-Team eskaliert.', internal: true, attachments: [], created_at: new Date(NOW - 6 * H).toISOString() },
  ],
  'hd-tk-003': [
    { id: 'hd-msg-003-1', ticket_id: 'hd-tk-003', author_id: 'Karin Pfister', body: 'Neuer Mitarbeiter Lukas Meier startet am 01.03. in der Buchhaltung. Bitte alle Standardzugänge einrichten.', internal: false, attachments: [], created_at: new Date(NOW - 48 * H).toISOString() },
    { id: 'hd-msg-003-2', ticket_id: 'hd-tk-003', author_id: IDS.users.julia, body: 'Wird vorbereitet. AD-Konto, E-Mail, ERP und Zeiterfassung.', internal: true, attachments: [], created_at: new Date(NOW - 46 * H).toISOString() },
  ],
  'hd-tk-004': [
    { id: 'hd-msg-004-1', ticket_id: 'hd-tk-004', author_id: 'Andreas Müller', body: 'Termine aus Outlook Desktop tauchen auf dem iPhone nicht auf. Andersrum genauso.', internal: false, attachments: [], created_at: new Date(NOW - 70 * H).toISOString() },
    { id: 'hd-msg-004-2', ticket_id: 'hd-tk-004', author_id: IDS.users.thomas, body: 'Das klingt nach einem Profilfehler in der Exchange-Synchronisation. Ich brauche das Modell Ihres iPhones.', internal: false, attachments: [], created_at: new Date(NOW - 20 * H).toISOString() },
  ],
  'hd-tk-005': [
    { id: 'hd-msg-005-1', ticket_id: 'hd-tk-005', author_id: 'Regula Vogt', body: 'Der Dell-Monitor an B-12 flackert sporadisch, etwa alle paar Minuten kurz.', internal: false, attachments: [], created_at: new Date(NOW - 80 * H).toISOString() },
  ],
  'hd-tk-006': [
    { id: 'hd-msg-006-1', ticket_id: 'hd-tk-006', author_id: 'Thomas Kunz', body: 'Beim Erstellen von Rechnungen kommt Fehler 500. Die ganze Buchhaltung steht.', internal: false, attachments: [], created_at: new Date(NOW - 54 * H).toISOString() },
    { id: 'hd-msg-006-2', ticket_id: 'hd-tk-006', author_id: IDS.users.thomas, body: 'Der ERP-Applikationsdienst hatte sich aufgehängt. Neustart durchgeführt, Rechnungslauf läuft wieder.', internal: false, attachments: [], created_at: new Date(NOW - 50 * H).toISOString() },
    { id: 'hd-msg-006-3', ticket_id: 'hd-tk-006', author_id: 'Thomas Kunz', body: 'Bestätigt, funktioniert wieder. Vielen Dank für die schnelle Hilfe!', internal: false, attachments: [], created_at: new Date(NOW - 49 * H).toISOString() },
  ],
  'hd-tk-007': [
    { id: 'hd-msg-007-1', ticket_id: 'hd-tk-007', author_id: 'Daniel Roth', body: 'Im Sitzungszimmer Pilatus bricht das WLAN bei Videokonferenzen ständig ab.', internal: false, attachments: [], created_at: new Date(NOW - 96 * H).toISOString() },
    { id: 'hd-msg-007-2', ticket_id: 'hd-tk-007', author_id: IDS.users.markus, body: 'Wir prüfen die Access-Point-Abdeckung. Eventuell ist ein zusätzlicher AP nötig.', internal: true, attachments: [], created_at: new Date(NOW - 8 * H).toISOString() },
  ],
  'hd-tk-008': [
    { id: 'hd-msg-008-1', ticket_id: 'hd-tk-008', author_id: 'Nicole Berger', body: 'Die Adobe-CC-Lizenz fürs Marketing ist abgelaufen, niemand kann mehr arbeiten.', internal: false, attachments: [], created_at: new Date(NOW - 100 * H).toISOString() },
    { id: 'hd-msg-008-2', ticket_id: 'hd-tk-008', author_id: IDS.users.julia, body: 'Verlängerung ist bei der Beschaffung angefragt. Ich warte auf die Budget-Freigabe.', internal: false, attachments: [], created_at: new Date(NOW - 30 * H).toISOString() },
  ],
  'hd-tk-009': [
    { id: 'hd-msg-009-1', ticket_id: 'hd-tk-009', author_id: 'System Alert', body: 'Automatische Meldung: Nächtliches Backup des Fileservers seit 3 Tagen fehlgeschlagen (Speicherplatz unzureichend).', internal: false, attachments: [], created_at: new Date(NOW - 130 * H).toISOString() },
    { id: 'hd-msg-009-2', ticket_id: 'hd-tk-009', author_id: IDS.users.thomas, body: 'Alte Snapshots bereinigt, Backup-Ziel erweitert. Nächster Lauf erfolgreich verifiziert.', internal: false, attachments: [], created_at: new Date(NOW - 120 * H).toISOString() },
  ],
  'hd-tk-010': [
    { id: 'hd-msg-010-1', ticket_id: 'hd-tk-010', author_id: 'Eveline Stauffer', body: 'Meine Zutrittskarte (Z-4412) öffnet das Büro im 3. OG seit heute Morgen nicht mehr.', internal: false, attachments: [], created_at: new Date(NOW - 10 * H).toISOString() },
  ],
  'hd-tk-011': [
    { id: 'hd-msg-011-1', ticket_id: 'hd-tk-011', author_id: 'Ruth Eberle', body: 'Bitte eine Rufweiterleitung für den Empfang auf +49 89 555 12 99 einrichten (Urlaub).', internal: false, attachments: [], created_at: new Date(NOW - 120 * H).toISOString() },
    { id: 'hd-msg-011-2', ticket_id: 'hd-tk-011', author_id: IDS.users.thomas, body: 'Weiterleitung ist aktiv und getestet. Gilt bis auf Widerruf.', internal: false, attachments: [], created_at: new Date(NOW - 100 * H).toISOString() },
  ],
  'hd-tk-012': [
    { id: 'hd-msg-012-1', ticket_id: 'hd-tk-012', author_id: 'Beat Kuhn', body: 'Für den Aussendienst-Kollegen Reto Graf brauchen wir ein ThinkPad T14 inkl. Dockingstation.', internal: false, attachments: [], created_at: new Date(NOW - 140 * H).toISOString() },
  ],
  'hd-tk-013': [
    { id: 'hd-msg-013-1', ticket_id: 'hd-tk-013', author_id: 'Patricia Hofer', body: 'Das Team von Projekt X braucht Zugriff auf den Sharepoint-Ordner /Projekte/X.', internal: false, attachments: [], created_at: new Date(NOW - 60 * H).toISOString() },
    { id: 'hd-msg-013-2', ticket_id: 'hd-tk-013', author_id: IDS.users.julia, body: 'Berechtigungen für Meier, Huber und Keller gesetzt. Bitte einmal abmelden und neu anmelden.', internal: false, attachments: [], created_at: new Date(NOW - 30 * H).toISOString() },
  ],
  'hd-tk-014': [
    { id: 'hd-msg-014-1', ticket_id: 'hd-tk-014', author_id: 'System Alert', body: 'Sophos meldet eine verdächtige Datei auf Arbeitsplatz C-03 (Einkauf). Quarantäne ist aktiv.', internal: false, attachments: [], created_at: new Date(NOW - 30 * H).toISOString() },
    { id: 'hd-msg-014-2', ticket_id: 'hd-tk-014', author_id: IDS.users.thomas, body: 'Datei analysiert — False Positive durch ein Makro. Ich whiteliste die Signatur nach Rücksprache.', internal: true, attachments: [], created_at: new Date(NOW - 6 * H).toISOString() },
  ],
  'hd-tk-015': [
    { id: 'hd-msg-015-1', ticket_id: 'hd-tk-015', author_id: 'Yvonne Stocker', body: 'Die Logitech-Rally-Kamera im Teams-Raum Säntis zeigt kein Bild mehr.', internal: false, attachments: [], created_at: new Date(NOW - 44 * H).toISOString() },
  ],
}

// Wire-shape canned responses (name/body — NOT title/content)
const cannedResponses: WireCannedResponse[] = [
  { id: 'hd-cr-001', tenant_id: 'tenant-001', name: 'Begrüßung', body: '<p>Guten Tag,</p><p>vielen Dank für Ihre Anfrage. Wir haben Ihr Ticket erhalten und werden uns so schnell wie möglich darum kümmern.</p><p>Freundliche Grüße,<br/>IT Helpdesk</p>', created_at: new Date(NOW - 90 * 24 * H).toISOString(), updated_at: new Date(NOW - 90 * 24 * H).toISOString() },
  { id: 'hd-cr-002', tenant_id: 'tenant-001', name: 'Ticket geschlossen', body: '<p>Guten Tag,</p><p>Ihr Ticket wurde bearbeitet und geschlossen. Sollte das Problem erneut auftreten, eröffnen Sie bitte ein neues Ticket mit Verweis auf diese Ticketnummer.</p><p>Freundliche Grüße,<br/>IT Helpdesk</p>', created_at: new Date(NOW - 90 * 24 * H).toISOString(), updated_at: new Date(NOW - 90 * 24 * H).toISOString() },
  { id: 'hd-cr-003', tenant_id: 'tenant-001', name: 'VPN Troubleshooting', body: '<p>Bitte führen Sie folgende Schritte aus:</p><ol><li>VPN-Client komplett beenden (Taskleiste prüfen)</li><li>Internetverbindung prüfen</li><li>VPN-Client neu starten</li><li>Server-Adresse: <strong>vpn.firma.de</strong></li></ol><p>Falls das Problem weiterhin besteht, senden Sie bitte die Logdateien.</p>', created_at: new Date(NOW - 80 * 24 * H).toISOString(), updated_at: new Date(NOW - 80 * 24 * H).toISOString() },
  { id: 'hd-cr-004', tenant_id: 'tenant-001', name: 'Passwort-Reset Anleitung', body: '<p>Sie können Ihr Passwort selbst zurücksetzen:</p><ol><li>Besuchen Sie <strong>https://password.firma.de</strong></li><li>Geben Sie Ihren Benutzernamen ein</li><li>Bestätigen Sie per SMS-Code</li><li>Setzen Sie ein neues Passwort (mind. 12 Zeichen)</li></ol>', created_at: new Date(NOW - 70 * 24 * H).toISOString(), updated_at: new Date(NOW - 70 * 24 * H).toISOString() },
  { id: 'hd-cr-005', tenant_id: 'tenant-001', name: 'Hardware-Bestellung', body: '<p>Vielen Dank für die Anfrage. Für Hardware-Bestellungen benötigen wir:</p><ul><li>Genaue Gerätebezeichnung</li><li>Kostenstelle / Budget-Freigabe</li><li>Gewünschtes Lieferdatum</li><li>Genehmigung des Vorgesetzten</li></ul>', created_at: new Date(NOW - 60 * 24 * H).toISOString(), updated_at: new Date(NOW - 60 * 24 * H).toISOString() },
  { id: 'hd-cr-006', tenant_id: 'tenant-001', name: 'Rückfrage', body: '<p>Guten Tag,</p><p>Für die weitere Bearbeitung benötigen wir noch folgende Informationen:</p><ul><li>[Bitte ergänzen]</li></ul><p>Bitte antworten Sie direkt auf dieses Ticket.</p>', created_at: new Date(NOW - 50 * 24 * H).toISOString(), updated_at: new Date(NOW - 50 * 24 * H).toISOString() },
]

// ============================================================================
// Helpers
// ============================================================================

function paginate<T>(
  items: T[],
  page: number,
  pageSize: number,
): { items: T[]; total: number } {
  const start = (page - 1) * pageSize
  return { items: items.slice(start, start + pageSize), total: items.length }
}

// ============================================================================
// Handlers
// ============================================================================

export const helpdeskHandlers = [
  // --- Tickets: list ---

  http.get(`${BASE}/tickets`, ({ request }) => {
    const url = new URL(request.url)
    const statusFilter = url.searchParams.get('status') as WireTicketStatus | null
    const page = Number(url.searchParams.get('page') ?? 1)
    const pageSize = Number(url.searchParams.get('page_size') ?? 50)

    let filtered = tickets
    if (statusFilter) {
      filtered = tickets.filter((t) => t.status === statusFilter)
    }

    const result = paginate(filtered, page, pageSize)
    return HttpResponse.json({ tickets: result.items, total: result.total })
  }),

  // --- Tickets: single ---

  http.get(`${BASE}/tickets/:id`, ({ params }) => {
    const ticket = tickets.find((t) => t.id === params.id)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    return HttpResponse.json(ticket)
  }),

  // --- Tickets: create ---

  http.post(`${BASE}/tickets`, async ({ request }) => {
    const body = (await request.json()) as {
      subject?: string
      priority?: WireTicketPriority
      assignee_id?: string
      queue_id?: string
    }
    const now = new Date().toISOString()
    const newTicket: WireTicket = {
      id: `hd-tk-${Date.now()}`,
      tenant_id: 'tenant-001',
      subject: body.subject ?? 'Neues Ticket',
      status: 'open',
      priority: body.priority ?? 'normal',
      assignee_id: body.assignee_id ?? null,
      // Requester = the active demo session (RBAC scope=own: creators must
      // see their own tickets, so never hardcode a fixed user here).
      requester_id: getDemoSessionUserId(),
      queue_id: body.queue_id ?? null,
      due_at: hoursFromNow(8),
      merged_into_id: null,
      first_response_at: null,
      resolved_at: null,
      created_at: now,
      updated_at: now,
    }
    tickets.unshift(newTicket)
    ticketMessages[newTicket.id] = []
    return HttpResponse.json(newTicket, { status: 201 })
  }),

  // --- Tickets: update ---

  http.put(`${BASE}/tickets/:id`, async ({ params, request }) => {
    const ticket = tickets.find((t) => t.id === params.id)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    const body = (await request.json()) as Partial<WireTicket>
    if (body.subject != null) ticket.subject = body.subject
    if (body.status != null) ticket.status = body.status
    if (body.priority != null) ticket.priority = body.priority
    if (body.assignee_id !== undefined) ticket.assignee_id = body.assignee_id
    if (body.queue_id !== undefined) ticket.queue_id = body.queue_id
    ticket.updated_at = new Date().toISOString()
    return HttpResponse.json(ticket)
  }),

  // --- Tickets: close ---

  http.post(`${BASE}/tickets/:id/close`, ({ params }) => {
    const ticket = tickets.find((t) => t.id === params.id)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    ticket.status = 'closed'
    ticket.resolved_at = ticket.resolved_at ?? new Date().toISOString()
    ticket.updated_at = new Date().toISOString()
    return HttpResponse.json(ticket)
  }),

  // --- Tickets: reopen ---

  http.post(`${BASE}/tickets/:id/reopen`, ({ params }) => {
    const ticket = tickets.find((t) => t.id === params.id)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    ticket.status = 'open'
    ticket.resolved_at = null
    ticket.updated_at = new Date().toISOString()
    return HttpResponse.json(ticket)
  }),

  // --- Tickets: assign ---

  http.post(`${BASE}/tickets/:id/assign`, async ({ params, request }) => {
    const ticket = tickets.find((t) => t.id === params.id)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    const body = (await request.json()) as { assignee_id: string }
    ticket.assignee_id = body.assignee_id
    ticket.updated_at = new Date().toISOString()
    return HttpResponse.json(ticket)
  }),

  // --- Tickets: merge ---

  http.post(`${BASE}/tickets/:id/merge`, async ({ params, request }) => {
    const source = tickets.find((t) => t.id === params.id)
    if (!source) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    const body = (await request.json()) as { target_ticket_id: string }
    source.status = 'merged'
    source.merged_into_id = body.target_ticket_id
    source.updated_at = new Date().toISOString()
    return HttpResponse.json(undefined, { status: 204 })
  }),

  // --- Tickets: apply SLA ---

  http.post(`${BASE}/tickets/:id/sla`, async ({ params, request }) => {
    const ticket = tickets.find((t) => t.id === params.id)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    const body = (await request.json()) as { sla_policy_id: string }
    const policy = slaPolicies.find((p) => p.id === body.sla_policy_id)
    if (policy) {
      ticket.due_at = hoursFromNow(policy.resolution_mins / 60)
    }
    ticket.updated_at = new Date().toISOString()
    return HttpResponse.json(ticket)
  }),

  // --- Tickets: SLA status ---

  http.get(`${BASE}/tickets/:id/sla-status`, ({ params }) => {
    const ticket = tickets.find((t) => t.id === params.id)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    const offsetH = SLA_OFFSETS[ticket.id] ?? 24
    const response: WireTicketSLAStatus = {
      ticket_id: ticket.id,
      status: slaStatus(offsetH),
      due_at: ticket.due_at,
      first_response_at: ticket.first_response_at,
    }
    return HttpResponse.json(response)
  }),

  // --- Messages: list ---

  http.get(`${BASE}/tickets/:id/messages`, ({ params }) => {
    const msgs = ticketMessages[params.id as string] ?? []
    return HttpResponse.json(msgs)
  }),

  // --- Messages: add ---

  http.post(`${BASE}/tickets/:id/messages`, async ({ params, request }) => {
    const ticketId = params.id as string
    const ticket = tickets.find((t) => t.id === ticketId)
    if (!ticket) return HttpResponse.json({ error: 'ticket not found' }, { status: 404 })
    const body = (await request.json()) as {
      body: string
      internal?: boolean
      attachments?: string[]
    }
    const msg: WireTicketMessage = {
      id: `hd-msg-${Date.now()}`,
      ticket_id: ticketId,
      author_id: IDS.users.stefan,
      body: body.body,
      internal: body.internal ?? false,
      attachments: body.attachments ?? [],
      created_at: new Date().toISOString(),
    }
    if (!ticketMessages[ticketId]) ticketMessages[ticketId] = []
    ticketMessages[ticketId].push(msg)
    ticket.updated_at = msg.created_at
    if (!ticket.first_response_at) ticket.first_response_at = msg.created_at
    return HttpResponse.json(msg, { status: 201 })
  }),

  // --- Queues ---

  http.get(`${BASE}/queues`, () => {
    return HttpResponse.json(queues)
  }),

  http.post(`${BASE}/queues`, async ({ request }) => {
    const body = (await request.json()) as {
      name: string
      default_assignee_id?: string
      sla_policy_id?: string
    }
    const now = new Date().toISOString()
    const q: WireQueue = {
      id: `hd-q-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: body.name,
      default_assignee_id: body.default_assignee_id ?? null,
      sla_policy_id: body.sla_policy_id ?? null,
      created_at: now,
      updated_at: now,
    }
    queues.push(q)
    return HttpResponse.json(q, { status: 201 })
  }),

  http.put(`${BASE}/queues/:id`, async ({ params, request }) => {
    const q = queues.find((x) => x.id === params.id)
    if (!q) return HttpResponse.json({ error: 'queue not found' }, { status: 404 })
    const body = (await request.json()) as Partial<WireQueue>
    if (body.name != null) q.name = body.name
    if (body.default_assignee_id !== undefined) q.default_assignee_id = body.default_assignee_id
    if (body.sla_policy_id !== undefined) q.sla_policy_id = body.sla_policy_id
    q.updated_at = new Date().toISOString()
    return HttpResponse.json(q)
  }),

  http.delete(`${BASE}/queues/:id`, ({ params }) => {
    const idx = queues.findIndex((x) => x.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'queue not found' }, { status: 404 })
    queues.splice(idx, 1)
    return HttpResponse.json(undefined, { status: 204 })
  }),

  // --- Canned responses ---

  http.get(`${BASE}/canned-responses`, () => {
    return HttpResponse.json(cannedResponses)
  }),

  http.post(`${BASE}/canned-responses`, async ({ request }) => {
    const body = (await request.json()) as { name: string; body: string }
    const now = new Date().toISOString()
    const cr: WireCannedResponse = {
      id: `hd-cr-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: body.name,
      body: body.body,
      created_at: now,
      updated_at: now,
    }
    cannedResponses.push(cr)
    return HttpResponse.json(cr, { status: 201 })
  }),

  http.put(`${BASE}/canned-responses/:id`, async ({ params, request }) => {
    const cr = cannedResponses.find((x) => x.id === params.id)
    if (!cr) return HttpResponse.json({ error: 'canned response not found' }, { status: 404 })
    const body = (await request.json()) as Partial<WireCannedResponse>
    if (body.name != null) cr.name = body.name
    if (body.body != null) cr.body = body.body
    cr.updated_at = new Date().toISOString()
    return HttpResponse.json(cr)
  }),

  http.delete(`${BASE}/canned-responses/:id`, ({ params }) => {
    const idx = cannedResponses.findIndex((x) => x.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'canned response not found' }, { status: 404 })
    cannedResponses.splice(idx, 1)
    return HttpResponse.json(undefined, { status: 204 })
  }),

  // --- SLA policies ---

  http.get(`${BASE}/sla-policies`, () => {
    return HttpResponse.json(slaPolicies)
  }),

  http.post(`${BASE}/sla-policies`, async ({ request }) => {
    const body = (await request.json()) as {
      name: string
      first_response_mins: number
      resolution_mins: number
      business_hours?: Record<string, unknown>
    }
    const now = new Date().toISOString()
    const policy: WireSLAPolicy = {
      id: `hd-sla-${Date.now()}`,
      tenant_id: 'tenant-001',
      name: body.name,
      first_response_mins: body.first_response_mins,
      resolution_mins: body.resolution_mins,
      business_hours: body.business_hours ?? null,
      created_at: now,
      updated_at: now,
    }
    slaPolicies.push(policy)
    return HttpResponse.json(policy, { status: 201 })
  }),

  http.put(`${BASE}/sla-policies/:id`, async ({ params, request }) => {
    const policy = slaPolicies.find((x) => x.id === params.id)
    if (!policy) return HttpResponse.json({ error: 'sla policy not found' }, { status: 404 })
    const body = (await request.json()) as Partial<WireSLAPolicy>
    if (body.name != null) policy.name = body.name
    if (body.first_response_mins != null) policy.first_response_mins = body.first_response_mins
    if (body.resolution_mins != null) policy.resolution_mins = body.resolution_mins
    if (body.business_hours !== undefined) policy.business_hours = body.business_hours
    policy.updated_at = new Date().toISOString()
    return HttpResponse.json(policy)
  }),

  http.delete(`${BASE}/sla-policies/:id`, ({ params }) => {
    const idx = slaPolicies.findIndex((x) => x.id === params.id)
    if (idx === -1) return HttpResponse.json({ error: 'sla policy not found' }, { status: 404 })
    slaPolicies.splice(idx, 1)
    return HttpResponse.json(undefined, { status: 204 })
  }),

  // --- KB Articles ---

  ...(() => {
    const kbArticles: WireKBArticle[] = [
      {
        id: 'hd-kb-001',
        tenant_id: 'tenant-001',
        title: 'VPN-Einrichtung für Mitarbeiter',
        content: 'Anleitung zur Einrichtung des VPN-Zugangs mit Cisco AnyConnect.',
        category: 'Netzwerk',
        status: 'published',
        author_id: IDS.users.admin,
        created_at: new Date(Date.now() - 30 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 5 * 24 * 3600000).toISOString(),
      },
      {
        id: 'hd-kb-002',
        tenant_id: 'tenant-001',
        title: 'Netzwerkdrucker einrichten',
        content: 'Schritt-für-Schritt-Anleitung für Windows und macOS.',
        category: 'Hardware',
        status: 'published',
        author_id: IDS.users.admin,
        created_at: new Date(Date.now() - 20 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 3 * 24 * 3600000).toISOString(),
      },
      {
        id: 'hd-kb-003',
        tenant_id: 'tenant-001',
        title: 'Passwort zurücksetzen',
        content: 'Self-Service-Portal für Passwort-Reset.',
        category: 'Sicherheit',
        status: 'published',
        author_id: IDS.users.admin,
        created_at: new Date(Date.now() - 15 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 2 * 24 * 3600000).toISOString(),
      },
      {
        id: 'hd-kb-004',
        tenant_id: 'tenant-001',
        title: 'E-Mail-Signatur einrichten',
        content: 'Offizielle Vorlage und Einrichtungsanleitung für Outlook.',
        category: 'E-Mail',
        status: 'draft',
        author_id: IDS.users.admin,
        created_at: new Date(Date.now() - 10 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 1 * 24 * 3600000).toISOString(),
      },
      {
        id: 'hd-kb-005',
        tenant_id: 'tenant-001',
        title: 'Home-Office IT-Checkliste',
        content: 'Alle wichtigen Punkte für sicheres Arbeiten im Home-Office.',
        category: 'Allgemein',
        status: 'published',
        author_id: IDS.users.admin,
        created_at: new Date(Date.now() - 7 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 7 * 24 * 3600000).toISOString(),
      },
    ]

    return [
      http.get(`${BASE}/kb-articles`, () => {
        return HttpResponse.json({ articles: kbArticles })
      }),

      http.post(`${BASE}/kb-articles`, async ({ request }) => {
        const body = (await request.json()) as Partial<WireKBArticle>
        const now = new Date().toISOString()
        const article: WireKBArticle = {
          id: `hd-kb-${Date.now()}`,
          tenant_id: 'tenant-001',
          title: body.title ?? '',
          content: body.content ?? '',
          category: body.category ?? '',
          status: body.status ?? 'draft',
          author_id: IDS.users.admin,
          created_at: now,
          updated_at: now,
        }
        kbArticles.push(article)
        return HttpResponse.json(article, { status: 201 })
      }),

      http.put(`${BASE}/kb-articles/:id`, async ({ params, request }) => {
        const article = kbArticles.find((x) => x.id === params.id)
        if (!article) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
        const body = (await request.json()) as Partial<WireKBArticle>
        if (body.title != null) article.title = body.title
        if (body.content != null) article.content = body.content
        if (body.category != null) article.category = body.category
        if (body.status != null) article.status = body.status
        article.updated_at = new Date().toISOString()
        return HttpResponse.json(article)
      }),

      http.delete(`${BASE}/kb-articles/:id`, ({ params }) => {
        const idx = kbArticles.findIndex((x) => x.id === params.id)
        if (idx === -1) return HttpResponse.json({ error: 'article not found' }, { status: 404 })
        kbArticles.splice(idx, 1)
        return HttpResponse.json(undefined, { status: 204 })
      }),
    ]
  })(),

  // --- Routing Rules ---

  ...(() => {
    const routingRules: WireRoutingRule[] = [
      {
        id: 'hd-rr-001',
        tenant_id: 'tenant-001',
        name: 'Netzwerk → Support-Team',
        conditions: JSON.stringify({ category: 'Netzwerk' }),
        target_queue_id: null,
        priority: 1,
        enabled: true,
        created_at: new Date(Date.now() - 30 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 5 * 24 * 3600000).toISOString(),
      },
      {
        id: 'hd-rr-002',
        tenant_id: 'tenant-001',
        name: 'Sicherheit → Kritisch eskalieren',
        conditions: JSON.stringify({ category: 'Sicherheit' }),
        target_queue_id: null,
        priority: 2,
        enabled: true,
        created_at: new Date(Date.now() - 20 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 2 * 24 * 3600000).toISOString(),
      },
      {
        id: 'hd-rr-003',
        tenant_id: 'tenant-001',
        name: 'Hardware → Außendienst',
        conditions: JSON.stringify({ category: 'Hardware' }),
        target_queue_id: null,
        priority: 3,
        enabled: false,
        created_at: new Date(Date.now() - 10 * 24 * 3600000).toISOString(),
        updated_at: new Date(Date.now() - 10 * 24 * 3600000).toISOString(),
      },
    ]

    return [
      http.get(`${BASE}/routing-rules`, () => {
        return HttpResponse.json({ rules: routingRules })
      }),

      http.post(`${BASE}/routing-rules`, async ({ request }) => {
        const body = (await request.json()) as Partial<WireRoutingRule>
        const now = new Date().toISOString()
        const rule: WireRoutingRule = {
          id: `hd-rr-${Date.now()}`,
          tenant_id: 'tenant-001',
          name: body.name ?? '',
          conditions: body.conditions ?? '{}',
          target_queue_id: body.target_queue_id ?? null,
          priority: body.priority ?? 0,
          enabled: body.enabled ?? true,
          created_at: now,
          updated_at: now,
        }
        routingRules.push(rule)
        return HttpResponse.json(rule, { status: 201 })
      }),

      http.put(`${BASE}/routing-rules/:id`, async ({ params, request }) => {
        const rule = routingRules.find((x) => x.id === params.id)
        if (!rule) return HttpResponse.json({ error: 'routing rule not found' }, { status: 404 })
        const body = (await request.json()) as Partial<WireRoutingRule>
        if (body.name != null) rule.name = body.name
        if (body.conditions != null) rule.conditions = body.conditions
        if (body.target_queue_id !== undefined) rule.target_queue_id = body.target_queue_id
        if (body.priority != null) rule.priority = body.priority
        if (body.enabled != null) rule.enabled = body.enabled
        rule.updated_at = new Date().toISOString()
        return HttpResponse.json(rule)
      }),

      http.delete(`${BASE}/routing-rules/:id`, ({ params }) => {
        const idx = routingRules.findIndex((x) => x.id === params.id)
        if (idx === -1) return HttpResponse.json({ error: 'routing rule not found' }, { status: 404 })
        routingRules.splice(idx, 1)
        return HttpResponse.json(undefined, { status: 204 })
      }),
    ]
  })(),

  // --- Stats ---

  http.get(`${BASE}/stats`, () => {
    const stats: WireHelpdeskStats = {
      open_tickets: 8,
      avg_response_time: '1.4 h',
      resolved_this_week: 23,
      customer_satisfaction: '4.6/5',
      weekly_breakdown: [
        { label: 'Mo', count: 5 },
        { label: 'Di', count: 8 },
        { label: 'Mi', count: 12 },
        { label: 'Do', count: 6 },
        { label: 'Fr', count: 9 },
        { label: 'Sa', count: 2 },
        { label: 'So', count: 1 },
      ],
    }
    return HttpResponse.json(stats)
  }),
]
