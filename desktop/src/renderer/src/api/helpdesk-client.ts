/**
 * Lightweight fetch wrapper for Helpdesk API endpoints.
 *
 * Follows the same pattern as dialer-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Once helpdesk routes are added
 * to openapi.yaml, hooks can migrate to the typed apiClient.
 */
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
  KBArticle,
  CreateKBArticleInput,
  UpdateKBArticleInput,
  RoutingRule,
  CreateRoutingRuleInput,
  UpdateRoutingRuleInput,
  HelpdeskStats,
} from './helpdesk-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Request helper
// ---------------------------------------------------------------------------

interface RequestOptions {
  method: string
  path: string
  body?: unknown
  params?: Record<string, string | number | boolean | undefined>
}

function request<T>(opts: RequestOptions): Promise<T> {
  return authenticatedRequest<T>(opts)
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
    path: `${BASE}/tickets/${sourceId}/merge`,
    body: { target_ticket_id: targetId },
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
    body: { sla_policy_id: policyId },
  })
}

export function getSLAStatus(ticketId: string) {
  return request<TicketSLAStatus>({
    method: 'GET',
    path: `${BASE}/tickets/${ticketId}/sla-status`,
  })
}

// ---------------------------------------------------------------------------
// KB Articles
// ---------------------------------------------------------------------------

export function listKBArticles() {
  return request<{ articles: KBArticle[] }>({ method: 'GET', path: `${BASE}/kb-articles` })
}

export function createKBArticle(body: CreateKBArticleInput) {
  return request<KBArticle>({ method: 'POST', path: `${BASE}/kb-articles`, body })
}

export function updateKBArticle(id: string, body: UpdateKBArticleInput) {
  return request<KBArticle>({ method: 'PUT', path: `${BASE}/kb-articles/${id}`, body })
}

export function deleteKBArticle(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/kb-articles/${id}` })
}

// ---------------------------------------------------------------------------
// Routing Rules
// ---------------------------------------------------------------------------

export function listRoutingRules() {
  return request<{ rules: RoutingRule[] }>({ method: 'GET', path: `${BASE}/routing-rules` })
}

export function createRoutingRule(body: CreateRoutingRuleInput) {
  return request<RoutingRule>({ method: 'POST', path: `${BASE}/routing-rules`, body })
}

export function updateRoutingRule(id: string, body: UpdateRoutingRuleInput) {
  return request<RoutingRule>({ method: 'PUT', path: `${BASE}/routing-rules/${id}`, body })
}

export function deleteRoutingRule(id: string) {
  return request<void>({ method: 'DELETE', path: `${BASE}/routing-rules/${id}` })
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

export function getHelpdeskStats() {
  return request<HelpdeskStats>({ method: 'GET', path: `${BASE}/stats` })
}
