/**
 * Lightweight fetch wrapper for Dialer API endpoints.
 *
 * Follows the same pattern as video-client.ts: manual typed fetch helper
 * with auth header injection and 401 retry. Once dialer routes are added
 * to openapi.yaml, hooks can migrate to the typed apiClient.
 */
import { authenticatedRequest } from './utils/authenticatedFetch'
import {
  normalizeSupervisorOverview,
  normalizeContactCalls,
} from './dialer-normalize'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Campaign {
  id: string
  tenant_id: string
  name: string
  description: string | null
  status: number
  mode: number
  settings: string | null
  created_by: string
  assigned_agent_ids: string[]
  contact_count: number
  completed_count: number
  created_at: string
  updated_at: string
  started_at: string | null
  completed_at: string | null
}

export interface CampaignContact {
  id: string
  campaign_id: string
  contact_id: string
  position: number
  status: number
  outcome_id: string | null
  notes: string | null
  callback_at: string | null
  last_called_at: string | null
  call_count: number
  contact_name: string
  contact_phone: string
  contact_company: string | null
}

export interface DialerCallSession {
  id: string
  campaign_contact_id: string
  call_session_id: string | null
  agent_id: string
  outcome_id: string | null
  duration_seconds: number | null
  notes: string | null
  next_action: string | null
  appointment_id: string | null
  created_at: string
  wrap_up_completed_at: string | null
}

export interface CallOutcome {
  id: string
  tenant_id: string
  label: string
  color: string
  is_positive: boolean
  is_callback: boolean
  is_appointment: boolean
  sort_order: number
  is_active: boolean
}

export interface AgentStatusEntry {
  user_id: string
  status: number
  campaign_id: string | null
  since: string
  first_name: string
  last_name: string
}

export interface CampaignDashboard {
  campaign_id: string
  total_contacts: number
  completed_contacts: number
  pending_contacts: number
  skipped_contacts: number
  callback_contacts: number
  total_calls: number
  positive_calls: number
  average_duration_seconds: number
  calls_per_hour: number
  outcome_breakdown: OutcomeBreakdownEntry[]
}

export interface AgentDashboard {
  agent_id: string
  total_calls_today: number
  average_duration_seconds: number
  calls_per_hour: number
  outcome_breakdown: OutcomeBreakdownEntry[]
  active_campaign_id: string | null
  active_campaign_name: string | null
}

export interface OutcomeBreakdownEntry {
  outcome_id: string
  outcome_label: string
  outcome_color: string
  count: number
}

export interface SupervisorAgent {
  user_id: string
  first_name: string
  last_name: string
  status: number
  campaign_id: string | null
  since: string
  calls_today: number
  avg_duration_seconds: number
  active_campaign_name: string | null
}

export interface RecentCallEntry {
  id: string
  contact_name: string
  contact_company: string | null
  outcome_label: string | null
  outcome_color: string
  is_positive: boolean
  duration_seconds: number
  agent_name: string
  campaign_name: string | null
  created_at: string
}

export interface ContactCallEntry {
  id: string
  outcome_label: string | null
  outcome_color: string
  is_positive: boolean
  duration_seconds: number
  notes: string | null
  agent_name: string
  created_at: string
}

export interface SupervisorOverview {
  agents: SupervisorAgent[]
  recent_calls: RecentCallEntry[]
  totals: {
    active_agents: number
    on_call: number
    calls_today: number
    appointments_today: number
  }
}

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
// Campaigns
// ---------------------------------------------------------------------------

const BASE = '/api/v1/dialer'

export function listCampaigns(params?: {
  page?: number
  page_size?: number
  status_filter?: number
}) {
  return request<{ campaigns: Campaign[]; total: number }>({
    method: 'GET',
    path: `${BASE}/campaigns`,
    params: params as Record<string, string | number | undefined>,
  })
}

export function getCampaign(id: string) {
  return request<Campaign>({ method: 'GET', path: `${BASE}/campaigns/${id}` })
}

export function createCampaign(body: {
  name: string
  description?: string
  mode?: number
  assigned_agent_ids?: string[]
  settings?: string
}) {
  return request<Campaign>({ method: 'POST', path: `${BASE}/campaigns`, body })
}

export function updateCampaign(
  id: string,
  body: {
    name?: string
    description?: string
    assigned_agent_ids?: string[]
    settings?: string
  },
) {
  return request<Campaign>({
    method: 'PUT',
    path: `${BASE}/campaigns/${id}`,
    body,
  })
}

export function startCampaign(id: string) {
  return request<Campaign>({
    method: 'POST',
    path: `${BASE}/campaigns/${id}/start`,
  })
}

export function pauseCampaign(id: string) {
  return request<Campaign>({
    method: 'POST',
    path: `${BASE}/campaigns/${id}/pause`,
  })
}

export function archiveCampaign(id: string) {
  return request<{ status: string }>({
    method: 'DELETE',
    path: `${BASE}/campaigns/${id}`,
  })
}

// ---------------------------------------------------------------------------
// Campaign Contacts
// ---------------------------------------------------------------------------

export function addContactsToCampaign(
  campaignId: string,
  body: { contact_ids?: string[]; filter_id?: string },
) {
  return request<{ added_count: number; skipped_count: number }>({
    method: 'POST',
    path: `${BASE}/campaigns/${campaignId}/contacts`,
    body,
  })
}

