import { useState, useMemo } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'
import {
  getMonthDates, dateToStr, getMonthLabel, formatMinutes, isToday,
} from './time-utils'

export default function MonthView() {
  const [monthOffset, setMonthOffset] = useState(0)
  const entries = useTimeTrackingStore((s) => s.entries)
  const categories = useTimeTrackingStore((s) => s.categories)
  const targets = useTimeTrackingStore((s) => s.targets)

  const monthDates = useMemo(() => getMonthDates(monthOffset), [monthOffset])
  const dailyTarget = targets.dailyHours * 60

  // Minutes per day per category
  const dayData = useMemo(() => {
    const data: Record<string, { total: number; byCat: Record<string, number> }> = {}
    for (const d of monthDates) {
      const ds = dateToStr(d)
      data[ds] = { total: 0, byCat: {} }
    }
    for (const entry of entries) {
      if (data[entry.date]) {
        data[entry.date].total += entry.durationMinutes
        data[entry.date].byCat[entry.categoryId] =
          (data[entry.date].byCat[entry.categoryId] || 0) + entry.durationMinutes
      }
    }
    return data
  }, [entries, monthDates])

  // Stats
  const daysWithEntries = Object.values(dayData).filter((d) => d.total > 0).length
  const totalMinutes = Object.values(dayData).reduce((s, d) => s + d.total, 0)
  const avgPerDay = daysWithEntries > 0 ? Math.round(totalMinutes / daysWithEntries) : 0

  // Working days in month (Mon-Fri)
  const workDays = monthDates.filter((d) => d.getDay() >= 1 && d.getDay() <= 5).length
  const monthTarget = workDays * dailyTarget

  const maxMinutesInDay = Math.max(dailyTarget, ...Object.values(dayData).map((d) => d.total))

  return (
    <div className="p-6 space-y-4">
      {/* Navigation */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => setMonthOffset((m) => m - 1)}>
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button
            size="sm"
            variant={monthOffset === 0 ? 'default' : 'outline'}
            onClick={() => setMonthOffset(0)}
          >
            Aktuell
          </Button>
          <Button size="sm" variant="outline" onClick={() => setMonthOffset((m) => m + 1)}>
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
        <span className="text-sm font-medium text-foreground">{getMonthLabel(monthOffset)}</span>
      </div>

      {/* Bar Chart */}
      <div className="rounded-xl border border-border bg-card p-4 overflow-x-auto">
        <div className="flex items-end gap-[3px] min-w-[600px]" style={{ height: 200 }}>
          {monthDates.map((date) => {
            const ds = dateToStr(date)
            const data = dayData[ds]
            const isWeekend = date.getDay() === 0 || date.getDay() === 6
            const barHeight = data.total > 0 ? Math.max(4, (data.total / maxMinutesInDay) * 180) : 0
            const targetLine = (dailyTarget / maxMinutesInDay) * 180
            const _isUnderTarget = data.total > 0 && data.total < dailyTarget && !isWeekend

            // Stack categories
            const catEntries = Object.entries(data.byCat)
            let _stackOffset = 0

            return (
              <div
                key={ds}
                className="flex-1 relative group"
                style={{ height: '100%' }}
              >
                {/* Target line */}
                {!isWeekend && (
                  <div
                    className="absolute left-0 right-0 border-t border-dashed border-muted-foreground/30"
                    style={{ bottom: targetLine }}
                  />
                )}

                {/* Stacked bars */}
                <div
                  className="absolute bottom-0 left-[1px] right-[1px] rounded-t-sm overflow-hidden transition-all"
                  style={{ height: barHeight }}
                >
                  {catEntries.map(([catId, mins]) => {
                    const cat = categories.find((c) => c.id === catId)
                    const segHeight = (mins / data.total) * 100
                    const el = (
                      <div
                        key={catId}
                        style={{ height: `${segHeight}%`, backgroundColor: cat?.color || '#6b7280' }}
                      />
                    )
                    _stackOffset += segHeight
                    return el
                  })}
                  {catEntries.length === 0 && <div className="w-full h-full bg-muted" />}
                </div>

                {/* Date label */}
                <div className={cn(
                  'absolute -bottom-5 left-0 right-0 text-center text-[9px]',
                  isToday(ds) ? 'text-primary font-bold' : isWeekend ? 'text-muted-foreground/40' : 'text-muted-foreground',
                )}>
                  {date.getDate()}
                </div>

                {/* Tooltip on hover */}
                <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block z-10 pointer-events-none">
                  <div className="bg-popover border border-border text-popover-foreground text-xs rounded-lg px-3 py-2 shadow-lg whitespace-nowrap">
                    <p className="font-medium">{date.getDate()}. {getMonthLabel(monthOffset).split(' ')[0]}</p>
                    <p>{data.total > 0 ? formatMinutes(data.total) : 'Keine Einträge'}</p>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-3">
        {categories.filter((c) => entries.some((e) => e.categoryId === c.id)).map((cat) => (
          <span key={cat.id} className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: cat.color }} />
            {cat.name}
          </span>
        ))}
      </div>

      {/* Month Summary */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">Total</p>
          <p className="text-xl font-bold text-foreground mt-1">{formatMinutes(totalMinutes)}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">Durchschnitt/Tag</p>
          <p className="text-xl font-bold text-foreground mt-1">{formatMinutes(avgPerDay)}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">Soll (Monat)</p>
          <p className="text-xl font-bold text-foreground mt-1">{formatMinutes(monthTarget)}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">Saldo</p>
          <p className={cn(
            'text-xl font-bold mt-1',
            totalMinutes >= monthTarget ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400',
          )}>
            {totalMinutes >= monthTarget ? '+' : ''}{formatMinutes(totalMinutes - monthTarget)}
          </p>
        </div>
      </div>
    </div>
  )
}
