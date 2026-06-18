import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { useTimeAnalytics, useTimeBalance } from '@/api/hooks/hr-hooks'
import { formatMinutes, formatHoursDecimal } from './time-utils'
import ExportDialog from './ExportDialog'

function getISOWeek(): number {
  const now = new Date()
  const d = new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate()))
  const dayNum = d.getUTCDay() || 7
  d.setUTCDate(d.getUTCDate() + 4 - dayNum)
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1))
  return Math.ceil(((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7)
}

export default function ReportsView() {
  const { t } = useTranslation()
  const [showExportDialog, setShowExportDialog] = useState(false)

  const { data: analytics } = useTimeAnalytics('month')
  const { data: balance } = useTimeBalance()

  // Category/project breakdown from analytics
  const projectStats = analytics?.byProject ?? []
  const totalMinutes = analytics?.totalNetMinutes ?? 0
  const maxProjectMinutes = Math.max(1, ...projectStats.map((p) => p.minutes))
  const overtimeMinutes = balance?.balanceMinutes ?? analytics?.overtimeMinutes ?? 0

  // Weekly trend — derive from dayTrend (last 4 weeks, group by ISO week)
  const weeklyTrend = useMemo(() => {
    if (!analytics?.dayTrend) return []
    // Group day-trend entries into calendar weeks
    const byWeek = new Map<string, number>()
    for (const d of analytics.dayTrend) {
      const date = new Date(d.date)
      const day = date.getDay()
      const monday = new Date(date)
      monday.setDate(date.getDate() - (day === 0 ? 6 : day - 1))
      const key = monday.toISOString().split('T')[0]
      byWeek.set(key, (byWeek.get(key) ?? 0) + d.netMinutes)
    }
    return [...byWeek.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .slice(-4)
      .map(([weekStart, minutes], _, arr) => {
        const isThisWeek = weekStart === arr[arr.length - 1]?.[0]
        const isLastWeek = weekStart === arr[arr.length - 2]?.[0]
        const kw = getISOWeek()
        const label = isThisWeek
          ? t('profil.zeiterfassung.reports.thisWeek')
          : isLastWeek
            ? t('profil.zeiterfassung.reports.lastWeek')
            : `${t('profil.zeiterfassung.reports.calendarWeek')} ${kw}`
        return { label, minutes }
      })
  }, [analytics, t])

  const weeklyTarget = (balance?.targetWeeklyMinutes ?? 2400)
  const maxWeekMinutes = Math.max(1, ...weeklyTrend.map((w) => w.minutes))
  const totalTarget = analytics?.targetMinutes ?? 0

  return (
    <div className="p-6 space-y-6 max-w-3xl mx-auto">
      {/* Hours per Project */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold text-foreground mb-4">
          {t('profil.zeiterfassung.reports.hoursPerCategory')}
        </h3>
        {projectStats.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4 text-center">{t('profil.zeiterfassung.noEntries')}</p>
        ) : (
          <div className="space-y-3">
            {projectStats.map((p) => {
              const percent = Math.round((p.minutes / maxProjectMinutes) * 100)
              const share = totalMinutes > 0 ? Math.round((p.minutes / totalMinutes) * 100) : 0
              return (
                <div key={p.projectId}>
                  <div className="flex items-center justify-between mb-1">
                    <span className="flex items-center gap-2 text-sm">
                      <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: p.color }} />
                      <span className="text-foreground">{p.name}</span>
                      {p.customerName && (
                        <span className="text-xs text-muted-foreground">({p.customerName})</span>
                      )}
                    </span>
                    <span className="text-sm text-muted-foreground tabular-nums">
                      {formatHoursDecimal(p.minutes)}h ({share}%)
                    </span>
                  </div>
                  <div className="w-full h-3 rounded-full bg-secondary overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all"
                      style={{ width: `${percent}%`, backgroundColor: p.color }}
                    />
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Weekly Trend */}
      {weeklyTrend.length > 0 && (
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-4">
            {t('profil.zeiterfassung.reports.weeklyTrend')}
          </h3>
          <div className="flex items-end gap-4" style={{ height: 120 }}>
            {weeklyTrend.map((week, i) => {
              const barHeight = week.minutes > 0 ? Math.max(8, (week.minutes / maxWeekMinutes) * 100) : 0
              const isOverTarget = week.minutes >= weeklyTarget
              return (
                <div key={i} className="flex-1 flex flex-col items-center gap-1">
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {week.minutes > 0 ? formatHoursDecimal(week.minutes) + 'h' : '-'}
                  </span>
                  <div className="w-full relative" style={{ height: 100 }}>
                    <div
                      className="absolute left-0 right-0 border-t border-dashed border-muted-foreground/40"
                      style={{ bottom: (weeklyTarget / maxWeekMinutes) * 100 }}
                    />
                    <div
                      className={cn(
                        'absolute bottom-0 left-1 right-1 rounded-t-md transition-all',
                        isOverTarget ? 'bg-success' : 'bg-warning',
                      )}
                      style={{ height: barHeight }}
                    />
                  </div>
                  <span className="text-[10px] text-muted-foreground text-center">{week.label}</span>
                </div>
              )
            })}
          </div>
          <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
            <span className="h-px w-4 border-t border-dashed border-muted-foreground/40" />
            {t('profil.zeiterfassung.overview.target')}: {formatHoursDecimal(weeklyTarget)}h/{t('profil.zeiterfassung.reports.week')}
          </div>
        </div>
      )}

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-2">
            {t('profil.zeiterfassung.reports.targetActual')}
          </h3>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.reports.worked')}</span>
              <span className="font-medium text-foreground">{formatMinutes(totalMinutes)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">
                {t('profil.zeiterfassung.overview.target')} ({analytics?.workedDays ?? 0} {t('profil.zeiterfassung.overview.days')})
              </span>
              <span className="font-medium text-foreground">{formatMinutes(totalTarget)}</span>
            </div>
            <div className="border-t border-border pt-2 flex justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.reports.difference')}</span>
              <span className={cn('font-semibold', (totalMinutes - totalTarget) >= 0 ? 'text-success' : 'text-destructive')}>
                {(totalMinutes - totalTarget) >= 0 ? '+' : ''}{formatMinutes(totalMinutes - totalTarget)}
              </span>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-2">
            {t('profil.zeiterfassung.reports.overtimeBalance')}
          </h3>
          <div className="flex items-end gap-2 mb-2">
            <span className={cn('text-3xl font-bold', overtimeMinutes >= 0 ? 'text-success' : 'text-destructive')}>
              {overtimeMinutes >= 0 ? '+' : ''}{formatHoursDecimal(overtimeMinutes)}h
            </span>
          </div>
          <p className="text-xs text-muted-foreground">
            {overtimeMinutes >= 0
              ? t('profil.zeiterfassung.reports.workedMore')
              : t('profil.zeiterfassung.reports.workedLess')}
          </p>
          {balance?.periodStart && (
            <p className="text-xs text-muted-foreground mt-1">
              {t('profil.zeiterfassung.overview.since', { date: balance.periodStart })}
            </p>
          )}
        </div>
      </div>

      {/* Export */}
      <div className="flex gap-3">
        <Button
          variant="outline"
          size="sm"
          className="gap-2"
          onClick={() => setShowExportDialog(true)}
        >
          <Download className="h-4 w-4" />
          {t('profil.zeiterfassung.export.export')}
        </Button>
      </div>

      <ExportDialog open={showExportDialog} onOpenChange={setShowExportDialog} />
    </div>
  )
}
