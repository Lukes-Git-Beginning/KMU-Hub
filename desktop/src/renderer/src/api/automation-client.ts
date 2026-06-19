/**
 * Fetch wrapper for Automation API endpoints.
 *
 * Follows the inbox-client.ts / calendar-client.ts pattern:
 * typed fetch helper with auth header injection and 401 retry.
 * Base path: /api/v1/automations
 */
import type {
  Automation,
  AutomationListResponse,
  AutomationListParams,
  AutomationExecution,
  ExecutionListResponse,
  ExecutionListParams,
  TriggerDefinitionListResponse,
  ActionDefinitionListResponse,
  TemplateListResponse,
  TestConditionResponse,
  DryRunResponse,
  AutomationStats,
  ConditionConfig,
} from './automation-types'
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal fetch helper
// ---------------------------------------------------------------------------

function request<T>(opts: { method: string; path: string; body?: unknown; params?: Record<string, string | number | boolean | string[] | undefined> }): Promise<T> {
  return authenticatedRequest<T>(opts)
}

function autoGet<T>(
  path: string,
  params?: Record<string, string | number | boolean | string[] | undefined>,
): Promise<T> {
  return request<T>({ method: 'GET', path: `/api/v1/automations${path}`, params })
}

function autoPost<T>(path: string, body?: unknown): Promise<T> {
  return request<T>({ method: 'POST', path: `/api/v1/automations${path}`, body })
}

function autoPut<T>(path: string, body: unknown): Promise<T> {
  return request<T>({ method: 'PUT', path: `/api/v1/automations${path}`, body })
}

function autoDelete(path: string): Promise<void> {
  return request<void>({ method: 'DELETE', path: `/api/v1/automations${path}` })
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

export function createAutomation(data: {
  name: string
  description?: string
  scope?: string
  trigger_type: string
  trigger_config?: Record<string, unknown>
  conditions?: ConditionConfig
  actions?: unknown[]
  max_steps?: number
}): Promise<Automation> {
  return autoPost<Automation>('', data)
}

export function updateAutomation(
  id: string,
  data: Partial<{
    name: string
    description: string
    scope: string
    trigger_type: string
    trigger_config: Record<string, unknown>
    conditions: ConditionConfig
    actions: unknown[]
    max_steps: number
  }>,
): Promise<Automation> {
  return autoPut<Automation>(`/${id}`, data)
}

export function deleteAutomation(id: string): Promise<void> {
  return autoDelete(`/${id}`)
}

export function getAutomation(id: string): Promise<Automation> {
  return autoGet<Automation>(`/${id}`)
}

export function listAutomations(
  params?: AutomationListParams,
): Promise<AutomationListResponse> {
  const queryParams: Record<string, string | number | boolean | undefined> = {}
  if (params?.owner_id) queryParams.owner_id = params.owner_id
  if (params?.scope) queryParams.scope = params.scope
  if (params?.trigger_type) queryParams.trigger_type = params.trigger_type
  if (params?.is_active !== undefined) queryParams.is_active = params.is_active
  if (params?.limit) queryParams.limit = params.limit
  if (params?.offset) queryParams.offset = params.offset
  return autoGet<AutomationListResponse>('', queryParams)
}

// ---------------------------------------------------------------------------
// Enable/Disable
// ---------------------------------------------------------------------------

export function enableAutomation(id: string): Promise<Automation> {
  return autoPost<Automation>(`/${id}/enable`)
}

export function disableAutomation(id: string): Promise<Automation> {
  return autoPost<Automation>(`/${id}/disable`)
}

// ---------------------------------------------------------------------------
// Execution logs
// ---------------------------------------------------------------------------

export function listExecutions(
  automationId: string,
  params?: ExecutionListParams,
): Promise<ExecutionListResponse> {
  const queryParams: Record<string, string | number | boolean | undefined> = {}
  if (params?.status) queryParams.status = params.status
  if (params?.limit) queryParams.limit = params.limit
  if (params?.offset) queryParams.offset = params.offset
  return autoGet<ExecutionListResponse>(`/${automationId}/executions`, queryParams)
}

export function getExecution(executionId: string): Promise<AutomationExecution> {
  return autoGet<AutomationExecution>(`/executions/${executionId}`)
}

/** List executions across all automations (page-level log tab). */
export function listAllExecutions(
  params?: ExecutionListParams,
): Promise<ExecutionListResponse> {
  const queryParams: Record<string, string | number | boolean | undefined> = {}
  if (params?.status) queryParams.status = params.status
  if (params?.limit) queryParams.limit = params.limit
  if (params?.offset) queryParams.offset = params.offset
  return autoGet<ExecutionListResponse>('/executions', queryParams)
}

// ---------------------------------------------------------------------------
// Catalog
// ---------------------------------------------------------------------------

export function listTriggerDefinitions(): Promise<TriggerDefinitionListResponse> {
  return autoGet<TriggerDefinitionListResponse>('/triggers')
}

export function listActionDefinitions(): Promise<ActionDefinitionListResponse> {
  return autoGet<ActionDefinitionListResponse>('/actions')
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

export function listTemplates(
  category?: string,
): Promise<TemplateListResponse> {
  const params: Record<string, string | undefined> = {}
  if (category) params.category = category
  return autoGet<TemplateListResponse>('/templates', params)
}

export function createFromTemplate(
  templateId: string,
  data: { name: string },
): Promise<Automation> {
  return autoPost<Automation>(`/templates/${templateId}/create`, data)
}

// ---------------------------------------------------------------------------
// Testing
// ---------------------------------------------------------------------------

export function testCondition(data: {
  condition: ConditionConfig
  sample_env: Record<string, unknown>
}): Promise<TestConditionResponse> {
  return autoPost<TestConditionResponse>('/test-condition', data)
}

export function dryRunAutomation(data: {
  automation_id: string
  sample_env: Record<string, unknown>
  /** Draft actions for simulating an unsaved automation. */
  actions?: { type: string }[]
}): Promise<DryRunResponse> {
  return autoPost<DryRunResponse>('/dry-run', data)
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

export function getAutomationStats(): Promise<AutomationStats> {
  return autoGet<AutomationStats>('/stats')
}
