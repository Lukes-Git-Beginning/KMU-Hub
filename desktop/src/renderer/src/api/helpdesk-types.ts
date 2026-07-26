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

/** Origin channel a ticket was created through (intake channels, §Ticket-Intake).
    Drives the "Herkunft" tabs in the module. */
export type TicketChannel = 'agent' | 'selfservice' | 'external'

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface Ticket {
  id: string
  tenant_id: string
  subject: string
  /** Initial problem statement shown at the top of the ticket detail. Optional —
      a ticket without one hides its description section. */
  description?: string
  status: TicketStatus
  priority: TicketPriority
  assignee_id: string | null
  /** Ownership key (RBAC scope=own). User UUID for internal tickets; for
      external requesters a synthetic id (name is carried in requester_name). */
  requester_id: string
  /** Display name of the requester. Set for external requesters who have no
      user account; internal tickets resolve the name from requester_id. */
  requester_name?: string
  /** Reply address for external requesters (public intake). */
  requester_email?: string
  /** True when the requester is not a tenant user (external/public intake). */
  requester_is_external?: boolean
  /** Origin channel (agent / self-service / external) — drives Herkunft tabs. */
  channel?: TicketChannel
  /** Category label (routing + display). */
  category?: string
  /** Intake values mapped onto helpdesk_ticket custom fields. */
  custom_fields?: Record<string, string | number | boolean>
  /** Customer satisfaction rating (1–5) submitted after close. */
  csat_rating?: number
  csat_comment?: string
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
  description?: string
  category?: string
  priority?: TicketPriority
  assignee_id?: string
  queue_id?: string
  /** Ownership key — omit to let the backend use the active session user. */
  requester_id?: string
  requester_name?: string
  requester_email?: string
  requester_is_external?: boolean
  /** Intake channel; defaults to 'agent' when created from the module. */
  channel?: TicketChannel
  custom_fields?: Record<string, string | number | boolean>
}

export interface UpdateTicketInput {
  subject?: string
  status?: TicketStatus
  priority?: TicketPriority
  assignee_id?: string
  queue_id?: string
  category?: string
  /** Partial update — merged into the ticket's existing custom_fields. */
  custom_fields?: Record<string, string | number | boolean>
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

// ---------------------------------------------------------------------------
// Business Hours
// ---------------------------------------------------------------------------

export interface DaySchedule {
  active: boolean
  start: string
  end: string
}

export interface HolidayEntry {
  id: string
  date: string
  name: string
}

/** Wire shape returned by GET/PUT /api/v1/helpdesk/business-hours */
export interface BusinessHoursResponse {
  tenant_id: string
  /** JSON-encoded {dayName: DaySchedule} map */
  schedule_json: string
  /** JSON-encoded HolidayEntry[] */
  holidays_json: string
  timezone: string
  updated_at: string | null
}

export interface UpdateBusinessHoursInput {
  schedule_json: string
  holidays_json: string
  timezone: string
}
