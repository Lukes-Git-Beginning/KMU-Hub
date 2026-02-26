/**
 * TypeScript types for the Plugin system module.
 *
 * Covers: plugin manifests, installations, permissions, validation rules,
 * workflow rules, industry templates, and execution logs. Matches the
 * gateway endpoints defined in route_plugins.go.
 */

// ---------------------------------------------------------------------------
// Enums / union types
// ---------------------------------------------------------------------------

export type InstallationStatus =
  | 'pending_approval'
  | 'active'
  | 'disabled'
  | 'error'
  | 'uninstalled'

export type PluginType = 'config' | 'wasm'

// ---------------------------------------------------------------------------
// Plugin Manifest
// ---------------------------------------------------------------------------

export interface PluginManifest {
  id: string
  slug: string
  name: string
  description: string
  version: string
  author: string
  settings_schema: Record<string, unknown>
  permissions: string[]
  hook_registrations: HookRegistration[]
  wasm_binary_hash?: string
  plugin_type: PluginType
  created_at: string
  updated_at: string
}

export interface HookRegistration {
  hook_type: string
  module: string
  entity_type: string
  priority: number
}

// ---------------------------------------------------------------------------
// Plugin Installation
// ---------------------------------------------------------------------------

export interface PluginInstallation {
  id: string
  tenant_id: string
  manifest_id: string
  status: InstallationStatus
  settings: Record<string, unknown>
  error_message?: string
  installed_by: string
  created_at: string
  updated_at: string
  /** Denormalized fields from manifest */
  manifest_slug: string
  manifest_name: string
  manifest_version: string
  plugin_type: PluginType
  required_permissions: string[]
  granted_permissions: string[]
  settings_schema: Record<string, unknown>
}

// ---------------------------------------------------------------------------
// Validation Rule
// ---------------------------------------------------------------------------

export interface ValidationRule {
  id: string
  tenant_id: string
  installation_id: string
  name: string
  description: string
  entity_type: string
  field_name: string
  rule_type: string
  rule_config: Record<string, unknown>
  error_message: string
  priority: number
  enabled: boolean
}

// ---------------------------------------------------------------------------
// Workflow Rule
// ---------------------------------------------------------------------------

export interface WorkflowRule {
  id: string
  tenant_id: string
  installation_id: string
  name: string
  description: string
  trigger_event: string
  conditions: Record<string, unknown>
  actions: Record<string, unknown>[]
  priority: number
  enabled: boolean
}

// ---------------------------------------------------------------------------
// Industry Template
// ---------------------------------------------------------------------------

export interface IndustryTemplate {
  id: string
  slug: string
  name: string
  description: string
  industry: string
  icon: string
  custom_fields: Record<string, unknown>[]
  validation_rules: Record<string, unknown>[]
  workflow_rules: Record<string, unknown>[]
  default_settings: Record<string, unknown>
}

// ---------------------------------------------------------------------------
// Execution Log
// ---------------------------------------------------------------------------

export interface ExecutionLog {
  id: string
  installation_id: string
  hook_type: string
  module: string
  entity_type: string
  entity_id: string
  duration_ms: number
  status: 'success' | 'error' | 'skipped'
  error_message?: string
  created_at: string
}

// ---------------------------------------------------------------------------
// Request / response helpers
// ---------------------------------------------------------------------------

export interface CreateManifestRequest {
  slug: string
  name: string
  description: string
  version: string
  author: string
  settings_schema: Record<string, unknown>
  permissions: string[]
  hook_registrations: HookRegistration[]
  wasm_binary_hash?: string
  plugin_type: PluginType
}

export interface InstallPluginRequest {
  manifest_id: string
}

export interface ApprovePermissionsRequest {
  permissions: string[]
}

export interface UpdateSettingsRequest {
  settings: Record<string, unknown>
}

export interface CreateValidationRuleRequest {
  installation_id: string
  name: string
  description: string
  entity_type: string
  field_name: string
  rule_type: string
  rule_config: Record<string, unknown>
  error_message: string
  priority: number
  enabled: boolean
}

export interface UpdateValidationRuleRequest {
  name?: string
  description?: string
  entity_type?: string
  field_name?: string
  rule_type?: string
  rule_config?: Record<string, unknown>
  error_message?: string
  priority?: number
  enabled?: boolean
}

export interface CreateWorkflowRuleRequest {
  installation_id: string
  name: string
  description: string
  trigger_event: string
  conditions: Record<string, unknown>
  actions: Record<string, unknown>[]
  priority: number
  enabled: boolean
}

export interface UpdateWorkflowRuleRequest {
  name?: string
  description?: string
  trigger_event?: string
  conditions?: Record<string, unknown>
  actions?: Record<string, unknown>[]
  priority?: number
  enabled?: boolean
}

export interface ApplyTemplateRequest {
  template_id: string
}
