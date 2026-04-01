/**
 * Plugin system API client.
 *
 * Follows the bexio-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the plugin gateway
 * routes under /api/v1/plugins/*.
 */
import { API_BASE_URL } from '@/lib/constants'
import type {
  PluginManifest,
  PluginInstallation,
  ValidationRule,
  WorkflowRule,
  IndustryTemplate,
  ExecutionLog,
  CreateManifestRequest,
  InstallPluginRequest,
  ApprovePermissionsRequest,
  UpdateSettingsRequest,
  CreateValidationRuleRequest,
  UpdateValidationRuleRequest,
  CreateWorkflowRuleRequest,
  UpdateWorkflowRuleRequest,
  ApplyTemplateRequest,
} from './plugin-types'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

class OfflineError extends Error {
  constructor() {
    super('Änderungen sind offline nicht möglich.')
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
}

async function request<T>(opts: RequestOptions): Promise<T> {
  if (!navigator.onLine && MUTATION_METHODS.has(opts.method)) {
    throw new OfflineError()
  }

  const url = `${API_BASE_URL}${opts.path}`
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

  const res = await fetch(url, init)

  if (!res.ok) {
    if (res.status === 401) {
      const { useAuthStore } = await import('@/stores/auth')
      const store = useAuthStore.getState()
      const newToken = await store.refreshToken()

      if (newToken) {
        headers['Authorization'] = `Bearer ${newToken}`
        const retryRes = await fetch(url, { ...init, headers })
        if (!retryRes.ok) {
          const err = await retryRes.json().catch(() => ({}))
          throw new Error(
            (err as Record<string, string>).error ||
              `Request failed: ${retryRes.status}`,
          )
        }
        if (retryRes.status === 204) return {} as T
        return retryRes.json() as Promise<T>
      }

      store.logout()
      throw new Error('Authentication expired')
    }

    const errBody = await res.json().catch(() => ({}))
    throw new Error(
      (errBody as Record<string, string>).error ||
        `Request failed: ${res.status}`,
    )
  }

  if (res.status === 204) return {} as T
  return res.json() as Promise<T>
}

// ---------------------------------------------------------------------------
// Manifests
// ---------------------------------------------------------------------------

export function listManifests() {
  return request<{ manifests: PluginManifest[] }>({
    method: 'GET',
    path: '/api/v1/plugins/manifests',
  })
}

export function getManifest(id: string) {
  return request<PluginManifest>({
    method: 'GET',
    path: `/api/v1/plugins/manifests/${id}`,
  })
}

export function createManifest(data: CreateManifestRequest) {
  return request<PluginManifest>({
    method: 'POST',
    path: '/api/v1/plugins/manifests',
    body: data,
  })
}

export function deleteManifest(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `/api/v1/plugins/manifests/${id}`,
  })
}

// ---------------------------------------------------------------------------
// Installations
// ---------------------------------------------------------------------------

export function listInstallations() {
  return request<{ installations: PluginInstallation[] }>({
    method: 'GET',
    path: '/api/v1/plugins/installations',
  })
}

export function installPlugin(data: InstallPluginRequest) {
  return request<PluginInstallation>({
    method: 'POST',
    path: '/api/v1/plugins/installations',
    body: data,
  })
}

export function enablePlugin(installationId: string) {
  return request<void>({
    method: 'POST',
    path: `/api/v1/plugins/installations/${installationId}/enable`,
  })
}

export function disablePlugin(installationId: string) {
  return request<void>({
    method: 'POST',
    path: `/api/v1/plugins/installations/${installationId}/disable`,
  })
}

export function uninstallPlugin(installationId: string) {
  return request<void>({
    method: 'DELETE',
    path: `/api/v1/plugins/installations/${installationId}`,
  })
}

// ---------------------------------------------------------------------------
// Permissions
// ---------------------------------------------------------------------------

