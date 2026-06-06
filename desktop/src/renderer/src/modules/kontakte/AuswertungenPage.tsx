/**
 * Auswertungen — CRM analytics for the Kontakte hub.
 *
 * Aggregates deals, leads, activities and contacts into KPI cards and charts:
 * pipeline funnel, win/loss conversion, activities by type, lead sources.
 * Read-only; data comes from the existing hooks (mock-first, real-API-ready).
 */
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts'
import { TrendingUp, Banknote, Target, Inbox, CalendarClock } from 'lucide-react'
import { usePipelineStages } from '@/api/hooks/usePipelineStages'
import { useDeals } from '@/api/hooks/useDeals'
import { useLeads } from '@/api/hooks/useLeads'
import { useActivities } from '@/api/hooks/useActivities'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/shared'
import { useChartTheme, categoricalPalette } from '@/modules/berichte/utils/chartTheme'
import { activityTypeLabel } from '@/modules/crm/activities/activityUtils'

type Stage = NonNullable<ReturnType<typeof usePipelineStages>['data']>['stages'][number]

function stageIsWon(s: Stage): boolean {
  const r = s as Record<string, unknown>
  return Boolean(r.isWon ?? r.is_won)
}
function stageIsLost(s: Stage): boolean {
  const r = s as Record<string, unknown>
  return Boolean(r.isLost ?? r.is_lost)
}
function fmtCurrency(v: number): string {
  return new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR', minimumFractionDigits: 0, maximumFractionDigits: 0 }).format(v)
}

const tooltipStyle = { background: 'var(--card)', border: '1px solid var(--border)', borderRadius: 8, fontSize: 12, color: 'var(--foreground)' }

