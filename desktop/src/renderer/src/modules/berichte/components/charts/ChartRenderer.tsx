import { useMemo } from 'react'
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  ComposedChart,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  PolarAngleAxis,
  RadialBar,
  RadialBarChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import type { ReportColumn, ReportResult, ReportViewOptions, VisualizationType } from '@/api/berichte-types'
import { useChartTheme, categoricalPalette, type ChartTheme } from '../../utils/chartTheme'
import { usePrefersReducedMotion } from '../../utils/chartMotion'
import { formatAxisTick, formatValue } from './chart-format'

interface ChartRendererProps {
  result: ReportResult
  viz: VisualizationType
  options?: ReportViewOptions
  height?: number
}

/** Cycle the 4-colour categorical palette for an arbitrary number of series. */
function paletteOf(theme: ChartTheme, n: number): string[] {
  const base = categoricalPalette(theme)
  return Array.from({ length: n }, (_, i) => base[i % base.length])
}

/** Pivot ReportSeries[] into a recharts-friendly row array. */
function pivot(result: ReportResult): { data: Record<string, unknown>[]; keys: string[] } {
  const series = result.series ?? []
  if (series.length === 0) return { data: [], keys: [] }
  const labels = series[0].data.map((d) => d.label)
  const keys = series.map((s) => s.label)
  const data = labels.map((label, i) => {
    const row: Record<string, unknown> = { label }
    series.forEach((s) => {
      row[s.label] = s.data[i]?.value ?? 0
    })
    return row
  })
  return { data, keys }
}

function measureType(result: ReportResult): ReportColumn['type'] {
  return result.columns.find((_, i) => i > 0)?.type ?? 'number'
}

interface TooltipPayloadItem {
  name?: string
  value?: number
  color?: string
}
function ChartTooltip({
  active,
  payload,
  label,
  type,
}: {
  active?: boolean
  payload?: TooltipPayloadItem[]
  label?: string
  type?: ReportColumn['type']
}) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border border-border bg-popover px-3 py-2 text-xs shadow-md">
      <p className="mb-1 font-medium text-foreground">{label}</p>
      {payload.map((p, i) => (
        <div key={i} className="flex items-center gap-2 text-muted-foreground">
          <span className="h-2 w-2 rounded-full" style={{ backgroundColor: p.color }} />
          <span>{p.name}</span>
          <span className="ml-auto font-medium text-foreground">{formatValue(p.value, type)}</span>
        </div>
      ))}
    </div>
  )
}

