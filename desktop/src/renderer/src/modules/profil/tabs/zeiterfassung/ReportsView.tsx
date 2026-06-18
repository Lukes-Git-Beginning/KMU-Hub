import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import {
  useTimeBalance, useTimeAnalytics, useWorkTimeEntries, useTimeCategories,
} from '@/api/hooks/hr-hooks'
import { adaptWorkTimeEntryToFE } from '@/api/hr-client'
import { formatMinutes, formatHoursDecimal, getWeekDates, dateToStr } from './time-utils'
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

  const { data: balance } = useTimeBalance()
  const { data: analyticsMonth } = useTimeAnalytics('month')
  const { data: categories = [] } = useTimeCategories()
  const { data: entriesRaw } = useWorkTimeEntries({ limit: 500 })

  const entries = useMemo(
    () => (entriesRaw?.entries ?? []).map(adaptWorkTimeEntryToFE),
    [entriesRaw],
  )

  // Hours per category (from entries)
  const categoryStats = useMemo(() => {
    const stats: Record<string, number> = {}
    for (const e of entries) {
      if (e.categoryId) {
        stats[e.categoryId] = (stats[e.categoryId] || 0) + e.durationMinutes
      }
    }
    return categories
      .map((cat) => ({ ...cat, minutes: stats[cat.id] || 0 }))
      .filter((c) => c.minutes > 0)
      .sort((a, b) => b.minutes - a.minutes)
  }, [entries, categories])

  const maxCategoryMinutes = Math.max(1, ...categoryStats.map((c) => c.minutes))
  const totalMinutes = categoryStats.reduce((s, c) => s + c.minutes, 0)

  // Weekly trend (last 4 weeks) — from entries
  const weeklyTrend = useMemo(() => {
    return Array.from({ length: 4 }, (_, i) => {
      const weekDates = getWeekDates(-3 + i)
      const dateStrs = weekDates.map(dateToStr)
      const mins = entries
        .filter((e) => dateStrs.includes(e.date))
        .reduce((s, e) => s + e.durationMinutes, 0)
      const weekNum = -3 + i
      return {
        label: weekNum === 0
          ? t('profil.zeiterfassung.reports.thisWeek')
          : weekNum === -1
            ? t('profil.zeiterfassung.reports.lastWeek')
            : `${t('profil.zeiterfassung.reports.calendarWeek')} ${weekNum + getISOWeek()}`,
        minutes: mins,
      }
    })
  }, [entries, t])

  const weeklyTarget = 40 * 60 // 40h in minutes
  const maxWeekMinutes = Math.max(1, ...weeklyTrend.map((w) => w.minutes))

  // Overtime balance from API
  const overtime = balance?.balanceMinutes ?? 0

  // Summary totals from month analytics
  const periodStart = analyticsMonth?.periodStart ?? ''
  const monthTotalMinutes = analyticsMonth?.totalNetMinutes ?? totalMinutes

  return (
    <div className="p-6 space-y-6 max-w-3xl mx-auto">
      {/* Hours per Category */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold text-foreground mb-4">{t('profil.zeiterfassung.reports.hoursPerCategory')}</h3>
        <div className="space-y-3">
          {categoryStats.map((cat) => {
            const percent = Math.round((cat.minutes / maxCategoryMinutes) * 100)
            const share = Math.round((cat.minutes / totalMinutes) * 100)
            return (
              <div key={cat.id}>
                <div className="flex items-center justify-between mb-1">
                  <span className="flex items-center gap-2 text-sm">
                    <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: cat.color }} />
                    <span className="text-foreground">{cat.name}</span>
                  </span>
                  <span className="text-sm text-muted-foreground tabular-nums">
                    {formatHoursDecimal(cat.minutes)}h ({share}%)
                  </span>
                </div>
                <div className="w-full h-3 rounded-full bg-secondary overflow-hidden">
                  <div
                    className="h-full rounded-full transition-all"
                    style={{ width: `${percent}%`, backgroundColor: cat.color }}
                  />
                </div>
              </div>
            )
          })}
          {categoryStats.length === 0 && (
            <p className="text-sm text-muted-foreground py-4 text-center">{t('profil.zeiterfassung.noEntries')}</p>
          )}
        </div>
      </div>

      {/* Weekly Trend */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold text-foreground mb-4">{t('profil.zeiterfassung.reports.weeklyTrend')}</h3>
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
                  {/* Target line */}
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

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-2">{t('profil.zeiterfassung.reports.targetActual')}</h3>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.reports.worked')}</span>
              <span className="font-medium text-foreground">{formatMinutes(monthTotalMinutes)}</span>
            </div>
            {periodStart && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">{t('profil.zeiterfassung.overview.target')}</span>
                <span className="font-medium text-foreground">{formatHoursDecimal(weeklyTarget)}h/{t('profil.zeiterfassung.reports.week')}</span>
              </div>
            )}
            <div className="border-t border-border pt-2 flex justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.reports.difference')}</span>
              <span className={cn(
                'font-semibold',
                overtime >= 0 ? 'text-success' : 'text-destructive',
              )}>
                {overtime >= 0 ? '+' : ''}{formatMinutes(overtime)}
              </span>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-2">{t('profil.zeiterfassung.reports.overtimeBalance')}</h3>
          <div className="flex items-end gap-2 mb-2">
            <span className={cn(
              'text-3xl font-bold',
              overtime >= 0 ? 'text-success' : 'text-destructive',
            )}>
              {overtime >= 0 ? '+' : ''}{formatHoursDecimal(overtime)}h
            </span>
          </div>
          <p className="text-xs text-muted-foreground">
            {overtime >= 0
              ? t('profil.zeiterfassung.reports.workedMore')
              : t('profil.zeiterfassung.reports.workedLess')}
          </p>
          {balance?.periodStart && (
            <p className="text-xs text-muted-foreground mt-1">
              {t('profil.zeiterfassung.overview.target')}: {new Date(balance.periodStart).toLocaleDateString('de-DE')}
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
