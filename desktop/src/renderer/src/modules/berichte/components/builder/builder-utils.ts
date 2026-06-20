import {
  Activity,
  BarChart3,
  Gauge,
  Hash,
  LineChart,
  PieChart,
  Table,
  TrendingUp,
  type LucideIcon,
} from 'lucide-react'
import type {
  BuilderQueryConfig,
  ReportDateRange,
  ReportFilter,
  ReportMeasure,
  ReportViewOptions,
  VisualizationType,
} from '@/api/berichte-types'
import type { ReportSource } from '../../report-sources/types'
import { fieldByKey } from '../../report-sources/types'

/** Full builder working state. Phases add UI; the shape is complete from E-1. */
export interface BuilderState {
  sourceId: string | null
  viz: VisualizationType
  /** Whether the user picked the viz manually (suppresses auto-select). */
  vizManual: boolean
  dimensions: string[]
  measures: ReportMeasure[]
  filters: ReportFilter[]
  dateRange?: ReportDateRange
  sort?: { field: string; dir: 'asc' | 'desc' }
  limit?: number
  options: ReportViewOptions
  name: string
}

export function emptyBuilderState(): BuilderState {
  return {
    sourceId: null,
    viz: 'bar',
    vizManual: false,
    dimensions: [],
    measures: [],
    filters: [],
    options: {},
    name: '',
  }
}

export const MAX_DIMENSIONS = 2

export interface VizMeta {
  type: VisualizationType
  labelKey: string
  label: string
  icon: LucideIcon
  hint: string
}

/** The visualization gallery (Style/Setup order roughly by frequency of use). */
export const VIZ_GALLERY: VizMeta[] = [
  { type: 'bar', labelKey: 'berichte.builder.viz.bar', label: 'Balken', icon: BarChart3, hint: 'Kategorien vergleichen' },
  { type: 'line', labelKey: 'berichte.builder.viz.line', label: 'Linie', icon: LineChart, hint: 'Verlauf über Zeit' },
  { type: 'area', labelKey: 'berichte.builder.viz.area', label: 'Fläche', icon: TrendingUp, hint: 'Verlauf mit Volumen' },
  { type: 'donut', labelKey: 'berichte.builder.viz.donut', label: 'Donut', icon: PieChart, hint: 'Anteile am Ganzen' },
  { type: 'kpi', labelKey: 'berichte.builder.viz.kpi', label: 'Kennzahl', icon: Hash, hint: 'Eine große Zahl' },
  { type: 'table', labelKey: 'berichte.builder.viz.table', label: 'Tabelle', icon: Table, hint: 'Rohdaten in Zeilen' },
  { type: 'combo', labelKey: 'berichte.builder.viz.combo', label: 'Kombi', icon: Activity, hint: 'Balken + Linie' },
  { type: 'gauge', labelKey: 'berichte.builder.viz.gauge', label: 'Tacho', icon: Gauge, hint: 'Wert gegen Ziel' },
]

/** Suggest a visualization from the current field selection (HubSpot-style). */
export function suggestViz(state: BuilderState, source: ReportSource | undefined): VisualizationType {
  const dimCount = state.dimensions.length
  const measCount = state.measures.length
  if (dimCount === 0 && measCount >= 1) return 'kpi'
  if (measCount >= 2 && dimCount <= 1) return 'combo'
  if (dimCount >= 1) {
    const first = fieldByKey(source ?? ({} as ReportSource), state.dimensions[0])
    if (first?.dataType === 'date') return 'line'
    return 'bar'
  }
  return 'bar'
}

/** Build a runnable query, or null if the selection is not yet meaningful. */
export function toQuery(state: BuilderState): BuilderQueryConfig | null {
  if (!state.sourceId) return null
  if (state.dimensions.length === 0 && state.measures.length === 0) return null
  return {
    kind: 'builder',
    sourceId: state.sourceId,
    viz: state.viz,
    dimensions: state.dimensions,
    measures: state.measures,
    filters: state.filters,
    dateRange: state.dateRange,
    sort: state.sort,
    limit: state.limit,
    options: state.options,
  }
}

/** Reconstruct builder state from a saved query (E-4 state restore). */
export function fromQuery(query: BuilderQueryConfig, name: string): BuilderState {
  return {
    sourceId: query.sourceId,
    viz: query.viz,
    vizManual: true,
    dimensions: query.dimensions ?? [],
    measures: query.measures ?? [],
    filters: query.filters ?? [],
    dateRange: query.dateRange,
    sort: query.sort,
    limit: query.limit,
    options: query.options ?? {},
    name,
  }
}
