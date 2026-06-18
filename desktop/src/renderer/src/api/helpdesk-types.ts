/**
 * TypeScript types for the Helpdesk module.
 *
 * Mirrors backend/internal/helpdesk/models.go and service.go signatures.
 * UUIDs are strings; dates are ISO 8601 strings.
 */

// ---------------------------------------------------------------------------
// Enums / union types
// ---------------------------------------------------------------------------

export type TicketStatus = 'open' | 'pending' | 'solved' | 'closed' | 'merged'
export type TicketPriority = 'low' | 'normal' | 'high' | 'urgent'
export type SLAStatusValue = 'on_track' | 'at_risk' | 'breached'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface Ticket {
  id: string
  tenant_id: string
  subject: string
  status: TicketStatus
  priority: TicketPriority
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

export interface TicketMessage {
  id: string
  ticket_id: string
  author_id: string
  body: string
  internal: boolean
  attachments: string[]
  created_at: string
}

export interface TicketQueue {
  id: string
  tenant_id: string
  name: string
  default_assignee_id: string | null
  sla_policy_id: string | null
  created_at: string
  updated_at: string
}

export interface CannedResponse {
  id: string
  tenant_id: string
  name: string
  body: string
  created_at: string
  updated_at: string
}

export interface SLAPolicy {
  id: string
  tenant_id: string
  name: string
  first_response_mins: number
  resolution_mins: number
  business_hours: Record<string, unknown> | null
  created_at: string
  updated_at: string
}

export interface TicketSLAStatus {
  ticket_id: string
  status: SLAStatusValue
  due_at: string | null
  first_response_at: string | null
}

// ---------------------------------------------------------------------------
// Request input types
// ---------------------------------------------------------------------------

export interface CreateTicketInput {
  subject: string
  priority?: TicketPriority
  assignee_id?: string
  queue_id?: string
}

export interface UpdateTicketInput {
  subject?: string
  status?: TicketStatus
  priority?: TicketPriority
  assignee_id?: string
  queue_id?: string
}

export interface AddMessageInput {
  body: string
  internal?: boolean
  attachments?: string[]
}

export interface CreateQueueInput {
  name: string
  default_assignee_id?: string
  sla_policy_id?: string
}

export interface UpdateQueueInput {
  name?: string
  default_assignee_id?: string
  sla_policy_id?: string
}

export interface CreateCannedResponseInput {
  name: string
  body: string
}

export interface UpdateCannedResponseInput {
  name?: string
  body?: string
}

export interface CreateSLAPolicyInput {
  name: string
  first_response_mins: number
  resolution_mins: number
  business_hours?: Record<string, unknown>
}

export interface UpdateSLAPolicyInput {
  name?: string
  first_response_mins?: number
  resolution_mins?: number
  business_hours?: Record<string, unknown>
}

// ---------------------------------------------------------------------------
// Response wrapper types
// ---------------------------------------------------------------------------

export interface ListTicketsResponse {
  tickets: Ticket[]
  total: number
}

// ---------------------------------------------------------------------------
// Knowledge-base articles
// ---------------------------------------------------------------------------

export type KBArticleStatus = 'draft' | 'published'

export interface KBArticle {
  id: string
  tenant_id: string
  title: string
  content: string
  category: string
  status: KBArticleStatus
  author_id: string
  created_at: string
  updated_at: string
}

export interface CreateKBArticleInput {
  title: string
  content?: string
  category?: string
  status?: KBArticleStatus
}

export interface UpdateKBArticleInput {
  title?: string
  content?: string
  category?: string
  status?: KBArticleStatus
}

// ---------------------------------------------------------------------------
// Routing rules
// ---------------------------------------------------------------------------

export interface RoutingRule {
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

export interface CreateRoutingRuleInput {
  name: string
  conditions?: string
  target_queue_id?: string
  priority?: number
  enabled?: boolean
}

export interface UpdateRoutingRuleInput {
  name?: string
  conditions?: string
  target_queue_id?: string
  priority?: number
  enabled?: boolean
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

export interface WeeklyDayCount {
  label: string
  count: number
}

export interface HelpdeskStats {
  open_tickets: number
  avg_response_time: string
  resolved_this_week: number
  customer_satisfaction: string
  weekly_breakdown: WeeklyDayCount[]
}