export function approvePermissions(
  installationId: string,
  data: ApprovePermissionsRequest,
) {
  return request<void>({
    method: 'POST',
    path: `/api/v1/plugins/installations/${installationId}/permissions/approve`,
    body: data,
  })
}

export function listPermissions(installationId: string) {
  return request<{ required: string[]; granted: string[] }>({
    method: 'GET',
    path: `/api/v1/plugins/installations/${installationId}/permissions`,
  })
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

export function getSettings(installationId: string) {
  return request<{ settings: Record<string, unknown>; schema: Record<string, unknown> }>({
    method: 'GET',
    path: `/api/v1/plugins/installations/${installationId}/settings`,
  })
}

export function updateSettings(
  installationId: string,
  data: UpdateSettingsRequest,
) {
  return request<void>({
    method: 'PUT',
    path: `/api/v1/plugins/installations/${installationId}/settings`,
    body: data,
  })
}

export function getSettingsSchema(installationId: string) {
  return request<{ schema: Record<string, unknown> }>({
    method: 'GET',
    path: `/api/v1/plugins/installations/${installationId}/settings/schema`,
  })
}

// ---------------------------------------------------------------------------
// Validation Rules
// ---------------------------------------------------------------------------

export function listValidationRules(installationId?: string) {
  const params = installationId ? `?installation_id=${installationId}` : ''
  return request<{ rules: ValidationRule[] }>({
    method: 'GET',
    path: `/api/v1/plugins/validation-rules${params}`,
  })
}

export function createValidationRule(data: CreateValidationRuleRequest) {
  return request<ValidationRule>({
    method: 'POST',
    path: '/api/v1/plugins/validation-rules',
    body: data,
  })
}

export function updateValidationRule(
  id: string,
  data: UpdateValidationRuleRequest,
) {
  return request<ValidationRule>({
    method: 'PUT',
    path: `/api/v1/plugins/validation-rules/${id}`,
    body: data,
  })
}

export function deleteValidationRule(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `/api/v1/plugins/validation-rules/${id}`,
  })
}

// ---------------------------------------------------------------------------
// Workflow Rules
// ---------------------------------------------------------------------------

export function listWorkflowRules(installationId?: string) {
  const params = installationId ? `?installation_id=${installationId}` : ''
  return request<{ rules: WorkflowRule[] }>({
    method: 'GET',
    path: `/api/v1/plugins/workflow-rules${params}`,
  })
}

export function createWorkflowRule(data: CreateWorkflowRuleRequest) {
  return request<WorkflowRule>({
    method: 'POST',
    path: '/api/v1/plugins/workflow-rules',
    body: data,
  })
}

export function updateWorkflowRule(
  id: string,
  data: UpdateWorkflowRuleRequest,
) {
  return request<WorkflowRule>({
    method: 'PUT',
    path: `/api/v1/plugins/workflow-rules/${id}`,
    body: data,
  })
}

export function deleteWorkflowRule(id: string) {
  return request<void>({
    method: 'DELETE',
    path: `/api/v1/plugins/workflow-rules/${id}`,
  })
}

// ---------------------------------------------------------------------------
// Industry Templates
// ---------------------------------------------------------------------------

export function listTemplates() {
  return request<{ templates: IndustryTemplate[] }>({
    method: 'GET',
    path: '/api/v1/plugins/templates',
  })
}

export function applyTemplate(data: ApplyTemplateRequest) {
  return request<{ installation_id: string }>({
    method: 'POST',
    path: '/api/v1/plugins/templates/apply',
    body: data,
  })
}

// ---------------------------------------------------------------------------
// Execution Logs
// ---------------------------------------------------------------------------

export function listExecutionLogs(installationId?: string, limit = 50) {
  const params = new URLSearchParams()
  if (installationId) params.set('installation_id', installationId)
  params.set('limit', String(limit))
  const qs = params.toString()
  return request<{ logs: ExecutionLog[] }>({
    method: 'GET',
    path: `/api/v1/plugins/execution-logs?${qs}`,
  })
}
