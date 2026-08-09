/**
 * Bidirectional adapters between Wire types (backend) and Display types (UI).
 *
 * Wire types: helpdesk-types.ts (snake_case, backend-aligned)
 * Display types: defined here (camelCase, UI-aligned, store-compatible)
 */
import type {
  Ticket as WireTicket,
  CannedResponse as WireCannedResponse,
  TicketStatus,
  TicketPriority,
  TicketChannel,
} from './helpdesk-types'
import { displayUserName } from '@/mocks/data/shared-ids'

// ---------------------------------------------------------------------------
// Display types (UI-facing, compatible with existing store type shapes)
// ---------------------------------------------------------------------------

export type DisplayTicket = {
  id: string
  ticketNr: string
  subject: string
  description: string
  status: 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'
  priority: 'low' | 'medium' | 'high' | 'critical'
  assignedTo: string
  contactName: string
  /** Raw wire ids for ownership checks (RBAC scope=own); display uses the
   *  resolved names above. Legacy seeds may carry free-text names here. */
  assigneeId: string | null
  requesterId: string
  /** Reply address for external requesters (empty for internal tickets). */
  requesterEmail?: string
  /** True when the requester is not a tenant user (external intake). */
  requesterIsExternal?: boolean
  queueId?: string
  slaDueAt: string
  slaOverdue: boolean
  slaDays: number
  slaHours: number
  createdAt: string
  updatedAt: string
  category?: string
  /** Origin channel the ticket came in through (Herkunft tabs). */
  channel?: TicketChannel
  autoRouted?: boolean
  csatRating?: number
  csatComment?: string
  customFields?: Record<string, string | number | boolean>
}

export type DisplayCannedResponse = {
  id: string
  title: string
  content: string
  category: string
  shortcut: string
}

// ---------------------------------------------------------------------------
// Status adapters
// ---------------------------------------------------------------------------

export function wireStatusToDisplay(s: TicketStatus): DisplayTicket['status'] {
  switch (s) {
    case 'open':   return 'open'
    case 'pending': return 'in_progress'
    case 'solved':  return 'resolved'
    case 'closed':  return 'closed'
    case 'merged':  return 'closed'
    default:        return 'open'
  }
}

export function displayStatusToWire(s: DisplayTicket['status']): TicketStatus {
  switch (s) {
    case 'open':        return 'open'
    case 'in_progress': return 'pending'
    case 'waiting':     return 'pending'
    case 'resolved':    return 'solved'
    case 'closed':      return 'closed'
    default:            return 'open'
  }
}

// ---------------------------------------------------------------------------
// Priority adapters
// ---------------------------------------------------------------------------

export function wirePriorityToDisplay(p: TicketPriority): DisplayTicket['priority'] {
  switch (p) {
    case 'low':    return 'low'
    case 'normal': return 'medium'
    case 'high':   return 'high'
    case 'urgent': return 'critical'
    default:       return 'medium'
  }
}

export function displayPriorityToWire(p: DisplayTicket['priority']): TicketPriority {
  switch (p) {
    case 'low':      return 'low'
    case 'medium':   return 'normal'
    case 'high':     return 'high'
    case 'critical': return 'urgent'
    default:         return 'normal'
  }
}

// ---------------------------------------------------------------------------
// WireTicket → DisplayTicket
// ---------------------------------------------------------------------------

export function wireTicketToDisplay(t: WireTicket): DisplayTicket {
  const now = Date.now()
  const dueMs = t.due_at ? new Date(t.due_at).getTime() : null
  const diffMs = dueMs !== null ? dueMs - now : null
  const slaOverdue = dueMs !== null ? dueMs < now : false
  const absDiffMs = diffMs !== null ? Math.abs(diffMs) : 0
  const slaDays = Math.floor(absDiffMs / (1000 * 60 * 60 * 24))
  const slaHours = Math.floor((absDiffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))

  // ticketNr: HD-YYYY-XXXX — Jahr aus created_at, lfd-Nr aus ersten 4 Hex-Chars der UUID.
  // Nicht-hex-IDs (Demo-Daten wie "hd-tk-001") brauchen einen Ersatzweg. Der war
  // eine Zeichensumme — und die ignoriert die Reihenfolge: "hd-tk-001" und
  // "hd-tk-010" ergaben dieselbe Nummer, fünf der vierzehn Demo-Tickets liefen
  // doppelt. Jetzt zuerst die laufende Nummer aus der ID lesen (die ist genau das,
  // was der Nutzer als Ticket-Nr erwartet), sonst ein positionsabhängiger Hash.
  const year = new Date(t.created_at).getFullYear()
  const parsedId = parseInt(t.id.replace(/-/g, '').slice(0, 4), 16)
  const trailing = /(\d+)$/.exec(t.id)
  let idNum: number
  if (!Number.isNaN(parsedId)) {
    idNum = (parsedId % 9999) + 1
  } else if (trailing) {
    // Demo-IDs tragen ihre laufende Nummer schon — genau das erwartet der Nutzer
    // als Ticket-Nr, statt einer aus der ID errechneten Zahl.
    idNum = Math.max(1, Number(trailing[1]) % 10000)
  } else {
    // FNV-1a: gleiche Zeichen in anderer Reihenfolge → andere Zahl.
    let h = 0x811c9dc5
    for (const c of t.id) h = Math.imul(h ^ c.charCodeAt(0), 0x01000193) >>> 0
    idNum = (h % 9999) + 1
  }
  const ticketNr = `HD-${year}-${String(idNum).padStart(4, '0')}`

  return {
    id: t.id,
    ticketNr,
    subject: t.subject,
    description: t.description ?? '',
    status: wireStatusToDisplay(t.status),
    priority: wirePriorityToDisplay(t.priority),
    assignedTo: displayUserName(t.assignee_id),
    // External requesters have no user account → carry their entered name;
    // internal tickets resolve the display name from the requester UUID.
    contactName: t.requester_name ?? displayUserName(t.requester_id),
    assigneeId: t.assignee_id,
    requesterId: t.requester_id,
    requesterEmail: t.requester_email,
    requesterIsExternal: t.requester_is_external ?? false,
    queueId: t.queue_id ?? undefined,
    slaDueAt: t.due_at ?? new Date(Date.now() + 8 * 3600000).toISOString(),
    slaOverdue,
    slaDays,
    slaHours,
    createdAt: t.created_at,
    updatedAt: t.updated_at,
    category: t.category,
    channel: t.channel ?? 'agent',
    autoRouted: false,
    csatRating: t.csat_rating,
    csatComment: t.csat_comment,
    customFields: t.custom_fields,
  }
}

// ---------------------------------------------------------------------------
// WireCannedResponse → DisplayCannedResponse
// ---------------------------------------------------------------------------

export function wireCannedToDisplay(c: WireCannedResponse): DisplayCannedResponse {
  return {
    id: c.id,
    title: c.name,
    content: c.body,
    category: 'Allgemein',
    shortcut: '',
  }
}

// ---------------------------------------------------------------------------
// DisplayCannedResponse → Wire create/update bodies
// ---------------------------------------------------------------------------

export function displayCannedToWireCreate(c: Omit<DisplayCannedResponse, 'id'>): { name: string; body: string } {
  return { name: c.title, body: c.content }
}
