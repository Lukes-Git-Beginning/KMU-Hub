import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight, Palmtree, ThermometerSun, Home, BookOpen, Calendar } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import {
  useWorkTimeEntries, useTimeCategories, useAbsenceCalendar, useWeeklySummary,
} from '@/api/hooks/hr-hooks'
import { adaptWorkTimeEntryToFE } from '@/api/hr-client'
import {
  getWeekDates, dateToStr, formatDateShort, getDateRangeLabel,
  formatMinutes, formatHoursDecimal, isToday,
} from './time-utils'
import ApprovalBanner from './ApprovalBanner'

const ABSENCE_ICONS: Record<string, typeof Palmtree> = {
  vacation: Palmtree,
  urlaub: Palmtree,
  sick: ThermometerSun,
  krankheit: ThermometerSun,
  homeoffice: Home,
  education: BookOpen,
  other: Calendar,
}

const ABSENCE_COLORS: Record<string, string> = {
  vacation: 'text-warning-foreground',
  urlaub: 'text-warning-foreground',
  sick: 'text-gray-500 dark:text-gray-400',
  krankheit: 'text-gray-500 dark:text-gray-400',
  homeoffice: 'text-blue-500 dark:text-blue-400',
  education: 'text-purple-500 dark:text-purple-400',
  other: 'text-muted-foreground',
}

