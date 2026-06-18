import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import {
  useWeeklySummary,
  useWorkTimeEntries,
  useMyWeekStatus,
} from '@/api/hooks/hr-hooks'
import {
  getWeekDates, dateToStr, formatDateShort, getDateRangeLabel,
  formatMinutes, formatHoursDecimal, isToday,
} from './time-utils'
import ApprovalBanner from './ApprovalBanner'
import type { WorkTimeEntry } from '@/api/hr-types'

function getWeekStartStr(offset: number): string {
  const d = new Date()
  const day = d.getDay()
  const mondayOffset = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + mondayOffset + offset * 7)
  return d.toISOString().split('T')[0]
}

export default function WeekView() {
  const { t } = useTranslation()
  const [weekOffset, setWeekOffset] = useState(0)

  const weekStartStr = getWeekStartStr(weekOffset)
  const weekDates = useMemo(() => getWeekDates(weekOffset), [weekOffset])
  const dateStrings = useMemo(() => weekDates.map(dateToStr), [weekDates])

  const { data: weeklySummary } = useWeeklySummary(weekStartStr)
  const { data: entriesData } = useWorkTimeEntries({
    start_date: dateStrings[0],
    end_date: dateStrings[dateStrings.length - 1],
  })
  const allEntries = entriesData?.entries ?? []
  const { data: weekStatus } = useMyWeekStatus(weekStartStr)

  // 480 = 8h default daily; real target from HRSettings (follow-up)
  const dailyTarget = 480
  const weeklyTarget = dailyTarget * 5

  // Day totals from summary when available, otherwise from entries
  const dayTotals = useMemo<Record<string, number>>(() => {
    if (weeklySummary?.days) {
      const map: Record<string, number> = {}
      for (const d of weeklySummary.days) map[d.date] = d.netWorkMinutes
      return map
    }
    const map: Record<string, number> = {}
    for (const ds of dateStrings) map[ds] = 0
    for (const e of allEntries as WorkTimeEntry[]) {
      const date = e.clockIn?.slice(0, 10) ?? ''
      if (map[date] !== undefined) {
        map[date] += e.netWorkMinutes ?? 0
      }
    }
    return map
  }, [weeklySummary, allEntries, dateStrings])

  // Project distribution per day (for visual rows)
  const projectRows = useMemo(() => {
    const projects = new Map<string, Record<string, number>>()
    for (const e of allEntries as WorkTimeEntry[]) {
      const date = e.clockIn?.slice(0, 10) ?? ''
      if (!dateStrings.includes(date)) continue
      const key = e.projectName ?? e.activity ?? t('profil.zeiterfassung.unknown')
      if (!projects.has(key)) {
        projects.set(key, Object.fromEntries(dateStrings.map((d) => [d, 0])))
      }
      const row = projects.get(key)!
      row[date] = (row[date] ?? 0) + (e.netWorkMinutes ?? 0)
    }
    return [...projects.entries()]
      .map(([name, days]) => ({ name, days }))
      .filter((r) => Object.values(r.days).some((v) => v > 0))
  }, [allEntries, dateStrings, t])

  const weekTotal = weeklySummary?.netWorkMinutes ?? Object.values(dayTotals).reduce((s, v) => s + v, 0)
  const weekOvertime = weekTotal - weeklyTarget

  return (
    <div className="p-6 space-y-4">
      {/* Approval Banner */}
      <ApprovalBanner weekStart={weekStartStr} weekStatus={weekStatus} />

      {/* Navigation */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => setWeekOffset((w) => w - 1)}>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant={weekOffset === 0 ? 'default' : 'outline'}
            onClick={() => setWeekOffset(0)}
          >
            {t('profil.zeiterfassung.overview.thisWeek')}
          </Button>
          <Button size="sm" variant="outline" onClick={() => setWeekOffset((w) => w + 1)}>
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
        <span className="text-sm font-medium text-foreground">{getDateRangeLabel(weekDates)}</span>
      </div>

      {/* Timesheet Grid */}
      <div className="rounded-xl border border-border bg-card overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border">
              <th className="text-left px-4 py-3 text-muted-foreground font-medium w-40">
                {t('profil.zeiterfassung.manual.category')}
              </th>
              {weekDates.map((date, i) => (
                <th
                  key={i}
                  className={cn(
                    'text-center px-2 py-3 font-medium min-w-[80px]',
                    isToday(dateStrings[i]) ? 'text-primary bg-primary/5' : 'text-muted-foreground',
                  )}
                >
                  {formatDateShort(date)}
                </th>
              ))}
              <th className="text-right px-4 py-3 text-foreground font-semibold w-24">Total</th>
            </tr>
          </thead>
          <tbody>
            {projectRows.map((row) => (
              <tr key={row.name} className="border-b border-border/50 hover:bg-accent/30 transition-colors">
                <td className="px-4 py-2.5">
                  <span className="font-medium text-foreground">{row.name}</span>
                </td>
                {dateStrings.map((ds, i) => {
                  const mins = row.days[ds] ?? 0
                  return (
                    <td
                      key={i}
                      className={cn(
                        'text-center px-2 py-2.5 tabular-nums',
                        isToday(ds) && 'bg-primary/5',
                        mins > 0 ? 'text-foreground' : 'text-muted-foreground/40',
                      )}
                    >
                      {mins > 0 ? formatHoursDecimal(mins) : '-'}
                    </td>
                  )
                })}
                <td className="text-right px-4 py-2.5 font-medium text-foreground tabular-nums">
                  {formatHoursDecimal(Object.values(row.days).reduce((s, v) => s + v, 0))}
                </td>
              </tr>
            ))}

            {projectRows.length === 0 && (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground">
                  {t('profil.zeiterfassung.week.noEntries')}
                </td>
              </tr>
            )}
          </tbody>
          <tfoot>
            <tr className="border-t-2 border-border">
              <td className="px-4 py-2.5 font-semibold text-foreground">{t('profil.zeiterfassung.total')}</td>
              {dateStrings.map((ds, i) => (
                <td
                  key={i}
                  className={cn(
                    'text-center px-2 py-2.5 font-semibold tabular-nums',
                    isToday(ds) && 'bg-primary/5',
                    dayTotals[ds] > 0 ? 'text-foreground' : 'text-muted-foreground/40',
                  )}
                >
                  {dayTotals[ds] > 0 ? formatHoursDecimal(dayTotals[ds]) : '-'}
                </td>
              ))}
              <td className="text-right px-4 py-2.5 font-bold text-primary tabular-nums">
                {formatHoursDecimal(weekTotal)}
              </td>
            </tr>
            <tr>
              <td className="px-4 py-2 text-xs text-muted-foreground">{t('profil.zeiterfassung.overview.target')}</td>
              {dateStrings.map((_, i) => (
                <td key={i} className="text-center px-2 py-2 text-xs text-muted-foreground tabular-nums">
                  {i < 5 ? formatHoursDecimal(dailyTarget) : '-'}
                </td>
              ))}
              <td className="text-right px-4 py-2 text-xs text-muted-foreground tabular-nums">
                {formatHoursDecimal(weeklyTarget)}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>

      {/* Week Summary */}
      <div className="flex items-center gap-4 text-sm">
        <span className="text-muted-foreground">{t('profil.zeiterfassung.week.weekBalance')}:</span>
        <span className={cn('font-semibold', weekOvertime >= 0 ? 'text-success' : 'text-warning-foreground')}>
          {weekOvertime >= 0 ? '+' : ''}{formatMinutes(weekOvertime)}
        </span>
        <span className="text-muted-foreground">|</span>
        <span className="text-muted-foreground">
          {t('profil.zeiterfassung.overview.actual')}: {formatHoursDecimal(weekTotal)}h / {t('profil.zeiterfassung.overview.target')}: {formatHoursDecimal(weeklyTarget)}h
        </span>
      </div>
    </div>
  )
}
