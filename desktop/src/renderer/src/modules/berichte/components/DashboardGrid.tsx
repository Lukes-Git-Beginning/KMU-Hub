import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Bar,
  BarChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { BarChart3, Filter, TrendingUp, X } from 'lucide-react'
import { EmptyState } from '@/components/shared/EmptyState'
import { EmptyReports } from '@/components/shared/illustrations'
import type { DashboardKPI, ReportDefinition, ReportResult } from '@/api/berichte-types'
import { useRunReport } from '@/api/hooks/useBerichte'
import { useChartTheme } from '../utils/chartTheme'
import { usePrefersReducedMotion } from '../utils/chartMotion'
import { KPICard } from './KPICard'

interface DashboardGridProps {
  kpis: DashboardKPI[]
  definitions: ReportDefinition[]
  moduleFilter: string
  onModuleFilterChange: (next: string) => void
  moduleOptions: { id: string; name: string }[]
  isLoading?: boolean
}

/**
 * Metrics whose decrease is "good" (lower is better).
 * Identified by label-substring match since we no longer have stable IDs.
 */
const INVERT_MARKERS = ['reaktion', 'ausschuss', 'fehler', 'response']

function shouldInvertGoodness(label: string): boolean {
  const lower = label.toLowerCase()
  return INVERT_MARKERS.some((m) => lower.includes(m))
}

const SPARKLINE_POINTS = 8

/**
 * Deterministic demo trend for a KPI card sparkline.
 *
 * The KPI payload has no time-series (the backend KPI service is not yet
 * available — see backend-gaps.md), so we synthesise a stable series seeded
 * from the KPI id. The trend direction follows `change_percent` so the line
 * visually agrees with the change badge. Same id → same series across renders.
 */
function buildSparklineSeries(kpi: DashboardKPI): { value: number }[] {
  let seed = 0
  for (let i = 0; i < kpi.id.length; i++) seed = (seed * 31 + kpi.id.charCodeAt(i)) >>> 0

  const change = kpi.change_percent ?? 2
  const drift = change / 100 // per-step trend share
  const points: { value: number }[] = []
  let value = 1
  for (let i = 0; i < SPARKLINE_POINTS; i++) {
    seed = (seed * 1664525 + 1013904223) >>> 0
    const noise = (seed / 0xffffffff - 0.5) * 0.12
    value = value * (1 + drift / SPARKLINE_POINTS) + noise
    points.push({ value: Math.max(0.05, value) })
  }
  return points
}

