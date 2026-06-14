/**
 * Auswertungen — time-tracking analytics (clockodo-style reports).
 *
 * Range toggle (week/month), KPI cards, and three charts: hours per day,
 * hours by project, and billable vs. non-billable split. Recharts styled
 * via useChartTheme (no rainbow defaults). Read-only; mock-first HR API.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts'
import { Clock, Receipt, TrendingUp, CalendarRange, Loader2 } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTimeAnalytics } from '@/api/hooks/hr-hooks'
import { useChartTheme } from '@/modules/berichte/utils/chartTheme'
import { formatWorkMinutes } from '@/lib/worktime'
import type { TimeAnalyticsRange } from '@/api/hr-types'

const tooltipStyle = {
  background: 'var(--card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  fontSize: 12,
  color: 'var(--foreground)',
}

function hours(min: number): number {
  return Math.round((min / 60) * 10) / 10
}

export function AuswertungenView() {
  const { t } = useTranslation()
  const theme = useChartTheme()
  const [range, setRange] = useState<TimeAnalyticsRange>('week')
  const { data, isLoading } = useTimeAnalytics(range)

  if (isLoading || !data) {
    return (
      <div className="flex items-center justify-center py-16">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
      </div>
    )
  }

  const billablePct = data.totalNetMinutes > 0
    ? Math.round((data.billableMinutes / data.totalNetMinutes) * 100)
    : 0
  const avgPerDay = data.workedDays > 0 ? Math.round(data.totalNetMinutes / data.workedDays) : 0

  const trend = data.dayTrend.map((d) => {
    const date = new Date(d.date)
    const label = range === 'week'
      ? date.toLocaleDateString('de-DE', { weekday: 'short' })
      : String(date.getDate())
    return {
      label,
      abrechenbar: hours(d.billableMinutes),
      rest: hours(Math.max(0, d.netMinutes - d.billableMinutes)),
    }
  })

  const projects = [...data.byProject]
    .filter((p) => p.minutes > 0)
    .sort((a, b) => b.minutes - a.minutes)

  const billableSplit = [
    { name: t('zeiterfassung.analytics.billable'), value: data.billableMinutes, color: theme.primary },
    { name: t('zeiterfassung.analytics.nonBillable'), value: data.nonBillableMinutes, color: theme.muted },
  ]

  const KPIS = [
    { icon: Clock, label: t('zeiterfassung.analytics.total'), value: formatWorkMinutes(data.totalNetMinutes), tone: 'text-foreground' },
    { icon: Receipt, label: t('zeiterfassung.analytics.billable'), value: `${formatWorkMinutes(data.billableMinutes)} · ${billablePct}%`, tone: 'text-success' },
    { icon: TrendingUp, label: t('zeiterfassung.analytics.overtime'), value: (data.overtimeMinutes > 0 ? '+' : '') + formatWorkMinutes(data.overtimeMinutes), tone: data.overtimeMinutes > 0 ? 'text-warning' : 'text-foreground' },
    { icon: CalendarRange, label: t('zeiterfassung.analytics.avgPerDay'), value: formatWorkMinutes(avgPerDay), tone: 'text-foreground' },
  ]

  return (
    <div className="space-y-6">
      {/* Range toggle */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-foreground">{t('zeiterfassung.analytics.title')}</h3>
        <div className="flex rounded-lg border border-border p-0.5">
          {(['week', 'month'] as const).map((r) => (
            <button
              key={r}
              onClick={() => setRange(r)}
              className={cn(
                'rounded-md px-3 py-1 text-xs font-medium transition-colors',
                range === r ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground',
              )}
            >
              {t(`zeiterfassung.analytics.range.${r}`)}
            </button>
          ))}
        </div>
      </div>

      {/* KPI cards */}
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {KPIS.map((kpi) => {
          const Icon = kpi.icon
          return (
            <div key={kpi.label} className="rounded-xl border border-border bg-card p-4">
              <div className="mb-2 flex items-center gap-2">
                <Icon className="h-4 w-4 text-muted-foreground" />
                <span className="text-xs text-muted-foreground">{kpi.label}</span>
              </div>
              <p className={cn('text-lg font-semibold tabular-nums', kpi.tone)}>{kpi.value}</p>
            </div>
          )
        })}
      </div>

      {/* Hours per day */}
      <div className="rounded-xl border border-border bg-card p-4">
        <h4 className="mb-4 text-sm font-medium text-foreground">{t('zeiterfassung.analytics.hoursPerDay')}</h4>
        <ResponsiveContainer width="100%" height={220}>
          <BarChart data={trend} margin={{ top: 4, right: 8, left: -16, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke={theme.grid} vertical={false} />
            <XAxis dataKey="label" tick={{ fontSize: 11, fill: theme.muted }} tickLine={false} axisLine={false} />
            <YAxis tick={{ fontSize: 11, fill: theme.muted }} tickLine={false} axisLine={false} unit="h" />
            <Tooltip contentStyle={tooltipStyle} cursor={{ fill: 'var(--secondary)', opacity: 0.4 }} />
            <Bar dataKey="abrechenbar" stackId="a" fill={theme.primary} radius={[0, 0, 0, 0]} name={t('zeiterfassung.analytics.billable')} />
            <Bar dataKey="rest" stackId="a" fill={theme.muted} fillOpacity={0.3} radius={[3, 3, 0, 0]} name={t('zeiterfassung.analytics.nonBillable')} />
          </BarChart>
        </ResponsiveContainer>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* By project */}
        <div className="rounded-xl border border-border bg-card p-4">
          <h4 className="mb-4 text-sm font-medium text-foreground">{t('zeiterfassung.analytics.byProject')}</h4>
          <div className="space-y-3">
            {projects.map((p) => {
              const max = projects[0]?.minutes || 1
              return (
                <div key={p.projectId} className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="flex items-center gap-2 truncate">
                      <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: p.color }} />
                      <span className="font-medium text-foreground">{p.name}</span>
                      <span className="truncate text-muted-foreground">{p.customerName}</span>
                    </span>
                    <span className="shrink-0 tabular-nums text-muted-foreground">{formatWorkMinutes(p.minutes)}</span>
                  </div>
                  <div className="h-2 w-full overflow-hidden rounded-full bg-secondary">
                    <div className="h-full rounded-full" style={{ width: `${(p.minutes / max) * 100}%`, backgroundColor: p.color }} />
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Billable split */}
        <div className="rounded-xl border border-border bg-card p-4">
          <h4 className="mb-4 text-sm font-medium text-foreground">{t('zeiterfassung.analytics.billableSplit')}</h4>
          <div className="flex items-center gap-4">
            <ResponsiveContainer width="50%" height={160}>
              <PieChart>
                <Pie data={billableSplit} dataKey="value" nameKey="name" innerRadius={42} outerRadius={64} paddingAngle={2} stroke="none">
                  {billableSplit.map((s) => (
                    <Cell key={s.name} fill={s.color} />
                  ))}
                </Pie>
                <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => formatWorkMinutes(v)} />
              </PieChart>
            </ResponsiveContainer>
            <div className="flex-1 space-y-2">
              {billableSplit.map((s) => (
                <div key={s.name} className="flex items-center justify-between text-xs">
                  <span className="flex items-center gap-2">
                    <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: s.color }} />
                    <span className="text-foreground">{s.name}</span>
                  </span>
                  <span className="tabular-nums text-muted-foreground">{formatWorkMinutes(s.value)}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
