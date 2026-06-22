/**
 * TypeScript types for the Berichte (Reports/BI) module.
 *
 * Mirrors backend/proto/berichte/v1/berichte.proto service surface
 * and backend/internal/berichte/models.go shape.
 * UUIDs are represented as strings; timestamps as ISO 8601 strings.
 * JSONB fields (query_config, params) are typed as Record<string, unknown>;
 * the gateway serialises them as raw JSON objects (like wiki content).
 */

import type { CodeBlock, QuoteBlock, SimpleTableBlock } from '@/components/shared/document'

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
  | 'hr'
  | 'zeiterfassung'
  | 'vertraege'
  | 'einkauf'
  | 'fuhrpark'
  | 'rapporte'
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

// ---------------------------------------------------------------------------
// Report Documents — multi-page authoring (Schicht 4)
//
// A ReportDocument is the multi-page report a user authors in the block editor:
// cover page + headings + rich text + embedded charts/tables/KPIs across many
// pages. It is distinct from ReportDefinition (= a single chart/data widget,
// "Schicht 3"). A chart block embeds a ReportDefinition by id, or carries an
// inline BuilderQueryConfig.
//
// Layout model: document → rows → columns → blocks. Default is one full-width
// column per row (top-down reading flow, carries long multi-page text). A row
// may split into 2–3 columns for KPI rows and "text beside chart" layouts.
// Atomic blocks (chart/table/kpi/image) never break across a page; text blocks
// flow across page boundaries.
// ---------------------------------------------------------------------------

/** Lifecycle status. No approval gate (too bureaucratic for SMEs). */
export type ReportDocStatus = 'draft' | 'final' | 'released' | 'archived'

export type ReportBlockType =
  | 'cover'
  | 'heading'
  | 'text'
  | 'chart'
  | 'table'
  | 'kpi'
  | 'callout'
  | 'bullet'
  | 'divider'
  | 'image'
  | 'pagebreak'

interface ReportBlockBase {
  id: string
  type: ReportBlockType
}

/** Cover page: title, optional subtitle, logo, author, date. */
export interface CoverBlock extends ReportBlockBase {
  type: 'cover'
  title: string
  subtitle?: string
  author?: string
  /** Render the document period/date line. */
  showDate?: boolean
  logoUrl?: string
}

export interface HeadingBlock extends ReportBlockBase {
  type: 'heading'
  level: 1 | 2
  text: string
}

/** Rich text (TipTap HTML). Flows across pages. */
export interface TextBlock extends ReportBlockBase {
  type: 'text'
  html: string
}

/** Embedded chart widget — references a saved definition or an inline query. */
export interface ChartBlock extends ReportBlockBase {
  type: 'chart'
  definitionId?: string
  query?: BuilderQueryConfig
  /** Visualization for definition-backed charts (builder queries carry their own). */
  viz?: VisualizationType
  caption?: string
}

/** Embedded data table — references a saved definition or an inline query. */
export interface TableBlock extends ReportBlockBase {
  type: 'table'
  definitionId?: string
  query?: BuilderQueryConfig
  caption?: string
}

/** Single highlighted metric. A KPI row = one row with several kpi blocks. */
export interface KpiBlock extends ReportBlockBase {
  type: 'kpi'
  label: string
  value: string
  unit?: string
  changePercent?: number | null
  /** Source label (module / data source). */
  source?: string
}

export type CalloutVariant = 'info' | 'success' | 'warning' | 'recommendation'

/** Callout / handling recommendation. */
export interface CalloutBlock extends ReportBlockBase {
  type: 'callout'
  variant: CalloutVariant
  title?: string
  html: string
}

export interface BulletBlock extends ReportBlockBase {
  type: 'bullet'
  items: string[]
  ordered?: boolean
}

export interface DividerBlock extends ReportBlockBase {
  type: 'divider'
}

export interface ImageBlock extends ReportBlockBase {
  type: 'image'
  url: string
  caption?: string
  alt?: string
}

/** Explicit page break in the printed/PDF output. */
export interface PageBreakBlock extends ReportBlockBase {
  type: 'pagebreak'
}

export type ReportBlock =
  | CoverBlock
  | HeadingBlock
  | TextBlock
  | ChartBlock
  | TableBlock
  | KpiBlock
  | CalloutBlock
  | BulletBlock
  | DividerBlock
  | ImageBlock
  | PageBreakBlock
  // Shared document-engine special blocks registered in the berichte registry.
  | CodeBlock
  | SimpleTableBlock
  | QuoteBlock

export interface ReportDocColumn {
  id: string
  /** Relative width weight (e.g. 1/1 = 50/50, 3/2 = 60/40). Default 1. */
  width?: number
  blocks: ReportBlock[]
}

export interface ReportRow {
  id: string
  /** 1–3 columns. One column = full width. */
  columns: ReportDocColumn[]
}

/** Document-level page setup (header/footer/page numbers, palette). */
export interface ReportDocSettings {
  showHeader?: boolean
  headerText?: string
  showFooter?: boolean
  showPageNumbers?: boolean
  logoUrl?: string
  /** Chart palette id, reused from the builder palettes. */
  palette?: string
  accentColor?: string
}

export interface ReportDocument {
  id: string
  tenant_id: string
  title: string
  description?: string
  module: ReportModule
  status: ReportDocStatus
  rows: ReportRow[]
  settings: ReportDocSettings
  /** Template this document was created from, if any. */
  template_id?: string | null
  created_by: string | null
  created_at: string
  updated_at: string
  /** Set when status first transitions to 'released'. */
  released_at?: string | null
}

export interface CreateReportDocumentInput {
  /** Optional — falls back to the template title or "Neuer Bericht". */
  title?: string
  description?: string
  module?: ReportModule
  template_id?: string | null
  rows?: ReportRow[]
  settings?: ReportDocSettings
}

export interface UpdateReportDocumentInput {
  title?: string
  description?: string
  module?: ReportModule
  status?: ReportDocStatus
  rows?: ReportRow[]
  settings?: ReportDocSettings
}

export interface ListReportDocumentsParams {
  module?: ReportModule
  status?: ReportDocStatus
  search?: string
}

export interface ListReportDocumentsResponse {
  documents: ReportDocument[]
  total: number
}

export interface ReportDocumentResponse {
  document: ReportDocument
}

/**
 * An external read-only share link for a report document (R-5c). The demo
 * stores tokens FE-side; the actual unauthenticated public endpoint is Luke's.
 */
export interface ReportShareToken {
  id: string
  document_id: string
  token: string
  /** null = never expires. */
  expires_at: string | null
  has_password: boolean
  view_count: number
  created_at: string
}

export interface CreateReportShareInput {
  /** Days until expiry; null/0 = never expires. */
  expires_in_days?: number | null
  password?: string
}

export interface ReportSharesResponse {
  shares: ReportShareToken[]
}

export interface ReportShareResponse {
  share: ReportShareToken
}

/** A starter template: a prebuilt block structure a new report is created from. */
export interface ReportTemplate {
  id: string
  title: string
  description: string
  module: ReportModule
  /** Lucide icon name resolved in the picker UI. */
  icon?: string
  rows: ReportRow[]
  settings?: ReportDocSettings
}

export interface ListReportTemplatesResponse {
  templates: ReportTemplate[]
}
