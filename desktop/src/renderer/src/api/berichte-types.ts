/**
 * TypeScript types for the Berichte (Reports/BI) module.
 *
 * Mirrors backend/proto/berichte/v1/berichte.proto service surface
 * and backend/internal/berichte/models.go shape.
 * UUIDs are represented as strings; timestamps as ISO 8601 strings.
 * JSONB fields (query_config, params) are typed as Record<string, unknown>;
 * the gateway serialises them as raw JSON objects (like wiki content).
 */

// ---------------------------------------------------------------------------
// Enum-like string unions
// ---------------------------------------------------------------------------

export type ReportFormat = 'pdf' | 'csv' | 'xlsx'
export type ReportModule =
  | 'finanzen'
  | 'crm'
  | 'helpdesk'
  | 'inventar'
  | 'produktion'
  | 'work'
  | 'kommunikation'
  | 'cross'
export type ReportKind = 'system' | 'custom'
export type ScheduleRunStatus = 'success' | 'failed' | 'skipped'
export type RunTrigger = 'manual' | 'scheduled' | 'api'
export type RunStatus = 'success' | 'failed'

export type QueryConfig = Record<string, unknown>
export type ReportParams = Record<string, unknown>

// ---------------------------------------------------------------------------
// Domain models
// ---------------------------------------------------------------------------

export interface ReportDefinition {
  id: string
  tenant_id: string
  name: string
  description: string
  module: ReportModule
  kind: ReportKind
  query_config: QueryConfig
  default_format: ReportFormat
  created_by: string | null
  is_published: boolean
  created_at: string
  updated_at: string
}

export interface ReportColumn {
  key: string
  label: string
  type?: 'string' | 'number' | 'date' | 'currency' | 'percent'
}

export interface ReportSeriesPoint {
  label: string
  value: number
}

export interface ReportSeries {
  id: string
  label: string
  data: ReportSeriesPoint[]
}

export interface ReportMeta {
  generated_at: string
  row_count: number
  definition_id: string
  from_cache?: boolean
}

export interface ReportResult {
  columns: ReportColumn[]
  rows: Record<string, unknown>[]
  series?: ReportSeries[]
  totals?: Record<string, unknown>
  meta: ReportMeta
}

export interface ReportRun {
  id: string
  tenant_id: string
  definition_id: string
  schedule_id: string | null
  trigger: RunTrigger
  params: ReportParams
  duration_ms: number | null
  row_count: number | null
  status: RunStatus
  error: string | null
  started_at: string
  completed_at: string | null
}

