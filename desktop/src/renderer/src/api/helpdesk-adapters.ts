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
  queueId?: string
  slaDueAt: string
  slaOverdue: boolean
  slaDays: number
  slaHours: number
  createdAt: string
  updatedAt: string
  category?: string
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
  // Fallback auf eine deterministische Zeichensumme, falls die ID nicht hex-parsbar
  // ist (z.B. Demo-IDs wie "tk-10") — sonst entsteht "HD-YYYY-0NaN".
  const year = new Date(t.created_at).getFullYear()
  const parsedId = parseInt(t.id.replace(/-/g, '').slice(0, 4), 16)
  const idBase = Number.isNaN(parsedId)
    ? Array.from(t.id).reduce((acc, c) => acc + c.charCodeAt(0), 0)
    : parsedId
  const idNum = (idBase % 9999) + 1
  const ticketNr = `HD-${year}-${String(idNum).padStart(4, '0')}`

  return {
    id: t.id,
    ticketNr,
    subject: t.subject,
    description: '',
    status: wireStatusToDisplay(t.status),
    priority: wirePriorityToDisplay(t.priority),
    assignedTo: displayUserName(t.assignee_id),
    contactName: displayUserName(t.requester_id),
    assigneeId: t.assignee_id,
    requesterId: t.requester_id,
    queueId: t.queue_id ?? undefined,
    slaDueAt: t.due_at ?? new Date(Date.now() + 8 * 3600000).toISOString(),
    slaOverdue,
    slaDays,
    slaHours,
    createdAt: t.created_at,
    updatedAt: t.updated_at,
    category: undefined,
    autoRouted: false,
    csatRating: undefined,
    csatComment: undefined,
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
