import { useEffect, useMemo, useState } from 'react'
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
import { ArrowDownRight, ArrowUpRight, BarChart3, Filter, Loader2, TrendingUp } from 'lucide-react'
import { EmptyState } from '@/components/shared/EmptyState'
import { EmptyReports } from '@/components/shared/illustrations'
import { DetailModal } from '@/components/shared/DetailModal'
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

/** Parse a German-formatted KPI value string ("84.532" / "2,4") to a number. */
function parseKpiValue(v: string): number {
  const n = parseFloat(v.replace(/\./g, '').replace(',', '.'))
  return Number.isFinite(n) ? n : 0
}

const DRILLDOWN_POINTS = 6

/**
 * Demo time-series for the drilldown modal: ends at the KPI's current value and
 * walks back `change_percent` over the last few months with stable seeded noise.
 */
function buildDrilldownSeries(kpi: DashboardKPI, labels: string[]): { label: string; value: number }[] {
  const end = parseKpiValue(kpi.value)
  const change = kpi.change_percent ?? 0
  const start = change !== 0 ? end / (1 + change / 100) : end * 0.94
  let seed = 7
  for (let i = 0; i < kpi.id.length; i++) seed = (seed * 31 + kpi.id.charCodeAt(i)) >>> 0
  const span = labels.length - 1 || 1
  return labels.map((label, i) => {
    seed = (seed * 1664525 + 1013904223) >>> 0
    const noise = (seed / 0xffffffff - 0.5) * 0.06
    const lin = start + ((end - start) * i) / span
    const raw = i === labels.length - 1 ? end : Math.max(0, lin * (1 + noise))
    // Whole numbers for large values (€, counts); one decimal for small ones (%, hours).
    const value = raw >= 100 ? Math.round(raw) : Math.round(raw * 10) / 10
    return { label, value }
  })
}

