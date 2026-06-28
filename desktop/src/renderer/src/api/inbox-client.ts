/**
 * Fetch wrapper for Unified Inbox API endpoints.
 *
 * Follows the calendar-client.ts / video-client.ts pattern:
 * typed fetch helper with auth header injection and 401 retry.
 * Base path: /api/v1/inbox
 */
import type {
  InboxListFilter,
  InboxMessageList,
  InboxMessage,
  InboxChannel,
  UnreadCountResponse,
  TeamInbox,
  TeamInboxMember,
  TeamMemberRole,
  RoutingRule,
  Condition,
} from './inbox-types'
import { authenticatedRequest } from './utils/authenticatedFetch'
import { normalizeWireTimestamps } from '@/api/wire-time'

// ---------------------------------------------------------------------------
// Wire-shape normalization
//
// The inbox runs through response.JSON over a protobuf message, so Timestamps
// arrive as { seconds, nanos } and the channel enum as an int (1=email, 2=chat,
// 3=notification). MSW returned the already-typed shapes, so this was mock-hidden
// and surfaced as `received_at.localeCompare is not a function` (sort crash) and
// a wrong channel filter against the real backend. Normalize on the way in.
// ---------------------------------------------------------------------------

const CHANNEL_BY_INT: Record<number, InboxChannel> = { 1: 'email', 2: 'chat', 3: 'notification' }

function normalizeMessage(raw: unknown): InboxMessage {
  const m = normalizeWireTimestamps(raw) as Record<string, unknown>
  return {
    ...(m as unknown as InboxMessage),
    channel:
      typeof m.channel === 'number'
        ? (CHANNEL_BY_INT[m.channel] ?? 'notification')
        : ((m.channel as InboxChannel) ?? 'notification'),
  }
}

// ---------------------------------------------------------------------------
// Internal fetch helpers
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | string[] | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
}

function inboxGet<T>(
  path: string,
  params?: Record<string, string | number | boolean | string[] | undefined>,
): Promise<T> {
  return request<T>({ method: 'GET', path: `/api/v1/inbox${path}`, params })
}

function inboxPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'POST', path: `/api/v1/inbox${path}`, body })
}

function inboxPut<T>(path: string, body: unknown): Promise<T> {
  return request<T>({ method: 'PUT', path: `/api/v1/inbox${path}`, body })
}

function inboxDelete(path: string): Promise<void> {
  return request<void>({ method: 'DELETE', path: `/api/v1/inbox${path}` })
}

// ---------------------------------------------------------------------------
// Message API
// ---------------------------------------------------------------------------

export function listMessages(filter: InboxListFilter): Promise<InboxMessageList> {
  const params: Record<string, string | number | boolean | undefined> = {}
  if (filter.channel) params.channel = filter.channel
  if (filter.is_read !== undefined) params.is_read = filter.is_read
  if (filter.is_starred !== undefined) params.is_starred = filter.is_starred
  if (filter.is_archived !== undefined) params.is_archived = filter.is_archived
  if (filter.team_inbox_id) params.team_inbox_id = filter.team_inbox_id
  if (filter.search) params.search = filter.search
  if (filter.page_size) params.page_size = filter.page_size
  if (filter.page_token) params.page_token = filter.page_token
  return inboxGet<unknown>('/messages', params).then((raw) => {
    const o = (raw ?? {}) as Record<string, unknown>
    return {
      ...(o as unknown as InboxMessageList),
      messages: (Array.isArray(o.messages) ? o.messages : []).map(normalizeMessage),
    }
  })
}

export function getMessage(id: string): Promise<InboxMessage> {
  // GET /messages/{id} wraps the message ({ message: {...} }), unlike the list.
  return inboxGet<unknown>(`/messages/${id}`).then((raw) => {
    const o = (raw ?? {}) as Record<string, unknown>
    return normalizeMessage('message' in o ? o.message : o)
  })
}

export function markRead(id: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/read`)
}

export function markUnread(id: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/unread`)
}

export function toggleStar(id: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/star`)
}

export function archiveMessage(id: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/archive`)
}

export function unarchiveMessage(id: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/unarchive`)
}

export function snoozeMessage(id: string, snoozeUntil: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/snooze`, { snooze_until: snoozeUntil })
}

export function unsnoozeMessage(id: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/unsnooze`)
}

export function replyToMessage(id: string, body: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/reply`, { body })
}

export function assignMessage(id: string, assigneeId: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/assign`, { assignee_id: assigneeId })
}

export function getUnreadCount(): Promise<UnreadCountResponse> {
  return inboxGet<UnreadCountResponse>('/messages/unread-count')
}

export function bulkMarkRead(ids: string[]): Promise<void> {
  return inboxPost<void>('/messages/bulk/read', { ids })
}

export function bulkArchive(ids: string[]): Promise<void> {
  return inboxPost<void>('/messages/bulk/archive', { ids })
}

export function claimMessage(id: string): Promise<void> {
  return inboxPost<void>(`/messages/${id}/claim`)
}

// ---------------------------------------------------------------------------
// Team Inbox API
// ---------------------------------------------------------------------------

export function listTeamInboxes(): Promise<TeamInbox[]> {
  return inboxGet<TeamInbox[]>('/teams')
}

export function createTeamInbox(data: Partial<TeamInbox>): Promise<TeamInbox> {
  return inboxPost<TeamInbox>('/teams', data)
}

export function updateTeamInbox(id: string, data: Partial<TeamInbox>): Promise<TeamInbox> {
  return inboxPut<TeamInbox>(`/teams/${id}`, data)
}

export function deleteTeamInbox(id: string): Promise<void> {
  return inboxDelete(`/teams/${id}`)
}

export function listTeamMembers(teamId: string): Promise<TeamInboxMember[]> {
  return inboxGet<TeamInboxMember[]>(`/teams/${teamId}/members`)
}

export function addTeamMember(
  teamId: string,
  userId: string,
  role: TeamMemberRole,
): Promise<void> {
  return inboxPost<void>(`/teams/${teamId}/members`, { user_id: userId, role })
}

export function removeTeamMember(teamId: string, userId: string): Promise<void> {
  return inboxDelete(`/teams/${teamId}/members/${userId}`)
}

// ---------------------------------------------------------------------------
// Routing Rules API
// ---------------------------------------------------------------------------

export function listRoutingRules(): Promise<RoutingRule[]> {
  return inboxGet<RoutingRule[]>('/rules')
}

export function createRoutingRule(data: Partial<RoutingRule>): Promise<RoutingRule> {
  return inboxPost<RoutingRule>('/rules', data)
}

export function updateRoutingRule(
  id: string,
  data: Partial<RoutingRule>,
): Promise<RoutingRule> {
  return inboxPut<RoutingRule>(`/rules/${id}`, data)
}

export function deleteRoutingRule(id: string): Promise<void> {
  return inboxDelete(`/rules/${id}`)
}

export function testRoutingRule(
  conditions: Condition,
  message: Partial<InboxMessage>,
): Promise<{ matches: boolean }> {
  return inboxPost<{ matches: boolean }>('/rules/test', { conditions, message })
}
