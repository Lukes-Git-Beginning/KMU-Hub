import { useMemo } from 'react'
import { Download } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'
import { formatMinutes, formatHoursDecimal, getWeekDates, dateToStr } from './time-utils'
import { toast } from 'sonner'

export default function ReportsView() {
  const entries = useTimeTrackingStore((s) => s.entries)
  const categories = useTimeTrackingStore((s) => s.categories)
  const targets = useTimeTrackingStore((s) => s.targets)

  // Hours per category (all time)
  const categoryStats = useMemo(() => {
    const stats: Record<string, number> = {}
    for (const e of entries) {
      stats[e.categoryId] = (stats[e.categoryId] || 0) + e.durationMinutes
    }
    return categories
      .map((cat) => ({ ...cat, minutes: stats[cat.id] || 0 }))
      .filter((c) => c.minutes > 0)
      .sort((a, b) => b.minutes - a.minutes)
  }, [entries, categories])

  const maxCategoryMinutes = Math.max(1, ...categoryStats.map((c) => c.minutes))
  const totalMinutes = categoryStats.reduce((s, c) => s + c.minutes, 0)

  // Weekly trend (last 4 weeks)
  const weeklyTrend = useMemo(() => {
    return Array.from({ length: 4 }, (_, i) => {
      const weekDates = getWeekDates(-3 + i)
      const dateStrs = weekDates.map(dateToStr)
      const mins = entries
        .filter((e) => dateStrs.includes(e.date))
        .reduce((s, e) => s + e.durationMinutes, 0)
      const weekNum = -3 + i
      return {
        label: weekNum === 0 ? 'Diese Woche' : weekNum === -1 ? 'Letzte Woche' : `KW ${weekNum + getISOWeek()}`,
        minutes: mins,
      }
    })
  }, [entries])

  const maxWeekMinutes = Math.max(1, ...weeklyTrend.map((w) => w.minutes))
  const weeklyTarget = targets.weeklyHours * 60

  // Overtime calculation
  // Simple: total entries minutes vs (working days * daily target)
  const uniqueDates = [...new Set(entries.map((e) => e.date))]
  const workingDays = uniqueDates.filter((ds) => {
    const d = new Date(ds)
    return d.getDay() >= 1 && d.getDay() <= 5
  }).length
  const totalTarget = workingDays * targets.dailyHours * 60
  const overtime = totalMinutes - totalTarget

  return (
    <div className="p-6 space-y-6 max-w-3xl mx-auto">
      {/* Hours per Category */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold text-foreground mb-4">Stunden pro Kategorie</h3>
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
        </div>
      </div>

      {/* Weekly Trend */}
      <div className="rounded-xl border border-border bg-card p-5">
        <h3 className="text-sm font-semibold text-foreground mb-4">Wochentrend</h3>
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
                      isOverTarget ? 'bg-emerald-500' : 'bg-amber-500',
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
          Soll: {formatHoursDecimal(weeklyTarget)}h/Woche
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-2">Soll / Ist Vergleich</h3>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Gearbeitet</span>
              <span className="font-medium text-foreground">{formatMinutes(totalMinutes)}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Soll ({workingDays} Arbeitstage)</span>
              <span className="font-medium text-foreground">{formatMinutes(totalTarget)}</span>
            </div>
            <div className="border-t border-border pt-2 flex justify-between text-sm">
              <span className="text-muted-foreground">Differenz</span>
              <span className={cn(
                'font-semibold',
                overtime >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400',
              )}>
                {overtime >= 0 ? '+' : ''}{formatMinutes(overtime)}
              </span>
            </div>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-2">Überstundensaldo</h3>
          <div className="flex items-end gap-2 mb-2">
            <span className={cn(
              'text-3xl font-bold',
              overtime >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400',
            )}>
              {overtime >= 0 ? '+' : ''}{formatHoursDecimal(overtime)}h
            </span>
          </div>
          <p className="text-xs text-muted-foreground">
            {overtime >= 0
              ? 'Du hast mehr gearbeitet als geplant'
              : 'Du hast weniger gearbeitet als geplant'}
          </p>
        </div>
      </div>

      {/* Export */}
      <div className="flex gap-3">
        <Button
          variant="outline"
          size="sm"
          className="gap-2"
          onClick={() => toast.success('CSV Export wird vorbereitet...')}
        >
          <Download className="h-4 w-4" />
          CSV Export
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="gap-2"
          onClick={() => toast.success('PDF Export wird vorbereitet...')}
        >
          <Download className="h-4 w-4" />
          PDF Export
        </Button>
      </div>
    </div>
  )
}

function getISOWeek(): number {
  const now = new Date()
  const d = new Date(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate()))
  const dayNum = d.getUTCDay() || 7
  d.setUTCDate(d.getUTCDate() + 4 - dayNum)
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1))
  return Math.ceil(((d.getTime() - yearStart.getTime()) / 86400000 + 1) / 7)
}