export function DashboardGrid({
  kpis,
  definitions,
  moduleFilter,
  onModuleFilterChange,
  moduleOptions,
  isLoading,
}: DashboardGridProps) {
  const { t } = useTranslation()
  const theme = useChartTheme()
  const reducedMotion = usePrefersReducedMotion()
  const [drilldownKpiId, setDrilldownKpiId] = useState<string | null>(null)

  const runReportMutation = useRunReport()
  const heroDef = useMemo(
    () => definitions.find((d) => d.module === 'finanzen' && d.kind === 'system'),
    [definitions],
  )

  const drilldownKpi = drilldownKpiId
    ? kpis.find((k) => k.id === drilldownKpiId) ?? null
    : null

  const heroSeries = useMemo(() => {
    const result = runReportMutation.data?.result as ReportResult | undefined
    return result?.series?.[0]?.data ?? []
  }, [runReportMutation.data])

  const handleHeroRun = () => {
    if (!heroDef || runReportMutation.isPending) return
    runReportMutation.mutate({ definitionId: heroDef.id })
  }

  const handleKpiClick = (kpi: DashboardKPI) => {
    setDrilldownKpiId((prev) => (prev === kpi.id ? null : kpi.id))
  }

  return (
    <>
      {/* Module filter */}
      <div className="mb-5 flex items-center gap-2">
        <Filter className="h-4 w-4 text-muted-foreground" />
        <select
          value={moduleFilter}
          onChange={(e) => onModuleFilterChange(e.target.value)}
          className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
        >
          <option value="all">{t('berichte.dashboard.alleModule')}</option>
          {moduleOptions.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
        {moduleFilter !== 'all' && (
          <button
            onClick={() => onModuleFilterChange('all')}
            className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary"
          >
            {t('berichte.dashboard.filterZuruecksetzen')}
          </button>
        )}
      </div>

      {/* KPI Cards */}
      <div
        className="mb-8 grid grid-cols-1 gap-4 animate-fade-up sm:grid-cols-2 lg:grid-cols-3"
        style={{ animationDelay: '100ms' }}
      >
        {!isLoading && kpis.length === 0 && (
          <div className="col-span-full">
            <EmptyState illustration={<EmptyReports />} title={t('berichte.dashboard.noKpis')} />
          </div>
        )}
        {kpis.map((kpi) => (
          <KPICard
            key={kpi.id}
            kpi={kpi}
            active={drilldownKpiId === kpi.id}
            onClick={() => handleKpiClick(kpi)}
            invertGoodness={shouldInvertGoodness(kpi.label)}
            sparklineData={buildSparklineSeries(kpi)}
          />
        ))}
      </div>

      {/* Drilldown — placeholder until run-per-KPI wiring lands */}
      {drilldownKpi && (
        <div className="mb-8 rounded-lg border border-primary/20 bg-card p-5">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-medium text-foreground">
              {drilldownKpi.label} — {t('berichte.dashboard.details')}
            </h3>
            <button
              onClick={() => setDrilldownKpiId(null)}
              className="rounded-lg p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="text-xs text-muted-foreground">
            {t('berichte.dashboard.drilldownPending', {
              defaultValue:
                'Drilldown-Details verfügbar sobald die Report-Definition für diese KPI angebunden ist.',
            })}
          </p>
        </div>
      )}

      {/* Hero chart */}
      <div className="grid grid-cols-1 gap-6 animate-fade-up lg:grid-cols-2" style={{ animationDelay: '200ms' }}>
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="mb-5 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-medium text-foreground">{t('berichte.chart.umsatzverlauf')}</h3>
              <p className="text-xs text-muted-foreground">{t('berichte.chart.umsatz6Monate')}</p>
            </div>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-light">
              <TrendingUp className="h-4 w-4 text-primary" />
            </div>
          </div>
          {heroSeries.length === 0 ? (
            <div className="flex h-48 flex-col items-center justify-center gap-2">
              <p className="text-xs text-muted-foreground">
                {t('berichte.chart.noData', { defaultValue: 'Noch keine Daten geladen.' })}
              </p>
              <button
                onClick={handleHeroRun}
                disabled={!heroDef || runReportMutation.isPending}
                className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary disabled:opacity-60"
              >
                {runReportMutation.isPending
                  ? t('berichte.chart.laedt', { defaultValue: 'Lade...' })
                  : t('berichte.chart.laden', { defaultValue: 'Hero-Report laden' })}
              </button>
            </div>
          ) : (
            <div className="h-48">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={heroSeries} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
                  <CartesianGrid stroke={theme.grid} vertical={false} />
                  <XAxis dataKey="label" stroke={theme.muted} tick={{ fontSize: 10 }} />
                  <YAxis stroke={theme.muted} tick={{ fontSize: 10 }} width={32} />
                  <Tooltip
                    contentStyle={{
                      background: 'var(--card)',
                      border: `1px solid ${theme.grid}`,
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                  />
                  <Line
                    type="monotone"
                    dataKey="value"
                    stroke={theme.primary}
                    strokeWidth={2}
                    dot={{ r: 3, fill: theme.primary }}
                    activeDot={{ r: 5 }}
                    isAnimationActive={!reducedMotion}
                  />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </div>

        {/* Secondary chart: ticket priority-like distribution from first non-hero series */}
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="mb-5 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-medium text-foreground">
                {t('berichte.chart.ticketsPrioritaet')}
              </h3>
              <p className="text-xs text-muted-foreground">{t('berichte.chart.offeneTickets')}</p>
            </div>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-warning-light">
              <BarChart3 className="h-4 w-4 text-warning" />
            </div>
          </div>
          {heroSeries.length === 0 ? (
            <div className="flex h-48 items-center justify-center">
              <p className="text-xs text-muted-foreground">
                {t('berichte.chart.noData', { defaultValue: 'Noch keine Daten geladen.' })}
              </p>
            </div>
          ) : (
            <div className="h-48">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={heroSeries} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
                  <CartesianGrid stroke={theme.grid} vertical={false} />
                  <XAxis dataKey="label" stroke={theme.muted} tick={{ fontSize: 10 }} />
                  <YAxis stroke={theme.muted} tick={{ fontSize: 10 }} width={32} />
                  <Tooltip
                    contentStyle={{
                      background: 'var(--card)',
                      border: `1px solid ${theme.grid}`,
                      borderRadius: 8,
                      fontSize: 12,
                    }}
                  />
                  <Bar
                    dataKey="value"
                    fill={theme.accent1}
                    radius={[4, 4, 0, 0]}
                    isAnimationActive={!reducedMotion}
                  />
                </BarChart>
              </ResponsiveContainer>
            </div>
          )}
        </div>
      </div>
    </>
  )
}
