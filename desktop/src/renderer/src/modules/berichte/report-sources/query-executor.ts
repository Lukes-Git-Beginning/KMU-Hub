/**
 * Demo query executor — interprets a BuilderQueryConfig against a source's
 * sample rows and produces a ReportResult (columns + rows + series + totals).
 *
 * This is the FE/demo stand-in for the real backend query executor (Luke 🔒).
 * The logic is intentionally complete (all filter operators, all aggregations,
 * up to 2 group-by dimensions) so that E-2 (filters) and E-3 (aggregation) only
 * need to add UI, not engine work.
 */
import type {
  AggregationFn,
  BuilderQueryConfig,
  ReportColumn,
  ReportFilter,
  ReportResult,
  ReportSeries,
} from '@/api/berichte-types'
import type { FieldDefinition, ReportSource } from './types'
import { fieldByKey } from './types'

const MONTHS_DE = ['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez']

// ---------------------------------------------------------------------------
// Filtering
// ---------------------------------------------------------------------------

function daysAgoIso(days: number): string {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  d.setDate(d.getDate() - days)
  return d.toISOString().slice(0, 10)
}

function matchesFilter(row: Record<string, unknown>, filter: ReportFilter): boolean {
  const raw = row[filter.field]
  const { operator, value, value2 } = filter
  switch (operator) {
    case 'eq':
      return String(raw) === String(value)
    case 'neq':
      return String(raw) !== String(value)
    case 'gt':
      return Number(raw) > Number(value)
    case 'lt':
      return Number(raw) < Number(value)
    case 'between':
      return Number(raw) >= Number(value) && Number(raw) <= Number(value2)
    case 'contains':
      return String(raw).toLowerCase().includes(String(value).toLowerCase())
    case 'startsWith':
      return String(raw).toLowerCase().startsWith(String(value).toLowerCase())
    case 'isEmpty':
      return raw === null || raw === undefined || raw === ''
    case 'in':
      return Array.isArray(value) && (value as unknown[]).map(String).includes(String(raw))
    case 'notIn':
      return Array.isArray(value) && !(value as unknown[]).map(String).includes(String(raw))
    case 'before':
      return String(raw) < String(value)
    case 'after':
      return String(raw) > String(value)
    case 'inLastDays': {
      const cutoff = daysAgoIso(Number(value))
      return String(raw) >= cutoff
    }
    default:
      return true
  }
}

function applyDateRange(
  rows: Record<string, unknown>[],
  query: BuilderQueryConfig,
): Record<string, unknown>[] {
  const dr = query.dateRange
  if (!dr || !dr.field) return rows
  let from: string | undefined
  let to: string | undefined
  const today = new Date()
  const iso = (d: Date) => d.toISOString().slice(0, 10)
  switch (dr.preset) {
    case 'last7':
      from = daysAgoIso(7)
      break
    case 'last30':
      from = daysAgoIso(30)
      break
    case 'last90':
      from = daysAgoIso(90)
      break
    case 'thisMonth':
      from = iso(new Date(today.getFullYear(), today.getMonth(), 1))
      break
    case 'lastMonth':
      from = iso(new Date(today.getFullYear(), today.getMonth() - 1, 1))
      to = iso(new Date(today.getFullYear(), today.getMonth(), 0))
      break
    case 'thisQuarter': {
      const q = Math.floor(today.getMonth() / 3)
      from = iso(new Date(today.getFullYear(), q * 3, 1))
      break
    }
    case 'thisYear':
      from = iso(new Date(today.getFullYear(), 0, 1))
      break
    case 'custom':
      from = dr.from
      to = dr.to
      break
    default:
      return rows
  }
  return rows.filter((row) => {
    const v = String(row[dr.field as string] ?? '')
    if (from && v < from) return false
    if (to && v > to) return false
    return true
  })
}

function applyFilters(
  rows: Record<string, unknown>[],
  query: BuilderQueryConfig,
): Record<string, unknown>[] {
  let out = applyDateRange(rows, query)
  for (const f of query.filters ?? []) {
    out = out.filter((row) => matchesFilter(row, f))
  }
  return out
}

