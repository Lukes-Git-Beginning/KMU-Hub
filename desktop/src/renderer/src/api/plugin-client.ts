/**
 * Plugin system API client.
 *
 * Follows the bexio-client.ts pattern: typed fetch helper with auth
 * header injection and 401 retry. All endpoints target the plugin gateway
 * routes under /api/v1/plugins/*.
 */
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
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function request<T>(opts: { method: string; path: string; body?: unknown }): Promise<T> {
  return authenticatedRequest<T>(opts)
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
