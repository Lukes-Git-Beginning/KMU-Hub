/**
 * TypeScript types for the Unified Inbox (Kommunikation) module.
 *
 * These types match the proto/API response shapes defined in the
 * inbox.v1.InboxService protobuf definition (Phase 14-01).
 */

// ---------------------------------------------------------------------------
// Channel types
// ---------------------------------------------------------------------------

export type InboxChannel = 'email' | 'chat' | 'notification'

// ---------------------------------------------------------------------------
// Core message types
// ---------------------------------------------------------------------------

export interface InboxMessage {
  id: string
  user_id: string
  channel: InboxChannel
  source_id: string
  sender_name: string
  sender_id?: string
  sender_email?: string
  subject: string
  preview: string
  is_read: boolean
  is_starred: boolean
  is_archived: boolean
  snoozed_until?: string // ISO timestamp
  assigned_to?: string
  team_inbox_id?: string
  tags: string[]
  deep_link: string
  crm_contact_id?: string
  metadata?: Record<string, unknown>
  received_at: string
  created_at: string
  updated_at: string
}

export interface InboxMessageList {
  messages: InboxMessage[]
  total_count: number
  next_page_token?: string
}

// ---------------------------------------------------------------------------
// Unread count types
// ---------------------------------------------------------------------------

export interface UnreadCount {
  channel: InboxChannel
  count: number
}

export interface UnreadCountResponse {
  counts: UnreadCount[]
  total: number
}

// ---------------------------------------------------------------------------
// Team inbox types
// ---------------------------------------------------------------------------

export type AssignmentMode = 'manual' | 'round_robin'
export type TeamVisibility = 'open' | 'private'
export type TeamMemberRole = 'admin' | 'member'

export interface TeamInbox {
  id: string
  name: string
  description?: string
  assignment_mode: AssignmentMode
  visibility: TeamVisibility
  created_by: string
  created_at: string
  updated_at: string
}

export interface TeamInboxMember {
  team_inbox_id: string
  user_id: string
  role: TeamMemberRole
  created_at: string
}

// ---------------------------------------------------------------------------
// Routing rule types
// ---------------------------------------------------------------------------

export interface Condition {
  and?: Condition[]
  or?: Condition[]
  field?: string
  operator?: string
  value?: string
}

export type RuleActionType = 'route_to_team' | 'assign_to' | 'add_tags' | 'auto_reply'

export interface RuleAction {
  type: RuleActionType
  config: Record<string, unknown>
}

export interface RoutingRule {
  id: string
  name: string
  channel?: InboxChannel
  conditions: Condition
  actions: RuleAction[]
  priority: number
  is_active: boolean
  created_by: string
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// Filter / query params
// ---------------------------------------------------------------------------

export interface InboxListFilter {
  channel?: InboxChannel
  is_read?: boolean
  is_starred?: boolean
  is_archived?: boolean
  team_inbox_id?: string
  search?: string
  page_size?: number
  page_token?: string
}