// ---------------------------------------------------------------------------
// Grouping / aggregation
// ---------------------------------------------------------------------------

/** Bucket key for a dimension value (dates collapse to month). */
function bucketValue(field: FieldDefinition | undefined, value: unknown): string {
  if (field?.dataType === 'date') {
    const s = String(value)
    const d = new Date(s)
    if (!Number.isNaN(d.getTime())) {
      return `${MONTHS_DE[d.getMonth()]} ${String(d.getFullYear()).slice(2)}`
    }
  }
  if (field?.dataType === 'enum') {
    return field.enumValues?.find((o) => o.value === String(value))?.label ?? String(value)
  }
  return String(value ?? '—')
}

/** Sort key that keeps month buckets chronological. */
function bucketSortKey(field: FieldDefinition | undefined, value: unknown): string {
  if (field?.dataType === 'date') {
    const d = new Date(String(value))
    if (!Number.isNaN(d.getTime())) {
      return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
    }
  }
  return String(value ?? '')
}

function aggregate(values: number[], fn: AggregationFn): number {
  if (fn === 'count') return values.length
  if (values.length === 0) return 0
  switch (fn) {
    case 'sum':
      return round(values.reduce((s, v) => s + v, 0))
    case 'avg':
      return round(values.reduce((s, v) => s + v, 0) / values.length)
    case 'min':
      return round(Math.min(...values))
    case 'max':
      return round(Math.max(...values))
    default:
      return 0
  }
}

function round(n: number): number {
  return Math.round(n * 100) / 100
}

function aggLabel(fn: AggregationFn): string {
  return { count: 'Anzahl', sum: 'Summe', avg: 'Durchschnitt', min: 'Minimum', max: 'Maximum' }[fn]
}

function colType(field: FieldDefinition | undefined): ReportColumn['type'] {
  if (!field) return 'number'
  if (field.format === 'currency') return 'currency'
  if (field.format === 'percent') return 'percent'
  if (field.dataType === 'date') return 'date'
  if (field.dataType === 'number') return 'number'
  return 'string'
}

// ---------------------------------------------------------------------------
// Executor
// ---------------------------------------------------------------------------

