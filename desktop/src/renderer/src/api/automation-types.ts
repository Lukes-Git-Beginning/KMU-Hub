/**
 * TypeScript types for the Automation module.
 *
 * Covers all automation entities: automations, executions, templates,
 * trigger/action definitions, conditions, and statistics.
 */

// ---------------------------------------------------------------------------
// Core automation entity
// ---------------------------------------------------------------------------

export type AutomationScope = 'personal' | 'team' | 'organization'
export type ExecutionStatus = 'running' | 'completed' | 'failed' | 'skipped' | 'aborted'
export type ConditionMode = 'simple' | 'expression'
export type OnErrorBehavior = 'continue' | 'abort'
export type TemplateCategory = 'vertrieb' | 'personal' | 'finanzen' | 'kommunikation'
export type TemplateComplexity = 'einfach' | 'mittel' | 'fortgeschritten'
export type FieldType = 'string' | 'number' | 'boolean' | 'date'

export interface Automation {
  id: string
  name: string
  description: string
  scope: AutomationScope
  owner_id: string
  trigger_type: string
  trigger_config: Record<string, unknown>
  conditions: ConditionConfig
  actions: ActionConfig[]
  is_active: boolean
  max_steps: number
  template_id: string | null
  last_triggered_at: string | null
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// Execution entities
// ---------------------------------------------------------------------------

export interface AutomationExecution {
  id: string
  automation_id: string
  chain_id: string
  trigger_event: Record<string, unknown>
  condition_result: boolean
  status: ExecutionStatus
  steps: ExecutionStep[]
  error_message: string
  started_at: string
  completed_at: string | null
  duration_ms: number
}

export interface ExecutionStep {
  action_type: string
  input: Record<string, unknown>
  output: Record<string, unknown>
  error: string
  duration_ms: number
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

export interface AutomationTemplate {
  id: string
  name: string
  description: string
  category: TemplateCategory
  complexity: TemplateComplexity
  trigger_type: string
  trigger_config: Record<string, unknown>
  conditions: ConditionConfig
  actions: ActionConfig[]
}

// ---------------------------------------------------------------------------
// Trigger definitions (catalog)
// ---------------------------------------------------------------------------

export interface TriggerDefinition {
  type: string
  module: string
  name: string
  description: string
  event_type: string
  fields: TriggerField[]
  config_params: ConfigParam[]
}

export interface TriggerField {
  key: string
  label: string
  type: FieldType
  operators: string[]
}

// ---------------------------------------------------------------------------
// Action definitions (catalog)
// ---------------------------------------------------------------------------

export interface ActionDefinition {
  type: string
  module: string
  name: string
  description: string
  params: ConfigParam[]
  output_fields: OutputField[]
}

export interface ConfigParam {
  key: string
  label: string
  type: string
  required: boolean
  default_value: unknown
}

export interface OutputField {
  key: string
  label: string
  type: string
}

// ---------------------------------------------------------------------------
// Condition config
// ---------------------------------------------------------------------------

export interface ConditionConfig {
  mode: ConditionMode
  simple: SimpleCondition | null
  expression: string
}

export interface SimpleCondition {
  and?: SimpleCondition[]
  or?: SimpleCondition[]
  field?: string
  operator?: string
  value?: unknown
}

// ---------------------------------------------------------------------------
// Action config
// ---------------------------------------------------------------------------

export interface ActionConfig {
  type: string
  config: Record<string, unknown>
  on_error: OnErrorBehavior
}

// ---------------------------------------------------------------------------
// Statistics
// ---------------------------------------------------------------------------

/**
 * Wire shape matches `GetAutomationStatsResponse` (backend/proto/automation/v1/automation.proto)
 * exactly — protojson with EmitUnpopulated:false, so zero-value fields are omitted (read via `?? 0`).
 * `success_rate` does not exist on the backend message; it is computed client-side in
 * automation-client.ts from successful_executions / total_executions.
 */
export interface AutomationStats {
  total_automations: number
  active_automations: number
  total_executions: number
  successful_executions: number
  failed_executions: number
  skipped_executions: number
  average_duration_ms: number
  success_rate: number
}

// ---------------------------------------------------------------------------
// API response wrappers
// ---------------------------------------------------------------------------

export interface AutomationListResponse {
  automations: Automation[]
  total_count: number
}

export interface ExecutionListResponse {
  executions: AutomationExecution[]
  total_count: number
}

export interface TemplateListResponse {
  templates: AutomationTemplate[]
}

export interface TriggerDefinitionListResponse {
  triggers: TriggerDefinition[]
}

export interface ActionDefinitionListResponse {
  actions: ActionDefinition[]
}

export interface TestConditionResponse {
  matches: boolean
  error: string
}

export interface DryRunResponse {
  condition_result: boolean
  steps: ExecutionStep[]
  would_execute: boolean
}

// ---------------------------------------------------------------------------
// Request params
// ---------------------------------------------------------------------------

export interface AutomationListParams {
  owner_id?: string
  scope?: AutomationScope
  trigger_type?: string
  is_active?: boolean
  limit?: number
  offset?: number
}

export interface ExecutionListParams {
  status?: ExecutionStatus
  limit?: number
  offset?: number
}