export default function AuswertungenPage() {
  const { t } = useTranslation()
  const theme = useChartTheme()
  const palette = categoricalPalette(theme)

  const { data: stagesData, isLoading: sl } = usePipelineStages()
  const { data: dealsData, isLoading: dl } = useDeals({ page_size: 200 })
  const { data: leads, isLoading: ll } = useLeads()
  const { data: actData, isLoading: al } = useActivities({ page_size: 200 })
  const isLoading = sl || dl || ll || al

  const stages = useMemo(() => stagesData?.stages ?? [], [stagesData])
  const deals = useMemo(() => dealsData?.deals ?? [], [dealsData])
  const allLeads = useMemo(() => leads ?? [], [leads])
  const activities = useMemo(() => (actData?.activities ?? []) as Array<Record<string, unknown>>, [actData])

  const metrics = useMemo(() => {
    const openStages = stages.filter((s) => !stageIsWon(s) && !stageIsLost(s))
    const openStageIds = new Set(openStages.map((s) => s.id))
    const wonStage = stages.find(stageIsWon)
    const lostStage = stages.find(stageIsLost)

    const dealsByStage = new Map<string, typeof deals>()
    for (const s of stages) dealsByStage.set(s.id ?? '', [])
    for (const d of deals) dealsByStage.get(d.stageId ?? '')?.push(d)

    const openDeals = deals.filter((d) => openStageIds.has(d.stageId ?? ''))
    const openValue = openDeals.reduce((sum, d) => sum + (d.value ?? 0), 0)
    const probByStage = new Map(stages.map((s) => [s.id ?? '', s.probability ?? 0]))
    const weighted = openDeals.reduce((sum, d) => sum + ((d.value ?? 0) * (probByStage.get(d.stageId ?? '') ?? 0)) / 100, 0)

    const wonCount = (dealsByStage.get(wonStage?.id ?? '') ?? []).length
    const lostCount = (dealsByStage.get(lostStage?.id ?? '') ?? []).length
    const winRate = wonCount + lostCount > 0 ? Math.round((wonCount / (wonCount + lostCount)) * 100) : 0

    // Funnel: open stages by total value (sorted by stage order/probability desc → wide to narrow)
    const funnel = openStages
      .map((s) => {
        const list = dealsByStage.get(s.id ?? '') ?? []
        return { name: s.name ?? '', value: list.reduce((sum, d) => sum + (d.value ?? 0), 0), count: list.length, color: s.color || theme.primary }
      })
      .sort((a, b) => b.value - a.value)

    const openLeads = allLeads.filter((l) => l.status === 'new' || l.status === 'contacted').length
    // Lead sources
    const sourceMap = new Map<string, number>()
    for (const l of allLeads) sourceMap.set(l.source, (sourceMap.get(l.source) ?? 0) + 1)
    const leadSources = [...sourceMap.entries()].map(([source, count]) => ({ name: t(`leads.source.${source}`), count }))

    // Activities by type + open/overdue
    const typeMap = new Map<string, number>()
    let dueActivities = 0
    const today = new Date(); today.setHours(0, 0, 0, 0)
    for (const a of activities) {
      const ty = (a.activity_type ?? a.type) as string
      typeMap.set(ty, (typeMap.get(ty) ?? 0) + 1)
      if (a.completed !== true && a.due_date) {
        const d = new Date(a.due_date as string)
        if (d.getTime() <= today.getTime() + 86_400_000) dueActivities++
      }
    }
    const activityTypes = [...typeMap.entries()].map(([ty, count]) => ({ name: t(activityTypeLabel(ty)), count }))

    return { openValue, weighted, winRate, wonCount, lostCount, funnel, openLeads, leadSources, activityTypes: activityTypes, dueActivities, totalDeals: deals.length }
  }, [stages, deals, allLeads, activities, theme.primary, t])

  if (isLoading) {
    return (
      <div className="space-y-4 p-6">
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-24 w-full" />)}
        </div>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-64 w-full" />)}
        </div>
      </div>
    )
  }

  if (metrics.totalDeals === 0 && metrics.openLeads === 0) {
    return <EmptyState icon={TrendingUp} title={t('crm.reports.emptyTitle')} description={t('crm.reports.empty')} />
  }

  const kpis = [
    { label: t('crm.reports.kpi.pipelineValue'), value: fmtCurrency(metrics.openValue), icon: Banknote, tone: 'text-foreground' },
    { label: t('crm.reports.kpi.weightedForecast'), value: fmtCurrency(metrics.weighted), icon: TrendingUp, tone: 'text-primary' },
    { label: t('crm.reports.kpi.winRate'), value: `${metrics.winRate}%`, icon: Target, tone: 'text-success' },
    { label: t('crm.reports.kpi.openLeads'), value: String(metrics.openLeads), icon: Inbox, tone: 'text-foreground' },
    { label: t('crm.reports.kpi.dueActivities'), value: String(metrics.dueActivities), icon: CalendarClock, tone: metrics.dueActivities > 0 ? 'text-warning' : 'text-foreground' },
  ]

  return (
    <div className="space-y-6 p-6">
      {/* KPI cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-5">
        {kpis.map((k) => {
          const Icon = k.icon
          return (
            <div key={k.label} className="rounded-xl border border-border bg-card p-4">
              <div className="flex items-center justify-between">
                <p className="text-xs text-muted-foreground">{k.label}</p>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </div>
              <p className={`mt-1 text-2xl font-semibold ${k.tone}`}>{k.value}</p>
            </div>
          )
        })}
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* Pipeline funnel */}
        <div className="rounded-xl border border-border bg-card p-4">
          <h3 className="mb-3 text-sm font-semibold text-foreground">{t('crm.reports.funnel')}</h3>
          {metrics.funnel.length === 0 ? (
            <p className="py-8 text-center text-xs text-muted-foreground">{t('crm.reports.noData')}</p>
          ) : (
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={metrics.funnel} layout="vertical" margin={{ top: 4, right: 16, bottom: 4, left: 8 }}>
                <CartesianGrid stroke={theme.grid} horizontal={false} />
                <XAxis type="number" stroke={theme.muted} tick={{ fontSize: 10 }} tickFormatter={(v) => fmtCurrency(Number(v))} />
                <YAxis type="category" dataKey="name" stroke={theme.muted} tick={{ fontSize: 11 }} width={90} />
                <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => [fmtCurrency(v), t('crm.reports.value')]} />
                <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                  {metrics.funnel.map((d, i) => <Cell key={i} fill={d.color} />)}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Conversion / win-loss */}
        <div className="rounded-xl border border-border bg-card p-4">
          <h3 className="mb-3 text-sm font-semibold text-foreground">{t('crm.reports.conversion')}</h3>
          <div className="flex items-center gap-6">
            <ResponsiveContainer width="50%" height={200}>
              <PieChart>
                <Pie
                  data={[
                    { name: t('crm.deals.stage.won'), value: metrics.wonCount },
                    { name: t('crm.deals.stage.lost'), value: metrics.lostCount },
                  ]}
                  dataKey="value"
                  innerRadius={50}
                  outerRadius={80}
                  paddingAngle={2}
                >
                  <Cell fill={theme.primary} />
                  <Cell fill={theme.muted} />
                </Pie>
                <Tooltip contentStyle={tooltipStyle} />
              </PieChart>
            </ResponsiveContainer>
            <div className="space-y-2">
              <div>
                <p className="text-3xl font-semibold text-success">{metrics.winRate}%</p>
                <p className="text-xs text-muted-foreground">{t('crm.reports.kpi.winRate')}</p>
              </div>
              <p className="text-sm text-foreground">{t('crm.deals.stage.won')}: <span className="font-medium">{metrics.wonCount}</span></p>
              <p className="text-sm text-muted-foreground">{t('crm.deals.stage.lost')}: <span className="font-medium">{metrics.lostCount}</span></p>
            </div>
          </div>
        </div>

        {/* Activities by type */}
        <div className="rounded-xl border border-border bg-card p-4">
          <h3 className="mb-3 text-sm font-semibold text-foreground">{t('crm.reports.activitiesByType')}</h3>
          {metrics.activityTypes.length === 0 ? (
            <p className="py-8 text-center text-xs text-muted-foreground">{t('crm.reports.noData')}</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={metrics.activityTypes} margin={{ top: 4, right: 8, bottom: 4, left: 8 }}>
                <CartesianGrid stroke={theme.grid} vertical={false} />
                <XAxis dataKey="name" stroke={theme.muted} tick={{ fontSize: 11 }} />
                <YAxis stroke={theme.muted} tick={{ fontSize: 10 }} width={28} allowDecimals={false} />
                <Tooltip contentStyle={tooltipStyle} />
                <Bar dataKey="count" radius={[4, 4, 0, 0]} fill={theme.primary} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Lead sources */}
        <div className="rounded-xl border border-border bg-card p-4">
          <h3 className="mb-3 text-sm font-semibold text-foreground">{t('crm.reports.leadSources')}</h3>
          {metrics.leadSources.length === 0 ? (
            <p className="py-8 text-center text-xs text-muted-foreground">{t('crm.reports.noData')}</p>
          ) : (
            <ResponsiveContainer width="100%" height={220}>
              <PieChart>
                <Pie data={metrics.leadSources} dataKey="count" nameKey="name" outerRadius={80} label={(e) => `${e.name}: ${e.value}`} labelLine={false} fontSize={10}>
                  {metrics.leadSources.map((_, i) => <Cell key={i} fill={palette[i % palette.length]} />)}
                </Pie>
                <Tooltip contentStyle={tooltipStyle} />
              </PieChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>
    </div>
  )
}