export function executeQuery(
  source: ReportSource,
  query: BuilderQueryConfig,
  rawRows: Record<string, unknown>[],
): ReportResult {
  const rows = applyFilters(rawRows, query)
  const now = new Date().toISOString()

  const dim0Key = query.dimensions[0]
  const dim1Key = query.dimensions[1]
  const dim0 = dim0Key ? fieldByKey(source, dim0Key) : undefined
  const dim1 = dim1Key ? fieldByKey(source, dim1Key) : undefined

  // Measures: fall back to a count when none chosen.
  const measures =
    query.measures.length > 0
      ? query.measures
      : [{ field: dim0Key ?? source.fields[0].key, agg: 'count' as AggregationFn }]

  // No dimension → single aggregate row (good for KPI / gauge).
  if (!dim0Key) {
    const totals: Record<string, unknown> = {}
    const cols: ReportColumn[] = []
    const singleRow: Record<string, unknown> = {}
    for (const m of measures) {
      const f = fieldByKey(source, m.field)
      const vals = rows.map((r) => Number(r[m.field])).filter((v) => !Number.isNaN(v))
      const val = aggregate(m.agg === 'count' ? rows.map(() => 1) : vals, m.agg)
      const key = m.field
      const label = `${aggLabel(m.agg)} ${m.agg === 'count' ? '' : f?.label ?? m.field}`.trim()
      cols.push({ key, label, type: m.agg === 'count' ? 'number' : colType(f) })
      totals[key] = val
      singleRow[key] = val
    }
    return {
      columns: cols,
      rows: [singleRow],
      series: [],
      totals,
      meta: { generated_at: now, row_count: rows.length, definition_id: query.sourceId },
    }
  }

  // Group rows by dim0 (and optionally dim1).
  const buckets = new Map<string, { label: string; sort: string; rows: Record<string, unknown>[] }>()
  for (const row of rows) {
    const key = bucketValue(dim0, row[dim0Key])
    if (!buckets.has(key)) {
      buckets.set(key, { label: key, sort: bucketSortKey(dim0, row[dim0Key]), rows: [] })
    }
    buckets.get(key)!.rows.push(row)
  }
  // Aggregated value of a measure within a bucket (for sort + series).
  const bucketMeasure = (b: { rows: Record<string, unknown>[] }, m: { field: string; agg: AggregationFn }) => {
    const vals = b.rows.map((r) => Number(r[m.field])).filter((v) => !Number.isNaN(v))
    return aggregate(m.agg === 'count' ? b.rows.map(() => 1) : vals, m.agg)
  }

  let ordered = [...buckets.values()]
  // Sort: by a chosen measure (desc/asc) else natural/chronological order.
  if (query.sort) {
    const sm = measures.find((m) => m.field === query.sort!.field)
    ordered.sort((a, b) => {
      const av = sm ? bucketMeasure(a, sm) : a.sort
      const bv = sm ? bucketMeasure(b, sm) : b.sort
      const cmp =
        typeof av === 'number' && typeof bv === 'number' ? av - bv : String(av).localeCompare(String(bv))
      return query.sort!.dir === 'desc' ? -cmp : cmp
    })
  } else {
    ordered.sort((a, b) => a.sort.localeCompare(b.sort))
  }
  // Top-N limit applies to the buckets so charts and table agree.
  if (query.limit && query.limit > 0) ordered = ordered.slice(0, query.limit)

  const columns: ReportColumn[] = [{ key: dim0Key, label: dim0?.label ?? dim0Key, type: colType(dim0) }]
  const series: ReportSeries[] = []
  const totals: Record<string, unknown> = {}

  if (dim1Key) {
    // Two dimensions, one measure → series per dim1 value.
    const measure = measures[0]
    const f = fieldByKey(source, measure.field)
    const dim1Values = [...new Set(rows.map((r) => bucketValue(dim1, r[dim1Key])))]
    for (const d1 of dim1Values) {
      columns.push({ key: `${measure.field}__${d1}`, label: d1, type: colType(f) })
      series.push({
        id: d1,
        label: d1,
        data: ordered.map((b) => {
          const sub = b.rows.filter((r) => bucketValue(dim1, r[dim1Key]) === d1)
          const vals = sub.map((r) => Number(r[measure.field])).filter((v) => !Number.isNaN(v))
          return { label: b.label, value: aggregate(measure.agg === 'count' ? sub.map(() => 1) : vals, measure.agg) }
        }),
      })
    }
  } else {
    // One dimension, N measures → series per measure.
    for (const m of measures) {
      const f = fieldByKey(source, m.field)
      const label = m.agg === 'count' ? aggLabel('count') : `${aggLabel(m.agg)} ${f?.label ?? m.field}`
      columns.push({ key: m.field, label, type: m.agg === 'count' ? 'number' : colType(f) })
      let runningTotal = 0
      series.push({
        id: m.field,
        label,
        data: ordered.map((b) => {
          const vals = b.rows.map((r) => Number(r[m.field])).filter((v) => !Number.isNaN(v))
          const val = aggregate(m.agg === 'count' ? b.rows.map(() => 1) : vals, m.agg)
          runningTotal += val
          return { label: b.label, value: val }
        }),
      })
      totals[m.field] = round(runningTotal)
    }
  }

  // Build table rows from the (already sorted + limited) ordered buckets.
  const tableRows: Record<string, unknown>[] = ordered.map((b, i) => {
    const row: Record<string, unknown> = { [dim0Key]: b.label }
    for (const s of series) {
      const colKey = dim1Key ? `${measures[0].field}__${s.id}` : s.id
      row[colKey] = s.data[i]?.value ?? 0
    }
    return row
  })

  return {
    columns,
    rows: tableRows,
    series,
    totals,
    meta: { generated_at: now, row_count: rows.length, definition_id: query.sourceId },
  }
}
