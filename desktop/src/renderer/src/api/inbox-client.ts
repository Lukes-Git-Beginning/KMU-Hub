/**
 * Fetch wrapper for Unified Inbox API endpoints.
 *
 * Follows the calendar-client.ts / video-client.ts pattern:
 * typed fetch helper with auth header injection and 401 retry.
 * Base path: /api/v1/inbox
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  InboxListFilter,
  InboxMessageList,
  InboxMessage,
  UnreadCountResponse,
  TeamInbox,
  TeamInboxMember,
  TeamMemberRole,
  RoutingRule,
  Condition,
} from './inbox-types'

// ---------------------------------------------------------------------------
// Internal fetch helpers
// ---------------------------------------------------------------------------

class OfflineError extends Error {
  constructor() {
    super(
      'Aenderungen sind offline nicht moeglich. Bitte stellen Sie die Internetverbindung wieder her.',
    )
    this.name = 'OfflineError'
  }
}

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | string[] | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new OfflineError()
  }

  const url = new URL(`${API_BASE_URL}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined) continue
      if (Array.isArray(value)) {
        for (const v of value) {
          url.searchParams.append(key, v)
        }
      } else {
        url.searchParams.set(key, String(value))
      }
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = {
    method: opts.method,
    headers,
  }

  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const response = await fetch(url.toString(), init)

  if (!response.ok) {
    // Attempt 401 token refresh once
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryResponse = await fetch(url.toString(), {
          ...init,
          headers,
        })

        if (!retryResponse.ok) {
          const errBody = await retryResponse.json().catch(() => ({}))
          throw new Error(
            (errBody as Record<string, string>).error ||
              `Request failed: ${retryResponse.status}`,
          )
        }

        if (retryResponse.status === 204) return {} as T
        return retryResponse.json() as Promise<T>
      }

      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await response.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${response.status}`,
    )
  }

  if (response.status === 204) return {} as T
  return response.json() as Promise<T>
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
  return inboxGet<InboxMessageList>('/messages', params)
}

export function getMessage(id: string): Promise<InboxMessage> {
  return inboxGet<InboxMessage>(`/messages/${id}`)
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
