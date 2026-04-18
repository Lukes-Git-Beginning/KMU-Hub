/**
 * Lightweight fetch wrapper for Helpdesk API endpoints.
 *
 * Follows the same pattern as dialer-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Once helpdesk routes are added
 * to openapi.yaml, hooks can migrate to the typed apiClient.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  Ticket,
  TicketMessage,
  TicketQueue,
  CannedResponse,
  SLAPolicy,
  TicketSLAStatus,
  CreateTicketInput,
  UpdateTicketInput,
  AddMessageInput,
  CreateQueueInput,
  UpdateQueueInput,
  CreateCannedResponseInput,
  UpdateCannedResponseInput,
  CreateSLAPolicyInput,
  UpdateSLAPolicyInput,
  ListTicketsResponse,
  TicketStatus,
} from './helpdesk-types'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

const MUTATION_METHODS = new Set(['POST', 'PUT', 'DELETE', 'PATCH'])

async function getAuthToken(): Promise<string | null> {
  const { useAuthStore } = await import('@/stores/auth')
  return useAuthStore.getState().accessToken
}

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new Error('Änderungen sind offline nicht möglich.')
  }

  const url = new URL(`${API_BASE_URL}${opts.path}`)

  if (opts.params) {
    for (const [key, value] of Object.entries(opts.params)) {
      if (value === undefined) continue
      url.searchParams.set(key, String(value))
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  const token = await getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const init: RequestInit = { method: opts.method, headers }

  if (opts.body !== undefined) {
    init.body = JSON.stringify(opts.body)
  }

  const response = await fetch(url.toString(), init)

  if (!response.ok) {
    if (response.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryResponse = await fetch(url.toString(), { ...init, headers })

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

// ---------------------------------------------------------------------------
// Base path
// ---------------------------------------------------------------------------

const BASE = '/api/v1/helpdesk'

// ---------------------------------------------------------------------------
// Tickets
// ---------------------------------------------------------------------------

export function listTickets(params?: {
  status?: TicketStatus
  page?: number
  page_size?: number
}) {
  return request<ListTicketsResponse>({
    method: 'GET',
    path: `${BASE}/tickets`,
    params: params as Record<string, string | number | undefined>,
  })
}

export function getTicket(id: string) {
  return request<Ticket>({ method: 'GET', path: `${BASE}/tickets/${id}` })
}

export function createTicket(body: CreateTicketInput) {
  return request<Ticket>({ method: 'POST', path: `${BASE}/tickets`, body })
}

export function updateTicket(id: string, body: UpdateTicketInput) {
  return request<Ticket>({ method: 'PUT', path: `${BASE}/tickets/${id}`, body })
}

export function closeTicket(id: string) {
  return request<Ticket>({ method: 'POST', path: `${BASE}/tickets/${id}/close` })
}

export function reopenTicket(id: string) {
  return request<Ticket>({ method: 'POST', path: `${BASE}/tickets/${id}/reopen` })
}

export function assignTicket(id: string, assigneeId: string) {
  return request<Ticket>({
    method: 'POST',
    path: `${BASE}/tickets/${id}/assign`,
    body: { assignee_id: assigneeId },
  })
}

export function mergeTickets(sourceId: string, targetId: string) {
  return request<void>({
    method: 'POST',
    path: `${BASE}/tickets/merge`,
    body: { source_id: sourceId, target_id: targetId },
  })
}

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

export function listMessages(ticketId: string) {
  return request<TicketMessage[]>({
    method: 'GET',
    path: `${BASE}/tickets/${ticketId}/messages`,
  })
}

export function addMessage(ticketId: string, body: AddMessageInput) {
  return request<TicketMessage>({
    method: 'POST',
    path: `${BASE}/tickets/${ticketId}/messages`,
    body,
  })
}

// ---------------------------------------------------------------------------
// Queues
// ---------------------------------------------------------------------------

export function listQueues() {
  return request<TicketQueue[]>({ method: 'GET', path: `${BASE}/queues` })
}

export function createQueue(body: CreateQueueInput) {
  return request<TicketQueue>({ method: 'POST', path: `${BASE}/queues`, body })
}

export function updateQueue(id: string, body: UpdateQueueInput) {
  return request<TicketQueue>({ method: 'PUT', path: `${BASE}/queues/${id}`, body })
}

export function deleteQueue(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/queues/${id}` })
}

// ---------------------------------------------------------------------------
// Canned responses
// ---------------------------------------------------------------------------

export function listCannedResponses() {
  return request<CannedResponse[]>({ method: 'GET', path: `${BASE}/canned-responses` })
}

export function createCannedResponse(body: CreateCannedResponseInput) {
  return request<CannedResponse>({ method: 'POST', path: `${BASE}/canned-responses`, body })
}

export function updateCannedResponse(id: string, body: UpdateCannedResponseInput) {
  return request<CannedResponse>({
    method: 'PUT',
    path: `${BASE}/canned-responses/${id}`,
    body,
  })
}

export function deleteCannedResponse(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/canned-responses/${id}` })
}

// ---------------------------------------------------------------------------
// SLA policies
// ---------------------------------------------------------------------------

export function listSLAPolicies() {
  return request<SLAPolicy[]>({ method: 'GET', path: `${BASE}/sla-policies` })
}

export function createSLAPolicy(body: CreateSLAPolicyInput) {
  return request<SLAPolicy>({ method: 'POST', path: `${BASE}/sla-policies`, body })
}

export function updateSLAPolicy(id: string, body: UpdateSLAPolicyInput) {
  return request<SLAPolicy>({ method: 'PUT', path: `${BASE}/sla-policies/${id}`, body })
}

export function deleteSLAPolicy(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/sla-policies/${id}` })
}

export function applySLAPolicy(ticketId: string, policyId: string) {
  return request<Ticket>({
    method: 'POST',
    path: `${BASE}/tickets/${ticketId}/sla`,
    body: { policy_id: policyId },
  })
}

export function getSLAStatus(ticketId: string) {
  return request<TicketSLAStatus>({
    method: 'GET',
    path: `${BASE}/tickets/${ticketId}/sla`,
  })
}