export function listCampaignContacts(
  campaignId: string,
  params?: { page?: number; page_size?: number; status_filter?: number },
) {
  return request<{ contacts: CampaignContact[]; total: number }>({
    method: 'GET',
    path: `${BASE}/campaigns/${campaignId}/contacts`,
    params: params as Record<string, string | number | undefined>,
  })
}

export function getNextContact(campaignId: string) {
  return request<CampaignContact>({
    method: 'GET',
    path: `${BASE}/campaigns/${campaignId}/next`,
  })
}

export function skipContact(campaignId: string, contactId: string) {
  return request<{ status: string }>({
    method: 'POST',
    path: `${BASE}/campaigns/${campaignId}/contacts/${contactId}/skip`,
  })
}

export function requeueContact(campaignId: string, contactId: string) {
  return request<{ status: string }>({
    method: 'POST',
    path: `${BASE}/campaigns/${campaignId}/contacts/${contactId}/requeue`,
  })
}

// ---------------------------------------------------------------------------
// Calls
// ---------------------------------------------------------------------------

export function initiateCall(body: {
  campaign_contact_id: string
  agent_id?: string
  call_session_id?: string
}) {
  return request<DialerCallSession>({
    method: 'POST',
    path: `${BASE}/calls`,
    body,
  })
}

export function logCallOutcome(
  callId: string,
  body: {
    outcome_id: string
    notes?: string
    next_action?: string
    callback_at?: string
    appointment_id?: string
  },
) {
  return request<DialerCallSession>({
    method: 'POST',
    path: `${BASE}/calls/${callId}/outcome`,
    body,
  })
}

export function saveCallNotes(callId: string, notes: string) {
  return request<{ status: string }>({
    method: 'PUT',
    path: `${BASE}/calls/${callId}/notes`,
    body: { notes },
  })
}

export function completeWrapUp(callId: string, agentId?: string) {
  return request<{ status: string }>({
    method: 'POST',
    path: `${BASE}/calls/${callId}/complete`,
    body: { agent_id: agentId },
  })
}

// ---------------------------------------------------------------------------
// Agent Status
// ---------------------------------------------------------------------------

export function getAgentStatus(userId?: string) {
  return request<AgentStatusEntry>({
    method: 'GET',
    path: `${BASE}/agent-status`,
    params: userId ? { user_id: userId } : undefined,
  })
}

export function setAgentStatus(body: {
  user_id?: string
  status: number
  campaign_id?: string
}) {
  return request<AgentStatusEntry>({
    method: 'PUT',
    path: `${BASE}/agent-status`,
    body,
  })
}

export function getCampaignAgents(campaignId: string) {
  return request<{ agents: AgentStatusEntry[] }>({
    method: 'GET',
    path: `${BASE}/agent-status/campaign/${campaignId}`,
  })
}

export function getSupervisorOverview() {
  // Real backend uses protojson with EmitUnpopulated=false → zero-value fields
  // (totals counters, calls_today, empty agent/call lists) are omitted. Normalize
  // fills them so a quiet dialer renders 0 instead of undefined/crash.
  return request<Partial<SupervisorOverview>>({
    method: 'GET',
    path: `${BASE}/supervisor`,
  }).then(normalizeSupervisorOverview)
}

export function getContactCalls(campaignContactId: string) {
  return request<{ calls?: Partial<ContactCallEntry>[] }>({
    method: 'GET',
    path: `${BASE}/contacts/${campaignContactId}/calls`,
  }).then(normalizeContactCalls)
}

export function getAgentDashboard(agentId?: string) {
  return request<AgentDashboard>({
    method: 'GET',
    path: `${BASE}/dashboard/agent`,
    params: agentId ? { agent_id: agentId } : undefined,
  })
}

// ---------------------------------------------------------------------------
// Campaign Dashboard
// ---------------------------------------------------------------------------

export function getCampaignDashboard(campaignId: string) {
  return request<CampaignDashboard>({
    method: 'GET',
    path: `${BASE}/campaigns/${campaignId}/dashboard`,
  })
}

// ---------------------------------------------------------------------------
// Outcomes
// ---------------------------------------------------------------------------

export function listOutcomes(includeInactive?: boolean) {
  return request<{ outcomes: CallOutcome[] }>({
    method: 'GET',
    path: `${BASE}/outcomes`,
    params: includeInactive ? { include_inactive: 'true' } : undefined,
  })
}

export function createOutcome(body: {
  label: string
  color: string
  is_positive: boolean
  is_callback: boolean
  is_appointment: boolean
  sort_order: number
}) {
  return request<CallOutcome>({
    method: 'POST',
    path: `${BASE}/outcomes`,
    body,
  })
}

export function updateOutcome(
  id: string,
  body: Partial<{
    label: string
    color: string
    is_positive: boolean
    is_callback: boolean
    is_appointment: boolean
    sort_order: number
    is_active: boolean
  }>,
) {
  return request<CallOutcome>({
    method: 'PUT',
    path: `${BASE}/outcomes/${id}`,
    body,
  })
}

export function deleteOutcome(id: string) {
  return request<{ status: string }>({
    method: 'DELETE',
    path: `${BASE}/outcomes/${id}`,
  })
}
