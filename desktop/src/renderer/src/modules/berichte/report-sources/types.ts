import type { LucideIcon } from 'lucide-react'
import type { AggregationFn, ReportModule, VisualizationType } from '@/api/berichte-types'

/** Data type of a source field — drives which filter operators are offered. */
export type FieldDataType = 'number' | 'string' | 'date' | 'enum' | 'boolean'

/**
 * Role of a field in the builder:
 *  - 'dimension' → group-by / category / X-axis (any data type)
 *  - 'measure'   → aggregated numeric value / Y-axis
 */
export type FieldRole = 'dimension' | 'measure'

/** Number display hint for a measure. */
export type FieldFormat = 'currency' | 'percent' | 'number' | 'date'

export interface EnumOption {
  value: string
  /** German default label; rendered via t(labelKey, { defaultValue }). */
  label: string
}

/** A single selectable field of a report source. */
export interface FieldDefinition {
  /** Stable key, used in sample rows and BuilderQueryConfig. */
  key: string
  /** i18n key, e.g. `berichte.fields.finanzen.total_gross`. */
  labelKey: string
  /** German default label (defaultValue until i18n keys land in E-5). */
  label: string
  dataType: FieldDataType
  role: FieldRole
  /** For enum fields: the allowed values. */
  enumValues?: EnumOption[]
  /** Display format for measures. */
  format?: FieldFormat
  /** Suggested aggregation when used as a measure. */
  defaultAgg?: AggregationFn
}

/**
 * A report data source. Each finished module registers one in the registry.
 * Adding a new module's reporting = drop a new `<module>.source.ts` + one line
 * in `registry.ts` — the only shared merge point.
 */
export interface ReportSource {
  /** Registry id, also used as BuilderQueryConfig.sourceId. */
  id: string
  module: ReportModule
  labelKey: string
  /** German default label. */
  label: string
  icon: LucideIcon
  /** Short description shown in the source picker. */
  description: string
  /** Sensible default visualization for this source. */
  defaultViz: VisualizationType
  fields: FieldDefinition[]
  /**
   * Deterministic demo rows for the preview executor (mock-first).
   * The MSW /preview handler applies filters/aggregation/sort/limit to these.
   */
  sampleRows: (count?: number) => Record<string, unknown>[]
}

/** Convenience: split a source's fields by role. */
export function dimensionsOf(source: ReportSource): FieldDefinition[] {
  return source.fields.filter((f) => f.role === 'dimension')
}
export function measuresOf(source: ReportSource): FieldDefinition[] {
  return source.fields.filter((f) => f.role === 'measure')
}
export function fieldByKey(source: ReportSource, key: string): FieldDefinition | undefined {
  return source.fields.find((f) => f.key === key)
}