export function DashboardGrid({
  kpis,
  definitions,
  moduleFilter,
  onModuleFilterChange,
  moduleOptions,
  isLoading,
}: DashboardGridProps) {
  const { t, i18n } = useTranslation()
  const theme = useChartTheme()
  const reducedMotion = usePrefersReducedMotion()
  const [drilldownKpiId, setDrilldownKpiId] = useState<string | null>(null)

  const heroRun = useRunReport()
  const ticketRun = useRunReport()

  const heroDef = useMemo(
    () => definitions.find((d) => (d.query_config as { kind?: string })?.kind === 'revenue'),
    [definitions],
  )
  const ticketDef = useMemo(
    () => definitions.find((d) => (d.query_config as { kind?: string })?.kind === 'tickets'),
    [definitions],
  )

  // Auto-load both hero charts once their definition resolves (no manual click).
  useEffect(() => {
    if (heroDef && !heroRun.data && !heroRun.isPending) heroRun.mutate({ definitionId: heroDef.id })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [heroDef?.id])
  useEffect(() => {
    if (ticketDef && !ticketRun.data && !ticketRun.isPending)
      ticketRun.mutate({ definitionId: ticketDef.id })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ticketDef?.id])

  const heroSeries = useMemo(() => {
    const result = heroRun.data?.result as ReportResult | undefined
    return result?.series?.[0]?.data ?? []
  }, [heroRun.data])
  const ticketSeries = useMemo(() => {
    const result = ticketRun.data?.result as ReportResult | undefined
    return result?.series?.[0]?.data ?? []
  }, [ticketRun.data])

  const drilldownKpi = drilldownKpiId ? kpis.find((k) => k.id === drilldownKpiId) ?? null : null

  const monthLabels = useMemo(() => {
    const now = new Date()
    return Array.from({ length: DRILLDOWN_POINTS }, (_, i) => {
      const d = new Date(now.getFullYear(), now.getMonth() - (DRILLDOWN_POINTS - 1 - i), 1)
      return d.toLocaleDateString(i18n.language, { month: 'short' })
    })
  }, [i18n.language])

  const drilldownData = useMemo(
    () => (drilldownKpi ? buildDrilldownSeries(drilldownKpi, monthLabels) : []),
    [drilldownKpi, monthLabels],
  )

  const handleKpiClick = (kpi: DashboardKPI) => {
    setDrilldownKpiId((prev) => (prev === kpi.id ? null : kpi.id))
  }

  const moduleName = (id: string) => moduleOptions.find((m) => m.id === id)?.name ?? id

  const renderChartFallback = (pending: boolean) => (
    <div className="flex h-48 items-center justify-center">
      {pending ? (
        <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
      ) : (
        <p className="text-xs text-muted-foreground">
          {t('berichte.chart.noData', { defaultValue: 'Noch keine Daten geladen.' })}
        </p>
      )}
    </div>
  )

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
            sparklineData={buildDrilldownSeries(kpi, monthLabels)}
          />
        ))}
      </div>

      {/* Hero charts */}
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
            renderChartFallback(heroRun.isPending)
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

        {/* Secondary chart: open tickets per month (own definition) */}
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
          {ticketSeries.length === 0 ? (
            renderChartFallback(ticketRun.isPending)
          ) : (
            <div className="h-48">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={ticketSeries} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
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

      {/* Drilldown modal */}
      <DetailModal
        open={!!drilldownKpi}
        onClose={() => setDrilldownKpiId(null)}
        title={drilldownKpi?.label}
        subtitle={drilldownKpi ? moduleName(String(drilldownKpi.module_id)) : undefined}
        badge={
          drilldownKpi?.change_percent != null ? (
            <span
              className={`ml-1 inline-flex items-center gap-0.5 rounded-full px-2 py-0.5 text-xs font-medium ${
                (drilldownKpi.change_percent >= 0) !== shouldInvertGoodness(drilldownKpi.label)
                  ? 'bg-success-light text-success'
                  : 'bg-error-light text-error'
              }`}
            >
              {drilldownKpi.change_percent >= 0 ? (
                <ArrowUpRight className="h-3 w-3" />
              ) : (
                <ArrowDownRight className="h-3 w-3" />
              )}
              {drilldownKpi.change_percent > 0 ? '+' : ''}
              {drilldownKpi.change_percent}%
            </span>
          ) : null
        }
      >
        {drilldownKpi && (
          <div className="space-y-5">
            {/* Current value headline */}
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-semibold text-foreground">{drilldownKpi.value}</span>
              {drilldownKpi.unit && (
                <span className="text-sm text-muted-foreground">{drilldownKpi.unit}</span>
              )}
            </div>

            {/* Mini time-series */}
            <div>
              <p className="mb-2 text-xs font-medium text-muted-foreground">
                {t('berichte.drilldown.verlauf', { defaultValue: 'Verlauf (Demo)' })}
              </p>
              <div className="h-44 rounded-lg border border-border bg-card p-3">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={drilldownData} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
                    <CartesianGrid stroke={theme.grid} vertical={false} />
                    <XAxis dataKey="label" stroke={theme.muted} tick={{ fontSize: 10 }} />
                    <YAxis stroke={theme.muted} tick={{ fontSize: 10 }} width={40} />
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
                      isAnimationActive={!reducedMotion}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* Key facts table */}
            <dl className="grid grid-cols-2 gap-px overflow-hidden rounded-lg border border-border bg-border text-sm">
              <div className="bg-card px-4 py-3">
                <dt className="text-xs text-muted-foreground">
                  {t('berichte.chart.aktuell', { defaultValue: 'Aktuell' })}
                </dt>
                <dd className="mt-0.5 font-medium text-foreground">
                  {drilldownKpi.value} {drilldownKpi.unit}
                </dd>
              </div>
              <div className="bg-card px-4 py-3">
                <dt className="text-xs text-muted-foreground">{t('berichte.dashboard.vorMonat')}</dt>
                <dd className="mt-0.5 font-medium text-foreground">
                  {drilldownKpi.change_percent != null
                    ? `${drilldownKpi.change_percent > 0 ? '+' : ''}${drilldownKpi.change_percent}%`
                    : '—'}
                </dd>
              </div>
              <div className="bg-card px-4 py-3">
                <dt className="text-xs text-muted-foreground">
                  {t('berichte.drilldown.modul', { defaultValue: 'Modul' })}
                </dt>
                <dd className="mt-0.5 font-medium text-foreground">
                  {moduleName(String(drilldownKpi.module_id))}
                </dd>
              </div>
              <div className="bg-card px-4 py-3">
                <dt className="text-xs text-muted-foreground">
                  {t('berichte.drilldown.zeitraum', { defaultValue: 'Zeitraum' })}
                </dt>
                <dd className="mt-0.5 font-medium text-foreground">
                  {monthLabels[0]} – {monthLabels[monthLabels.length - 1]}
                </dd>
              </div>
            </dl>

            <p className="text-[10px] text-muted-foreground/60">
              {t('berichte.drilldown.hinweis', {
                defaultValue: 'Demo-Werte — finale BI-Anbindung folgt.',
              })}
            </p>
          </div>
        )}
      </DetailModal>
    </>
  )
}
