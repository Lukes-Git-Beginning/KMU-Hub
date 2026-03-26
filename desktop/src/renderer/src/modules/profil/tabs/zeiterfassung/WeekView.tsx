import { useState, useMemo } from 'react'
import { ChevronLeft, ChevronRight, Palmtree, ThermometerSun, Home, BookOpen, Calendar } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore, getOvertimeForWeek } from '@/stores/timetracking'
import {
  getWeekDates, dateToStr, formatDateShort, getDateRangeLabel,
  formatMinutes, formatHoursDecimal, isToday,
} from './time-utils'
import ApprovalBanner from './ApprovalBanner'

const ABSENCE_ICONS: Record<string, typeof Palmtree> = {
  vacation: Palmtree,
  sick: ThermometerSun,
  homeoffice: Home,
  education: BookOpen,
  other: Calendar,
}

const ABSENCE_LABELS: Record<string, string> = {
  vacation: 'Urlaub',
  sick: 'Krank',
  homeoffice: 'Homeoffice',
  education: 'Weiterbildung',
  other: 'Sonstiges',
}

const ABSENCE_COLORS: Record<string, string> = {
  vacation: 'text-warning-foreground',
  sick: 'text-gray-500 dark:text-gray-400',
  homeoffice: 'text-blue-500 dark:text-blue-400',
  education: 'text-purple-500 dark:text-purple-400',
  other: 'text-muted-foreground',
}

export default function WeekView() {
  const [weekOffset, setWeekOffset] = useState(0)
  const entries = useTimeTrackingStore((s) => s.entries)
  const categories = useTimeTrackingStore((s) => s.categories)
  const targets = useTimeTrackingStore((s) => s.targets)
  const absences = useTimeTrackingStore((s) => s.absences)

  const weekDates = useMemo(() => getWeekDates(weekOffset), [weekOffset])
  const dateStrings = useMemo(() => weekDates.map(dateToStr), [weekDates])

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
      if (dateStrings.includes(entry.date) && data[entry.categoryId]) {
        data[entry.categoryId][entry.date] += entry.durationMinutes
      }
    }
    return data
  }, [entries, categories, dateStrings])

  // Column totals
  const dayTotals = useMemo(() => {
    const totals: Record<string, number> = {}
    for (const ds of dateStrings) {
      totals[ds] = Object.values(gridData).reduce((sum, catData) => sum + (catData[ds] || 0), 0)
    }
    return totals
  }, [gridData, dateStrings])

  // Row totals
  const catTotals = useMemo(() => {
    const totals: Record<string, number> = {}
    for (const cat of categories) {
      totals[cat.id] = dateStrings.reduce((sum, ds) => sum + (gridData[cat.id]?.[ds] || 0), 0)
    }
    return totals
  }, [gridData, categories, dateStrings])

  const weekTotal = Object.values(dayTotals).reduce((s, v) => s + v, 0)
  const weekTarget = targets.weeklyHours * 60
  const dailyTarget = targets.dailyHours * 60

  // Only show categories that have entries this week
  const activeCats = categories.filter((c) => catTotals[c.id] > 0)

  // Overtime for the week (6.8)
  const weekStartStr = dateStrings[0]
  const weekOvertime = getOvertimeForWeek(entries, weekStartStr, weekTarget)

  // Absence days in this week (6.10)
  const absenceDays = useMemo(() => {
    const map: Record<string, string> = {} // dateStr -> absence type
    for (const absence of absences) {
      if (absence.status !== 'approved') continue
      const start = new Date(absence.startDate)
      const end = new Date(absence.endDate)
      for (const ds of dateStrings) {
        const d = new Date(ds)
        if (d >= start && d <= end) {
          map[ds] = absence.type
        }
      }
    }
    return map
  }, [absences, dateStrings])

  return (
    <div className="p-6 space-y-4">
      {/* Approval Banner (6.9) */}
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
            Diese Woche
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
              <th className="text-left px-4 py-3 text-muted-foreground font-medium w-40">Kategorie</th>
              {weekDates.map((date, i) => {
                const absType = absenceDays[dateStrings[i]]
                const AbsIcon = absType ? ABSENCE_ICONS[absType] : null
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
                        <span className={cn('flex items-center gap-1 text-[10px]', ABSENCE_COLORS[absType])}>
                          <AbsIcon className="h-3 w-3" />
                          {ABSENCE_LABELS[absType]}
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
                  Keine Einträge in dieser Woche
                </td>
              </tr>
            )}
          </tbody>
          <tfoot>
            {/* Day totals */}
            <tr className="border-t-2 border-border">
              <td className="px-4 py-2.5 font-semibold text-foreground">Gesamt</td>
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
              <td className="px-4 py-2 text-xs text-muted-foreground">Soll</td>
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

      {/* Week Summary with Overtime (6.8) */}
      <div className="flex items-center gap-4 text-sm">
        <span className="text-muted-foreground">Wochensaldo:</span>
        <span className={cn(
          'font-semibold',
          weekOvertime >= 0 ? 'text-success' : 'text-warning-foreground',
        )}>
          {weekOvertime >= 0 ? '+' : ''}{formatMinutes(weekOvertime)}
        </span>
        <span className="text-muted-foreground">|</span>
        <span className="text-muted-foreground">
          Ist: {formatHoursDecimal(weekTotal)}h / Soll: {formatHoursDecimal(weekTarget)}h
        </span>
      </div>
    </div>
  )
}
