import { useState, useMemo } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'
import {
  getWeekDates, dateToStr, formatDateShort, getDateRangeLabel,
  formatMinutes, formatHoursDecimal, isToday,
} from './time-utils'

export default function WeekView() {
  const [weekOffset, setWeekOffset] = useState(0)
  const entries = useTimeTrackingStore((s) => s.entries)
  const categories = useTimeTrackingStore((s) => s.categories)
  const targets = useTimeTrackingStore((s) => s.targets)

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

  return (
    <div className="p-6 space-y-4">
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
                  Keine Eintraege in dieser Woche
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

      {/* Week Summary */}
      <div className="flex items-center gap-4 text-sm">
        <span className="text-muted-foreground">Wochensaldo:</span>
        <span className={cn(
          'font-semibold',
          weekTotal >= weekTarget ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400',
        )}>
          {weekTotal >= weekTarget ? '+' : ''}{formatMinutes(weekTotal - weekTarget)}
        </span>
      </div>
    </div>
  )
}