export function ChartRenderer({ result, viz, options, height = 320 }: ChartRendererProps) {
  const theme = useChartTheme()
  const reduceMotion = usePrefersReducedMotion()
  const animate = !reduceMotion
  const { data, keys } = useMemo(() => pivot(result), [result])
  const vType = measureType(result)
  const colors = paletteOf(theme, Math.max(keys.length, result.rows.length))
  const showLegend = options?.showLegend ?? keys.length > 1
  const legendPos = options?.legendPosition ?? 'bottom'

  // ---- Table ----
  if (viz === 'table') {
    return (
      <div className="max-h-[420px] overflow-auto rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-secondary">
            <tr>
              {result.columns.map((c) => (
                <th key={c.key} className="px-3 py-2 text-left font-medium text-muted-foreground">
                  {c.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {result.rows.map((row, i) => (
              <tr key={i} className="border-t border-border-muted">
                {result.columns.map((c) => (
                  <td key={c.key} className="px-3 py-2 text-foreground">
                    {formatValue(row[c.key], c.type)}
                  </td>
                ))}
              </tr>
            ))}
            {result.rows.length === 0 && (
              <tr>
                <td colSpan={result.columns.length} className="px-3 py-6 text-center text-muted-foreground">
                  Keine Daten
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    )
  }

  // ---- KPI ----
  if (viz === 'kpi') {
    const firstKey = keys[0]
    const total =
      (result.totals && Object.values(result.totals)[0] != null
        ? Number(Object.values(result.totals)[0])
        : (result.series?.[0]?.data.reduce((s, d) => s + d.value, 0) ?? 0))
    const spark = result.series?.[0]?.data ?? []
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-8" style={{ minHeight: height }}>
        <p className="text-sm text-muted-foreground">{firstKey ?? result.columns[1]?.label}</p>
        <p className="text-5xl font-semibold tracking-tight text-foreground">
          {formatValue(total, vType)}
        </p>
        {spark.length > 1 && (
          <div className="h-16 w-full max-w-sm">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={spark}>
                <Line
                  type="monotone"
                  dataKey="value"
                  stroke={theme.primary}
                  strokeWidth={2}
                  dot={false}
                  isAnimationActive={animate}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>
    )
  }

  // ---- Gauge ----
  if (viz === 'gauge') {
    const points = result.series?.[0]?.data ?? []
    // Current value = latest period; target = 110 % of the best period (or value if flat).
    const value = points.length
      ? points[points.length - 1].value
      : Number(Object.values(result.totals ?? {})[0] ?? 0)
    const maxPoint = points.length ? Math.max(...points.map((d) => d.value)) : value
    const target = Math.max(maxPoint, value) * 1.1 || 1
    const pct = Math.min(100, Math.max(0, Math.round((value / target) * 100)))
    const gaugeData = [{ name: 'value', value: pct, fill: theme.primary }]
    return (
      <div className="relative mx-auto flex items-center justify-center" style={{ width: '100%', maxWidth: height + 120, height }}>
        <ResponsiveContainer width="100%" height="100%">
          <RadialBarChart innerRadius="68%" outerRadius="100%" data={gaugeData} startAngle={210} endAngle={-30}>
            <PolarAngleAxis type="number" domain={[0, 100]} tick={false} />
            <RadialBar background dataKey="value" cornerRadius={8} isAnimationActive={animate} />
          </RadialBarChart>
        </ResponsiveContainer>
        <div className="absolute flex flex-col items-center">
          <span className="text-4xl font-semibold text-foreground">{pct} %</span>
          <span className="text-xs text-muted-foreground">{formatValue(value, vType)}</span>
        </div>
      </div>
    )
  }

  // ---- Donut ----
  if (viz === 'donut') {
    const donutData = (result.series?.[0]?.data ?? []).map((d) => ({ name: d.label, value: d.value }))
    return (
      <div className="mx-auto" style={{ width: '100%', maxWidth: height + 120, height }}>
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={donutData}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="50%"
              startAngle={90}
              endAngle={-270}
              innerRadius="55%"
              outerRadius="80%"
              paddingAngle={1}
              isAnimationActive={animate}
              label={options?.showDataLabels ? (e: { name?: string }) => e.name ?? '' : undefined}
            >
              {donutData.map((_, i) => (
                <Cell key={i} fill={colors[i % colors.length]} />
              ))}
            </Pie>
            <Tooltip content={<ChartTooltip type={vType} />} />
            {showLegend && <Legend verticalAlign={legendPos === 'top' ? 'top' : 'bottom'} />}
          </PieChart>
        </ResponsiveContainer>
      </div>
    )
  }

  // ---- Combo (bars + line) ----
  if (viz === 'combo') {
    return (
      <ResponsiveContainer width="100%" height={height}>
        <ComposedChart data={data} margin={{ top: 8, right: 8, bottom: 4, left: 4 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={theme.grid} vertical={false} />
          <XAxis dataKey="label" tick={{ fontSize: 11, fill: theme.muted }} />
          <YAxis tick={{ fontSize: 11, fill: theme.muted }} tickFormatter={(v) => formatAxisTick(v, vType)} />
          <Tooltip content={<ChartTooltip type={vType} />} />
          {showLegend && <Legend />}
          {keys[0] && <Bar dataKey={keys[0]} fill={colors[0]} radius={[4, 4, 0, 0]} isAnimationActive={animate} />}
          {keys[1] && (
            <Line type="monotone" dataKey={keys[1]} stroke={colors[1]} strokeWidth={2} dot={false} isAnimationActive={animate} />
          )}
        </ComposedChart>
      </ResponsiveContainer>
    )
  }

  // ---- Area ----
  if (viz === 'area') {
    return (
      <ResponsiveContainer width="100%" height={height}>
        <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 4, left: 4 }}>
          <defs>
            {keys.map((k, i) => (
              <linearGradient key={k} id={`grad-${i}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={colors[i]} stopOpacity={0.7} />
                <stop offset="100%" stopColor={colors[i]} stopOpacity={0.05} />
              </linearGradient>
            ))}
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke={theme.grid} vertical={false} />
          <XAxis dataKey="label" tick={{ fontSize: 11, fill: theme.muted }} />
          <YAxis tick={{ fontSize: 11, fill: theme.muted }} tickFormatter={(v) => formatAxisTick(v, vType)} />
          <Tooltip content={<ChartTooltip type={vType} />} />
          {showLegend && <Legend />}
          {keys.map((k, i) => (
            <Area
              key={k}
              type="monotone"
              dataKey={k}
              stroke={colors[i]}
              fill={`url(#grad-${i})`}
              strokeWidth={2}
              stackId={options?.stacked ? '1' : undefined}
              isAnimationActive={animate}
            />
          ))}
        </AreaChart>
      </ResponsiveContainer>
    )
  }

  // ---- Line ----
  if (viz === 'line') {
    return (
      <ResponsiveContainer width="100%" height={height}>
        <LineChart data={data} margin={{ top: 8, right: 8, bottom: 4, left: 4 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={theme.grid} vertical={false} />
          <XAxis dataKey="label" tick={{ fontSize: 11, fill: theme.muted }} />
          <YAxis tick={{ fontSize: 11, fill: theme.muted }} tickFormatter={(v) => formatAxisTick(v, vType)} />
          <Tooltip content={<ChartTooltip type={vType} />} />
          {showLegend && <Legend />}
          {keys.map((k, i) => (
            <Line
              key={k}
              type="monotone"
              dataKey={k}
              stroke={colors[i]}
              strokeWidth={2}
              dot={false}
              isAnimationActive={animate}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    )
  }

  // ---- Bar (default) ----
  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart data={data} margin={{ top: 8, right: 8, bottom: 4, left: 4 }}>
        <CartesianGrid strokeDasharray="3 3" stroke={theme.grid} vertical={false} />
        <XAxis dataKey="label" tick={{ fontSize: 11, fill: theme.muted }} />
        <YAxis tick={{ fontSize: 11, fill: theme.muted }} tickFormatter={(v) => formatAxisTick(v, vType)} />
        <Tooltip content={<ChartTooltip type={vType} />} cursor={{ fill: theme.grid, opacity: 0.3 }} />
        {showLegend && <Legend />}
        {keys.map((k, i) => (
          <Bar
            key={k}
            dataKey={k}
            fill={colors[i]}
            radius={[4, 4, 0, 0]}
            stackId={options?.stacked ? '1' : undefined}
            isAnimationActive={animate}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  )
}