export default function WeekView() {
  const { t } = useTranslation()
  const [weekOffset, setWeekOffset] = useState(0)

  const ABSENCE_LABELS: Record<string, string> = {
    vacation: t('profil.zeiterfassung.absenceType.vacation'),
    urlaub: t('profil.zeiterfassung.absenceType.vacation'),
    sick: t('profil.zeiterfassung.absenceType.sick'),
    krankheit: t('profil.zeiterfassung.absenceType.sick'),
    homeoffice: t('profil.zeiterfassung.absenceType.homeoffice'),
    education: t('profil.zeiterfassung.absenceType.education'),
    other: t('profil.zeiterfassung.absenceType.other'),
  }

  const weekDates = useMemo(() => getWeekDates(weekOffset), [weekOffset])
  const dateStrings = useMemo(() => weekDates.map(dateToStr), [weekDates])
  const weekStartStr = dateStrings[0]
  const weekEndStr = dateStrings[6]

  const { data: entriesRaw } = useWorkTimeEntries({ start_date: weekStartStr, end_date: weekEndStr })
  const { data: categories = [] } = useTimeCategories()
  const { data: absences = [] } = useAbsenceCalendar({ start_date: weekStartStr, end_date: weekEndStr })
  const { data: weeklySummary } = useWeeklySummary(weekStartStr)

  const entries = useMemo(
    () => (entriesRaw?.entries ?? []).map(adaptWorkTimeEntryToFE),
    [entriesRaw],
  )

  // Build grid data: category → day → minutes
  const gridData = useMemo(() => {
    const data: Record<string, Record<string, number>> = {}
    for (const cat of categories) {
      data[cat.id] = {}
      for (const ds of dateStrings) {
        data[cat.id][ds] = 0
      }
    }
    for (const entry of entries) {
      const catId = entry.categoryId
      if (catId && dateStrings.includes(entry.date) && data[catId]) {
        data[catId][entry.date] += entry.durationMinutes
      }
    }
    return data
  }, [entries, categories, dateStrings])

  // Column totals
  const dayTotals = useMemo(() => {
    const totals: Record<string, number> = {}
    for (const ds of dateStrings) {
      totals[ds] = (Object.values(gridData) as Record<string, number>[]).reduce(
        (sum: number, catData: Record<string, number>) => sum + (catData[ds] || 0),
        0,
      )
    }
    return totals
  }, [gridData, dateStrings])

  // Row totals
  const catTotals = useMemo(() => {
    const totals: Record<string, number> = {}
    for (const cat of categories) {
      totals[cat.id] = dateStrings.reduce(
        (sum: number, ds: string) => sum + (gridData[cat.id]?.[ds] || 0),
        0,
      )
    }
    return totals
  }, [gridData, categories, dateStrings])

  const weekTotal = weeklySummary?.netWorkMinutes ?? (Object.values(dayTotals) as number[]).reduce((s: number, v: number) => s + v, 0)
  const weekTarget = 5 * 8.4 * 60
  const dailyTarget = 8.4 * 60
  const weekOvertime = (weeklySummary?.totalOvertimeMinutes ?? (weekTotal - weekTarget))

  // Only show categories that have entries this week
  const activeCats = categories.filter((c) => catTotals[c.id] > 0)

  // Absence days in this week
  const absenceDays = useMemo(() => {
    const map: Record<string, string> = {}
    for (const absence of absences) {
      for (const ds of dateStrings) {
        const d = new Date(ds)
        const start = new Date(absence.startDate)
        const end = new Date(absence.endDate)
        if (d >= start && d <= end) {
          map[ds] = absence.leaveTypeKey?.toLowerCase() ?? 'other'
        }
      }
    }
    return map
  }, [absences, dateStrings])

  return (
    <div className="p-6 space-y-4">
      {/* Approval Banner */}
      <ApprovalBanner weekStart={weekStartStr} />

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
              <th className="text-left px-4 py-3 text-muted-foreground font-medium w-40">{t('profil.zeiterfassung.manual.category')}</th>
              {weekDates.map((date, i) => {
                const absType = absenceDays[dateStrings[i]]
                const AbsIcon = absType ? (ABSENCE_ICONS[absType] || Calendar) : null
                return (
                  <th
                    key={i}
                    className={cn(
                      'text-center px-2 py-3 font-medium min-w-[80px]',
                      isToday(dateStrings[i]) ? 'text-primary bg-primary/5' : 'text-muted-foreground',
                    )}
                  >
                    <div className="flex flex-col items-center gap-0.5">
                      <span>{formatDateShort(date)}</span>
                      {absType && AbsIcon && (
                        <span className={cn('flex items-center gap-1 text-[10px]', ABSENCE_COLORS[absType] ?? 'text-muted-foreground')}>
                          <AbsIcon className="h-3 w-3" />
                          {ABSENCE_LABELS[absType] ?? absType}
                        </span>
                      )}
                    </div>
                  </th>
                )
              })}
              <th className="text-right px-4 py-3 text-foreground font-semibold w-24">Total</th>
            </tr>
          </thead>
          <tbody>
            {activeCats.map((cat) => (
              <tr key={cat.id} className="border-b border-border/50 hover:bg-accent/30 transition-colors">
                <td className="px-4 py-2.5">
                  <span className="flex items-center gap-2">
                    <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: cat.color }} />
                    <span className="font-medium text-foreground">{cat.name}</span>
                  </span>
                </td>
                {dateStrings.map((ds, i) => {
                  const mins = gridData[cat.id]?.[ds] || 0
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
                  {catTotals[cat.id] > 0 ? formatHoursDecimal(catTotals[cat.id]) : '-'}
                </td>
              </tr>
            ))}

            {activeCats.length === 0 && (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground">
                  {t('profil.zeiterfassung.week.noEntries')}
                </td>
              </tr>
            )}
          </tbody>
          <tfoot>
            {/* Day totals */}
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
            {/* Soll row */}
            <tr>
              <td className="px-4 py-2 text-xs text-muted-foreground">{t('profil.zeiterfassung.overview.target')}</td>
              {dateStrings.map((_, i) => (
                <td key={i} className="text-center px-2 py-2 text-xs text-muted-foreground tabular-nums">
                  {i < 5 ? formatHoursDecimal(dailyTarget) : '-'}
                </td>
              ))}
              <td className="text-right px-4 py-2 text-xs text-muted-foreground tabular-nums">
                {formatHoursDecimal(weekTarget)}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>

      {/* Week Summary */}
      <div className="flex items-center gap-4 text-sm">
        <span className="text-muted-foreground">{t('profil.zeiterfassung.week.weekBalance')}:</span>
        <span className={cn(
          'font-semibold',
          weekOvertime >= 0 ? 'text-success' : 'text-warning-foreground',
        )}>
          {weekOvertime >= 0 ? '+' : ''}{formatMinutes(weekOvertime)}
        </span>
        <span className="text-muted-foreground">|</span>
        <span className="text-muted-foreground">
          {t('profil.zeiterfassung.overview.actual')}: {formatHoursDecimal(weekTotal)}h / {t('profil.zeiterfassung.overview.target')}: {formatHoursDecimal(weekTarget)}h
        </span>
      </div>
    </div>
  )
}
