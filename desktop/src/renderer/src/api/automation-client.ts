/**
 * Fetch wrapper for Automation API endpoints.
 *
 * Follows the inbox-client.ts / calendar-client.ts pattern:
 * typed fetch helper with auth header injection and 401 retry.
 * Base path: /api/v1/automations
 */
import { API_BASE_URL } from '@/lib/constants'
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
  AutomationTemplate,
  TestConditionResponse,
  DryRunResponse,
  AutomationStats,
  ConditionConfig,
} from './automation-types'

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
}): Promise<DryRunResponse> {
  return autoPost<DryRunResponse>('/dry-run', data)
}

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

export function getAutomationStats(): Promise<AutomationStats> {
  return autoGet<AutomationStats>('/stats')
}