export interface ReportSchedule {
  id: string
  tenant_id: string
  definition_id: string
  name: string
  cron_expression: string
  recipients: string[]
  format: ReportFormat
  params: ReportParams
  active: boolean
  last_run_at: string | null
  last_run_status: ScheduleRunStatus | null
  last_run_error: string | null
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface DashboardKPI {
  id: string
  label: string
  /** Proto defines value as string for flexibility (percent, currency, count). */
  value: string
  unit: string
  change_percent: number | null
  module_id: ReportModule | string
}

// ---------------------------------------------------------------------------
// Request inputs
// ---------------------------------------------------------------------------

export interface CreateDefinitionInput {
  name: string
  description?: string
  module: ReportModule
  kind?: ReportKind
  query_config: QueryConfig | BuilderQueryConfig
  default_format?: ReportFormat
  is_published?: boolean
}

export interface UpdateDefinitionInput {
  name?: string
  description?: string
  module?: ReportModule
  query_config?: QueryConfig | BuilderQueryConfig
  default_format?: ReportFormat
  is_published?: boolean
}

export interface ListDefinitionsParams {
  module?: ReportModule
  kind?: ReportKind
  is_published?: boolean
  search?: string
  page?: number
  page_size?: number
  sort_by?: string
  sort_desc?: boolean
}

export interface RunReportInput {
  params?: ReportParams
  force_refresh?: boolean
  trigger?: RunTrigger
}

export interface ExportReportInput {
  format: ReportFormat
  params?: ReportParams
}

export interface CreateScheduleInput {
  definition_id: string
  name: string
  cron_expression: string
  recipients: string[]
  format: ReportFormat
  params?: ReportParams
  active?: boolean
}

export interface UpdateScheduleInput {
  name?: string
  cron_expression?: string
  recipients?: string[]
  format?: ReportFormat
  params?: ReportParams
  active?: boolean
}

export interface ListSchedulesParams {
  definition_id?: string
  active?: boolean
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Response envelopes
// ---------------------------------------------------------------------------

export interface ListDefinitionsResponse {
  definitions: ReportDefinition[]
  total: number
}

export interface DefinitionResponse {
  definition: ReportDefinition
}

export interface RunReportResponse {
  result: ReportResult
  run: ReportRun
}

export interface InvalidateCacheResponse {
  evicted: number
}

export interface ListSchedulesResponse {
  schedules: ReportSchedule[]
  total: number
}

export interface ScheduleResponse {
  schedule: ReportSchedule
}

export interface DashboardKPIsResponse {
  kpis: DashboardKPI[]
  generated_at: string
}

/** Bundled export download with filename parsed from Content-Disposition. */
export interface ExportedReport {
  blob: Blob
  filename: string
  content_type: string
}

// ---------------------------------------------------------------------------
// Report Builder — typed no-code query schema
//
// The builder produces a BuilderQueryConfig, which is persisted verbatim into
// ReportDefinition.query_config (JSONB). The backend query executor (Luke) reads
// it; in demo mode the MSW /preview handler interprets it against each source's
// sample rows. This is the FE contract for the self-service report builder.
// ---------------------------------------------------------------------------

/** Chart kinds the builder can render. */
export type VisualizationType =
  | 'table'
  | 'bar'
  | 'line'
  | 'area'
  | 'donut'
  | 'kpi'
  | 'combo'
  | 'gauge'

/** Aggregation functions applied to measures. */
export type AggregationFn = 'count' | 'sum' | 'avg' | 'min' | 'max'

/** Filter operators. Which ones are valid depends on the field's data type. */
export type FilterOperator =
  | 'eq'
  | 'neq'
  | 'gt'
  | 'lt'
  | 'between'
  | 'contains'
  | 'startsWith'
  | 'isEmpty'
  | 'in'
  | 'notIn'
  | 'before'
  | 'after'
  | 'inLastDays'

/** A single filter condition (all filters are AND-combined). */
export interface ReportFilter {
  /** Field key, resolved against the source's field definitions. */
  field: string
  operator: FilterOperator
  /** Primary value (string/number/string[] depending on operator). */
  value?: unknown
  /** Secondary value, only for `between`. */
  value2?: unknown
}

/** A measure (numeric field) with its aggregation. */
export interface ReportMeasure {
  field: string
  agg: AggregationFn
}

/** Relative date presets for the date-range quick picks. */
export type DateRangePreset =
  | 'last7'
  | 'last30'
  | 'last90'
  | 'thisMonth'
  | 'lastMonth'
  | 'thisQuarter'
  | 'thisYear'
  | 'custom'

export interface ReportDateRange {
  /** Date field the range applies to. */
  field?: string
  preset?: DateRangePreset
  from?: string
  to?: string
}

/** Per-report visual styling options (the "Style" tab). */
export interface ReportViewOptions {
  /** Named palette id from the chart theme; undefined = default. */
  palette?: string
  showLegend?: boolean
  legendPosition?: 'top' | 'right' | 'bottom'
  showDataLabels?: boolean
  axisXTitle?: string
  axisYTitle?: string
  /** Stack bars/areas instead of grouping. */
  stacked?: boolean
  /** Number format for measure values. */
  numberFormat?: 'plain' | 'thousands' | 'currency' | 'percent'
}

/**
 * The full no-code query produced by the builder. Stored in query_config.
 * `kind: 'builder'` distinguishes it from legacy system `query_config` shapes
 * (which use `{ kind: 'revenue' | 'tickets' | ... }`).
 */
export interface BuilderQueryConfig {
  kind: 'builder'
  /** Report-source id from the report-sources registry. */
  sourceId: string
  viz: VisualizationType
  /** Group-by dimension field keys (max 2). */
  dimensions: string[]
  /** Aggregated numeric fields. */
  measures: ReportMeasure[]
  /** AND-combined filter conditions. */
  filters: ReportFilter[]
  dateRange?: ReportDateRange
  sort?: { field: string; dir: 'asc' | 'desc' }
  /** Top-N row limit (e.g. top 10 customers). */
  limit?: number
  options?: ReportViewOptions
}

/** Type guard: is a query_config a builder query? */
export function isBuilderQuery(config: unknown): config is BuilderQueryConfig {
  return !!config && (config as { kind?: string }).kind === 'builder'
}

/** Request body for the demo preview executor (POST /berichte/preview). */
export interface PreviewReportInput {
  query: BuilderQueryConfig
}

export interface PreviewReportResponse {
  result: ReportResult
}
