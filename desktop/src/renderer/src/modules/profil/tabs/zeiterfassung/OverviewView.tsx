import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
import {
  Clock, TrendingUp, TrendingDown, Palmtree, Calendar,
  ArrowRight, Timer,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import {
  useDailySummary,
  useWeeklySummary,
  useTimeBalance,
  useWorkTimeEntries,
  useWorkTimeStatus,
  useHRSettings,
} from '@/api/hooks/hr-hooks'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, formatHoursDecimal, getWeekDates, dateToStr, isToday } from './time-utils'
import type { WorkTimeEntry } from '@/api/hr-types'

interface OverviewViewProps {
  onNavigate: (view: string) => void
}

function getWeekStartStr(): string {
  const d = new Date()
  const day = d.getDay()
  const mondayOffset = day === 0 ? -6 : 1 - day
  d.setDate(d.getDate() + mondayOffset)
  return d.toISOString().split('T')[0]
}

export default function OverviewView({ onNavigate }: OverviewViewProps) {
  const { t } = useTranslation()
  const elapsed = useTimerTick()

  const weekStart = getWeekStartStr()
  const today = new Date().toISOString().split('T')[0]

  const { data: dailySummary } = useDailySummary(today)
  const { data: weeklySummary } = useWeeklySummary(weekStart)
  const { data: balance } = useTimeBalance()
  const { data: entriesData } = useWorkTimeEntries()
  const allEntries = entriesData?.entries ?? []
  const { data: status } = useWorkTimeStatus()
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const { data: hrSettings } = useHRSettings() as { data: any }

  const dailyTarget = (hrSettings?.work_hours_per_day ?? 8) * 60
  const weeklyTarget = dailyTarget * 5

  const todayMinutes = dailySummary?.netWorkMinutes ?? 0
  const weekMinutes = weeklySummary?.netWorkMinutes ?? 0
  const overtimeMinutes = balance?.balanceMinutes ?? 0
  const todayPercent = Math.min(100, Math.round((todayMinutes / dailyTarget) * 100))
  const weekPercent = Math.min(100, Math.round((weekMinutes / weeklyTarget) * 100))

  // Daily bar chart: last 7 days
  const weekDayData = useMemo(() => {
    const dayNames = [
      t('profil.zeiterfassung.dayShort.mon'),
      t('profil.zeiterfassung.dayShort.tue'),
      t('profil.zeiterfassung.dayShort.wed'),
      t('profil.zeiterfassung.dayShort.thu'),
      t('profil.zeiterfassung.dayShort.fri'),
      t('profil.zeiterfassung.dayShort.sat'),
      t('profil.zeiterfassung.dayShort.sun'),
    ]
    if (weeklySummary?.days) {
      return weeklySummary.days.map((day, i) => ({
        day: dayNames[i] ?? String(i),
        date: day.date,
        minutes: day.netWorkMinutes,
        isToday: isToday(day.date),
      }))
    }
    // Fallback: derive from allEntries
    return getWeekDates(0).map((date, i) => {
      const ds = dateToStr(date)
      const mins = (allEntries as WorkTimeEntry[])
        .filter((e) => e.clockIn?.slice(0, 10) === ds)
        .reduce((s, e) => s + (e.netWorkMinutes ?? 0), 0)
      return { day: dayNames[i] ?? String(i), date: ds, minutes: mins, isToday: isToday(ds) }
    })
  }, [weeklySummary, allEntries, t])

  // Recent entries (last 5)
  const recentEntries = useMemo(
    () =>
      [...(allEntries as WorkTimeEntry[])]
        .sort((a, b) => (b.clockIn ?? '').localeCompare(a.clockIn ?? ''))
        .slice(0, 5),
    [allEntries],
  )

  const isClockedIn = status?.isClockedIn ?? false

  return (
    <div className="p-6 space-y-6 max-w-4xl mx-auto">
      {/* Active Shift Banner */}
      {isClockedIn && (
        <div className="rounded-xl border-2 border-primary/50 bg-primary/5 p-4">
          <div className="flex items-center gap-3">
            <div className="relative">
              <Timer className="h-5 w-5 text-primary" />
              <span className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-success animate-pulse" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-foreground">
                {t('profil.zeiterfassung.today.activeShift')}
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
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('profil.zeiterfassung.viewToday')}
            </span>
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
              ? `+${formatMinutes(todayMinutes - dailyTarget)} ${t('profil.zeiterfassung.overview.aboveTarget')}`
              : `${t('profil.zeiterfassung.overview.remaining')} ${formatMinutes(dailyTarget - todayMinutes)}`}
          </p>
        </button>

        {/* This Week */}
        <button
          onClick={() => onNavigate('week')}
          className="rounded-xl border border-border bg-card p-5 text-left hover:border-primary/40 transition-colors group"
        >
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('profil.zeiterfassung.overview.thisWeek')}
            </span>
            <Calendar className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors" />
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className="text-2xl font-bold text-foreground tabular-nums">
              {formatHoursDecimal(weekMinutes)}
            </span>
            <span className="text-sm text-muted-foreground mb-0.5">/ {formatHoursDecimal(weeklyTarget)}h</span>
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
          <p className="text-xs text-muted-foreground mt-1.5">{weekPercent}% {t('profil.zeiterfassung.overview.done')}</p>
        </button>

        {/* Overtime */}
        <button
          onClick={() => onNavigate('reports')}
          className="rounded-xl border border-border bg-card p-5 text-left hover:border-primary/40 transition-colors group"
        >
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('profil.zeiterfassung.overtime')}
            </span>
            {overtimeMinutes >= 0 ? (
              <TrendingUp className="h-4 w-4 text-success" />
            ) : (
              <TrendingDown className="h-4 w-4 text-warning" />
            )}
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className={cn(
              'text-2xl font-bold tabular-nums',
              overtimeMinutes >= 0 ? 'text-success' : 'text-warning-foreground',
            )}>
              {overtimeMinutes >= 0 ? '+' : ''}{formatHoursDecimal(overtimeMinutes)}h
            </span>
          </div>
          <p className="text-xs text-muted-foreground mt-1.5">
            {balance?.periodStart
              ? t('profil.zeiterfassung.overview.since', { date: balance.periodStart })
              : t('profil.zeiterfassung.overtime')}
          </p>
        </button>

        {/* Vacation placeholder — real data from leave balance (separate hook) */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {t('profil.zeiterfassung.overview.remainingVacation')}
            </span>
            <Palmtree className="h-4 w-4 text-success" />
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className="text-2xl font-bold text-foreground tabular-nums">—</span>
          </div>
          <p className="text-xs text-muted-foreground mt-1.5">
            {t('profil.zeiterfassung.overview.seeAbsences')}
          </p>
        </div>
      </div>

      {/* Week at a Glance */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-foreground">{t('profil.zeiterfassung.overview.weekAtGlance')}</h3>
            <button
              onClick={() => onNavigate('week')}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              {t('common.details')} <ArrowRight className="h-3 w-3" />
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
            <span>{t('profil.zeiterfassung.overview.target')}: {formatHoursDecimal(weeklyTarget)}h</span>
            <span>|</span>
            <span className={cn('font-medium', weekMinutes >= weeklyTarget ? 'text-success' : 'text-foreground')}>
              {t('profil.zeiterfassung.overview.actual')}: {formatHoursDecimal(weekMinutes)}h
            </span>
          </div>
        </div>

        {/* Recent Activity */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-foreground">{t('profil.zeiterfassung.overview.recentEntries')}</h3>
            <button
              onClick={() => onNavigate('today')}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              {t('profil.zeiterfassung.overview.all')} <ArrowRight className="h-3 w-3" />
            </button>
          </div>
          {recentEntries.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">{t('profil.zeiterfassung.noEntries')}</p>
          ) : (
            <div className="space-y-2">
              {recentEntries.map((entry) => {
                const entryDate = entry.clockIn?.slice(0, 10) ?? ''
                const entryIsToday = isToday(entryDate)
                return (
                  <div key={entry.id} className="flex items-center gap-2.5 py-1.5">
                    <span className="h-2 w-2 rounded-full shrink-0 bg-primary/50" />
                    <span className="text-xs text-muted-foreground w-16 shrink-0 tabular-nums">
                      {entryIsToday ? t('profil.zeiterfassung.viewToday') : formatDateLabel(entryDate)}
                    </span>
                    <span className="text-sm text-foreground truncate flex-1">
                      {entry.activity ?? entry.note ?? t('profil.zeiterfassung.unknown')}
                    </span>
                    <span className="text-xs text-muted-foreground tabular-nums shrink-0">
                      {formatMinutes(entry.netWorkMinutes ?? 0)}
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
  if (dateStr === yesterday.toISOString().split('T')[0]) return i18next.t('profil.zeiterfassung.overview.yesterday')
  return `${d.getDate().toString().padStart(2, '0')}.${(d.getMonth() + 1).toString().padStart(2, '0')}.`
}
