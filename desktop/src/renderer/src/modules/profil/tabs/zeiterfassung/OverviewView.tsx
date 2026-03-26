import { useMemo } from 'react'
import {
  Clock, TrendingUp, TrendingDown, Palmtree, Calendar,
  ArrowRight, Timer,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore, getOvertimeSaldo } from '@/stores/timetracking'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import {
  formatMinutes, formatHoursDecimal, isToday,
  getWeekDates, dateToStr,
} from './time-utils'

const VACATION_TOTAL = 25
const VACATION_USED = 10

interface OverviewViewProps {
  onNavigate: (view: string) => void
}

export default function OverviewView({ onNavigate }: OverviewViewProps) {
  const entries = useTimeTrackingStore((s) => s.entries)
  const categories = useTimeTrackingStore((s) => s.categories)
  const targets = useTimeTrackingStore((s) => s.targets)
  const absences = useTimeTrackingStore((s) => s.absences)
  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const elapsed = useTimerTick()

  // ── This Week ───────────────────────────────────────
  const thisWeekDates = useMemo(() => getWeekDates(0).map(dateToStr), [])
  const thisWeekEntries = useMemo(
    () => entries.filter((e) => thisWeekDates.includes(e.date)),
    [entries, thisWeekDates],
  )
  const weekMinutes = thisWeekEntries.reduce((s, e) => s + e.durationMinutes, 0)
  const weekTarget = targets.weeklyHours * 60
  const weekPercent = Math.min(100, Math.round((weekMinutes / weekTarget) * 100))

  // ── Today ───────────────────────────────────────────
  const todayEntries = useMemo(
    () => entries.filter((e) => isToday(e.date)),
    [entries],
  )
  const todayMinutes = todayEntries.reduce((s, e) => s + e.durationMinutes, 0)
  const dailyTarget = targets.dailyHours * 60
  const todayPercent = Math.min(100, Math.round((todayMinutes / dailyTarget) * 100))

  // ── Overtime (all-time, using helper 6.8) ───────────
  const overtime = useMemo(() => getOvertimeSaldo(entries, targets), [entries, targets])
  const uniqueDates = [...new Set(entries.map((e) => e.date))]
  const workingDays = uniqueDates.filter((ds) => {
    const d = new Date(ds)
    return d.getDay() >= 1 && d.getDay() <= 5
  }).length
  const totalMinutesAllTime = entries.reduce((s, e) => s + e.durationMinutes, 0)
  const totalTarget = workingDays * targets.dailyHours * 60

  // ── This Month ──────────────────────────────────────
  const thisMonth = new Date().toISOString().slice(0, 7) // YYYY-MM
  const monthEntries = useMemo(
    () => entries.filter((e) => e.date.startsWith(thisMonth)),
    [entries, thisMonth],
  )
  const monthMinutes = monthEntries.reduce((s, e) => s + e.durationMinutes, 0)

  // ── Vacation ────────────────────────────────────────
  const vacationRemaining = VACATION_TOTAL - VACATION_USED
  const pendingVacation = absences.filter(
    (a) => a.type === 'vacation' && a.status === 'pending',
  ).reduce((s, a) => s + a.days, 0)
  const sickDays = absences.filter(
    (a) => a.type === 'sick' && a.status === 'approved',
  ).reduce((s, a) => s + a.days, 0)

  // ── Category breakdown this week ────────────────────
  const weekCategoryStats = useMemo(() => {
    const stats: Record<string, number> = {}
    for (const e of thisWeekEntries) {
      stats[e.categoryId] = (stats[e.categoryId] || 0) + e.durationMinutes
    }
    return categories
      .map((cat) => ({ ...cat, minutes: stats[cat.id] || 0 }))
      .filter((c) => c.minutes > 0)
      .sort((a, b) => b.minutes - a.minutes)
  }, [thisWeekEntries, categories])

  // ── Daily breakdown this week ───────────────────────
  const weekDayData = useMemo(() => {
    const dayNames = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']
    return getWeekDates(0).map((date, i) => {
      const ds = dateToStr(date)
      const mins = entries
        .filter((e) => e.date === ds)
        .reduce((s, e) => s + e.durationMinutes, 0)
      return { day: dayNames[i], date: ds, minutes: mins, isToday: isToday(ds) }
    })
  }, [entries])

  // ── Recent entries ──────────────────────────────────
  const recentEntries = useMemo(
    () => [...entries]
      .sort((a, b) => `${b.date}${b.startTime}`.localeCompare(`${a.date}${a.startTime}`))
      .slice(0, 5),
    [entries],
  )

  const getCat = (id: string) => categories.find((c) => c.id === id)

  return (
    <div className="p-6 space-y-6 max-w-4xl mx-auto">
      {/* Active Timer Banner */}
      {activeTimer.status !== 'idle' && activeTimer.categoryId && (
        <div className="rounded-xl border-2 border-primary/50 bg-primary/5 p-4">
          <div className="flex items-center gap-3">
            <div className="relative">
              <Timer className="h-5 w-5 text-primary" />
              {activeTimer.status === 'running' && (
                <span className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-success animate-pulse" />
              )}
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-foreground">
                {getCat(activeTimer.categoryId)?.name}
                {activeTimer.description && (
                  <span className="text-muted-foreground ml-2">— {activeTimer.description}</span>
                )}
              </p>
            </div>
            <span className="text-2xl font-mono font-bold text-primary tabular-nums">
              {formatElapsed(elapsed)}
            </span>
          </div>
        </div>
      )}

      {/* Top Stats Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {/* Today */}
        <button
          onClick={() => onNavigate('today')}
          className="rounded-xl border border-border bg-card p-5 text-left hover:border-primary/40 transition-colors group"
        >
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Heute</span>
            <Clock className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors" />
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className="text-2xl font-bold text-foreground tabular-nums">
              {formatHoursDecimal(todayMinutes)}
            </span>
            <span className="text-sm text-muted-foreground mb-0.5">/ {formatHoursDecimal(dailyTarget)}h</span>
          </div>
          <div className="w-full h-2 rounded-full bg-secondary overflow-hidden">
            <div
              className={cn(
                'h-full rounded-full transition-all',
                todayPercent >= 90 ? 'bg-success' : todayPercent >= 60 ? 'bg-warning' : 'bg-primary/40',
              )}
              style={{ width: `${todayPercent}%` }}
            />
          </div>
          <p className="text-xs text-muted-foreground mt-1.5">
            {todayMinutes >= dailyTarget
              ? `+${formatMinutes(todayMinutes - dailyTarget)} über Soll`
              : `Noch ${formatMinutes(dailyTarget - todayMinutes)}`}
          </p>
        </button>

        {/* This Week */}
        <button
          onClick={() => onNavigate('week')}
          className="rounded-xl border border-border bg-card p-5 text-left hover:border-primary/40 transition-colors group"
        >
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Diese Woche</span>
            <Calendar className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors" />
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className="text-2xl font-bold text-foreground tabular-nums">
              {formatHoursDecimal(weekMinutes)}
            </span>
            <span className="text-sm text-muted-foreground mb-0.5">/ {targets.weeklyHours}h</span>
          </div>
          <div className="w-full h-2 rounded-full bg-secondary overflow-hidden">
            <div
              className={cn(
                'h-full rounded-full transition-all',
                weekPercent >= 90 ? 'bg-success' : weekPercent >= 60 ? 'bg-warning' : 'bg-primary/40',
              )}
              style={{ width: `${weekPercent}%` }}
            />
          </div>
          <p className="text-xs text-muted-foreground mt-1.5">{weekPercent}% erledigt</p>
        </button>

        {/* Overtime */}
        <button
          onClick={() => onNavigate('reports')}
          className="rounded-xl border border-border bg-card p-5 text-left hover:border-primary/40 transition-colors group"
        >
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Überstunden</span>
            {overtime >= 0 ? (
              <TrendingUp className="h-4 w-4 text-success" />
            ) : (
              <TrendingDown className="h-4 w-4 text-warning" />
            )}
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className={cn(
              'text-2xl font-bold tabular-nums',
              overtime >= 0 ? 'text-success' : 'text-warning-foreground',
            )}>
              {overtime >= 0 ? '+' : ''}{formatHoursDecimal(overtime)}h
            </span>
          </div>
          <p className="text-xs text-muted-foreground mt-1.5">
            Soll: {formatMinutes(totalTarget)} | Ist: {formatMinutes(totalMinutesAllTime)}
          </p>
          <p className="text-xs text-muted-foreground">
            Basierend auf {workingDays} Arbeitstagen
          </p>
        </button>

        {/* Vacation */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Resturlaub</span>
            <Palmtree className="h-4 w-4 text-success" />
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className="text-2xl font-bold text-foreground tabular-nums">{vacationRemaining}</span>
            <span className="text-sm text-muted-foreground mb-0.5">/ {VACATION_TOTAL} Tage</span>
          </div>
          <div className="w-full h-2 rounded-full bg-secondary overflow-hidden">
            <div
              className="h-full rounded-full bg-success transition-all"
              style={{ width: `${Math.round((VACATION_USED / VACATION_TOTAL) * 100)}%` }}
            />
          </div>
          <div className="flex items-center gap-3 mt-1.5">
            <span className="text-xs text-muted-foreground">{VACATION_USED} genommen</span>
            {pendingVacation > 0 && (
              <span className="text-xs text-warning-foreground">{pendingVacation} beantragt</span>
            )}
          </div>
        </div>
      </div>

      {/* Week at a Glance + Category Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Daily Bar Chart */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-foreground">Woche im Überblick</h3>
            <button
              onClick={() => onNavigate('week')}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              Details <ArrowRight className="h-3 w-3" />
            </button>
          </div>
          <div className="flex items-end gap-2" style={{ height: 120 }}>
            {weekDayData.map((day, i) => {
              const isWeekend = i >= 5
              const target = isWeekend ? 0 : dailyTarget
              const barHeight = day.minutes > 0 ? Math.max(8, (day.minutes / Math.max(dailyTarget, day.minutes)) * 100) : 0
              const isOver = day.minutes >= target && target > 0
              return (
                <div key={day.day} className="flex-1 flex flex-col items-center gap-1">
                  <span className="text-[10px] text-muted-foreground tabular-nums">
                    {day.minutes > 0 ? formatHoursDecimal(day.minutes) : ''}
                  </span>
                  <div className="w-full relative" style={{ height: 80 }}>
                    {!isWeekend && (
                      <div
                        className="absolute left-0 right-0 border-t border-dashed border-muted-foreground/30"
                        style={{ bottom: '100%' }}
                      />
                    )}
                    <div
                      className={cn(
                        'absolute bottom-0 left-1 right-1 rounded-t-md transition-all',
                        day.isToday ? 'bg-primary' : isOver ? 'bg-success' : day.minutes > 0 ? 'bg-primary/60' : 'bg-transparent',
                      )}
                      style={{ height: `${barHeight}%` }}
                    />
                  </div>
                  <span className={cn(
                    'text-xs font-medium',
                    day.isToday ? 'text-primary' : 'text-muted-foreground',
                  )}>
                    {day.day}
                  </span>
                </div>
              )
            })}
          </div>
          <div className="mt-3 flex items-center gap-4 text-xs text-muted-foreground">
            <span>Soll: {targets.weeklyHours}h</span>
            <span>|</span>
            <span className={cn(
              'font-medium',
              weekMinutes >= weekTarget ? 'text-success' : 'text-foreground',
            )}>
              Ist: {formatHoursDecimal(weekMinutes)}h
            </span>
            <span>|</span>
            <span className={cn(
              'font-medium',
              weekMinutes >= weekTarget ? 'text-success' : 'text-warning-foreground',
            )}>
              {weekMinutes >= weekTarget ? '+' : ''}{formatMinutes(weekMinutes - weekTarget)}
            </span>
          </div>
        </div>

        {/* Category Breakdown */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-foreground">Kategorien (diese Woche)</h3>
            <button
              onClick={() => onNavigate('categories')}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              Verwalten <ArrowRight className="h-3 w-3" />
            </button>
          </div>
          {weekCategoryStats.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">Keine Einträge diese Woche</p>
          ) : (
            <div className="space-y-3">
              {weekCategoryStats.map((cat) => {
                const percent = weekMinutes > 0 ? Math.round((cat.minutes / weekMinutes) * 100) : 0
                return (
                  <div key={cat.id}>
                    <div className="flex items-center justify-between mb-1">
                      <span className="flex items-center gap-2 text-sm">
                        <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: cat.color }} />
                        <span className="text-foreground">{cat.name}</span>
                      </span>
                      <span className="text-xs text-muted-foreground tabular-nums">
                        {formatHoursDecimal(cat.minutes)}h ({percent}%)
                      </span>
                    </div>
                    <div className="w-full h-2 rounded-full bg-secondary overflow-hidden">
                      <div
                        className="h-full rounded-full transition-all"
                        style={{ width: `${percent}%`, backgroundColor: cat.color }}
                      />
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      {/* Key Metrics + Recent Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Key Metrics */}
        <div className="rounded-xl border border-border bg-card p-5">
          <h3 className="text-sm font-semibold text-foreground mb-4">Arbeitsvertrag</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Wochenstunden (Soll)</span>
              <span className="font-medium text-foreground">{targets.weeklyHours}h</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Tagesstunden (Soll)</span>
              <span className="font-medium text-foreground">{targets.dailyHours}h</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Monatsstunden (Soll)</span>
              <span className="font-medium text-foreground">{targets.monthlyHours}h</span>
            </div>
            <div className="border-t border-border pt-3 flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Dieser Monat (Ist)</span>
              <span className="font-semibold text-foreground">{formatHoursDecimal(monthMinutes)}h</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Krankheitstage (Jahr)</span>
              <span className="font-medium text-foreground">{sickDays} Tage</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Urlaubsanspruch</span>
              <span className="font-medium text-foreground">{VACATION_TOTAL} Tage/Jahr</span>
            </div>
          </div>
        </div>

        {/* Recent Activity */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-foreground">Letzte Einträge</h3>
            <button
              onClick={() => onNavigate('today')}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              Alle <ArrowRight className="h-3 w-3" />
            </button>
          </div>
          {recentEntries.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">Noch keine Einträge</p>
          ) : (
            <div className="space-y-2">
              {recentEntries.map((entry) => {
                const cat = getCat(entry.categoryId)
                const entryIsToday = isToday(entry.date)
                return (
                  <div
                    key={entry.id}
                    className="flex items-center gap-2.5 py-1.5"
                  >
                    <span
                      className="h-2 w-2 rounded-full shrink-0"
                      style={{ backgroundColor: cat?.color || '#6b7280' }}
                    />
                    <span className="text-xs text-muted-foreground w-16 shrink-0 tabular-nums">
                      {entryIsToday ? 'Heute' : formatDateLabel(entry.date)}
                    </span>
                    <span className="text-sm text-foreground truncate flex-1">
                      {entry.description}
                    </span>
                    <span className="text-xs text-muted-foreground tabular-nums shrink-0">
                      {formatMinutes(entry.durationMinutes)}
                    </span>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function formatDateLabel(dateStr: string): string {
  const d = new Date(dateStr)
  const today = new Date()
  const yesterday = new Date()
  yesterday.setDate(today.getDate() - 1)

  if (dateStr === yesterday.toISOString().split('T')[0]) return 'Gestern'

  return `${d.getDate().toString().padStart(2, '0')}.${(d.getMonth() + 1).toString().padStart(2, '0')}.`
}
