import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import i18next from 'i18next'
import {
  Clock, TrendingUp, TrendingDown, Palmtree, Calendar,
  ArrowRight, Timer,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import {
  useWorkTimeStatus, useDailySummary, useWeeklySummary, useTimeBalance,
  useLeaveBalance, useWorkTimeEntries, useTimeCategories,
} from '@/api/hooks/hr-hooks'
import { adaptWorkTimeEntryToFE } from '@/api/hr-client'
import { formatMinutes, formatHoursDecimal, isToday, getWeekDates, dateToStr } from './time-utils'

interface OverviewViewProps {
  onNavigate: (view: string) => void
}

export default function OverviewView({ onNavigate }: OverviewViewProps) {
  const { t } = useTranslation()

  const today = useMemo(() => new Date().toISOString().split('T')[0], [])
  const weekDates = useMemo(() => getWeekDates(0), [])
  const weekStartStr = useMemo(() => dateToStr(weekDates[0]), [weekDates])
  const weekEndStr = useMemo(() => dateToStr(weekDates[6]), [weekDates])

  const { data: workStatus } = useWorkTimeStatus()
  const { data: dailySummary } = useDailySummary(today)
  const { data: weeklySummary } = useWeeklySummary(weekStartStr)
  const { data: balance } = useTimeBalance()
  const { data: leaveBalance } = useLeaveBalance()
  const { data: categories = [] } = useTimeCategories()
  const { data: recentRaw } = useWorkTimeEntries({ limit: 5 })
  const { data: weekEntriesRaw } = useWorkTimeEntries({ start_date: weekStartStr, end_date: weekEndStr })

  // ── Today ───────────────────────────────────────────
  const todayMinutes = dailySummary?.netWorkMinutes ?? 0
  const dailyTarget = 8.4 * 60
  const todayPercent = Math.min(100, Math.round((todayMinutes / dailyTarget) * 100))

  // ── This Week ───────────────────────────────────────
  const weekMinutes = weeklySummary?.netWorkMinutes ?? 0
  const weekTarget = 5 * 8.4 * 60
  const weekPercent = Math.min(100, Math.round((weekMinutes / weekTarget) * 100))

  // ── Overtime ────────────────────────────────────────
  const overtime = balance?.balanceMinutes ?? 0

  // ── Vacation ────────────────────────────────────────
  const vacationTotal = leaveBalance?.totalEntitlement ?? 25
  const vacationUsed = leaveBalance?.used ?? 0
  const vacationRemaining = leaveBalance?.remaining ?? (vacationTotal - vacationUsed)

  // ── Recent entries ──────────────────────────────────
  const recentEntries = useMemo(
    () => (recentRaw?.entries ?? []).map(adaptWorkTimeEntryToFE),
    [recentRaw],
  )

  const weekEntries = useMemo(
    () => (weekEntriesRaw?.entries ?? []).map(adaptWorkTimeEntryToFE),
    [weekEntriesRaw],
  )

  // ── Category breakdown this week ────────────────────
  const weekCategoryStats = useMemo(() => {
    const stats: Record<string, number> = {}
    for (const e of weekEntries) {
      if (e.categoryId) stats[e.categoryId] = (stats[e.categoryId] || 0) + e.durationMinutes
    }
    return categories
      .map((cat) => ({ ...cat, minutes: stats[cat.id] || 0 }))
      .filter((c) => c.minutes > 0)
      .sort((a, b) => b.minutes - a.minutes)
  }, [weekEntries, categories])

  // ── Daily bar chart ─────────────────────────────────
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
    const dayMap: Record<string, number> = {}
    if (weeklySummary?.days) {
      for (const d of weeklySummary.days) dayMap[d.date] = d.netWorkMinutes
    } else {
      for (const e of weekEntries) dayMap[e.date] = (dayMap[e.date] || 0) + e.durationMinutes
    }
    return getWeekDates(0).map((date, i) => {
      const ds = dateToStr(date)
      return { day: dayNames[i], date: ds, minutes: dayMap[ds] ?? 0, isToday: isToday(ds) }
    })
  }, [weeklySummary, weekEntries, t])

  const getCat = (id: string | null) => id ? categories.find((c) => c.id === id) : undefined

  // Active shift banner
  const isActiveShift = workStatus?.isClockedIn && workStatus.currentShiftStart

  return (
    <div className="p-6 space-y-6 max-w-4xl mx-auto">
      {/* Active Timer Banner */}
      {isActiveShift && (
        <div className="rounded-xl border-2 border-primary/50 bg-primary/5 p-4">
          <div className="flex items-center gap-3">
            <div className="relative">
              <Timer className="h-5 w-5 text-primary" />
              <span className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-success animate-pulse" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-foreground">
                {workStatus.isOnBreak
                  ? t('profil.zeiterfassung.break')
                  : t('profil.zeiterfassung.active')}
              </p>
            </div>
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
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('profil.zeiterfassung.viewToday')}</span>
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
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('profil.zeiterfassung.overview.thisWeek')}</span>
            <Calendar className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors" />
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className="text-2xl font-bold text-foreground tabular-nums">
              {formatHoursDecimal(weekMinutes)}
            </span>
            <span className="text-sm text-muted-foreground mb-0.5">/ {formatHoursDecimal(weekTarget)}h</span>
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
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('profil.zeiterfassung.overtime')}</span>
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
          {balance?.periodStart && (
            <p className="text-xs text-muted-foreground mt-1.5">
              {t('profil.zeiterfassung.overview.target')}: {new Date(balance.periodStart).toLocaleDateString('de-DE')}
            </p>
          )}
        </button>

        {/* Vacation */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">{t('profil.zeiterfassung.overview.remainingVacation')}</span>
            <Palmtree className="h-4 w-4 text-success" />
          </div>
          <div className="flex items-end gap-1.5 mb-2">
            <span className="text-2xl font-bold text-foreground tabular-nums">{vacationRemaining}</span>
            <span className="text-sm text-muted-foreground mb-0.5">/ {vacationTotal} {t('profil.zeiterfassung.overview.days')}</span>
          </div>
          <div className="w-full h-2 rounded-full bg-secondary overflow-hidden">
            <div
              className="h-full rounded-full bg-success transition-all"
              style={{ width: `${Math.round((vacationUsed / vacationTotal) * 100)}%` }}
            />
          </div>
          <div className="flex items-center gap-3 mt-1.5">
            <span className="text-xs text-muted-foreground">{vacationUsed} {t('profil.zeiterfassung.overview.taken')}</span>
          </div>
        </div>
      </div>

      {/* Week at a Glance + Category Breakdown */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Daily Bar Chart */}
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
            <span>{t('profil.zeiterfassung.overview.target')}: {formatHoursDecimal(weekTarget)}h</span>
            <span>|</span>
            <span className={cn(
              'font-medium',
              weekMinutes >= weekTarget ? 'text-success' : 'text-foreground',
            )}>
              {t('profil.zeiterfassung.overview.actual')}: {formatHoursDecimal(weekMinutes)}h
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
            <h3 className="text-sm font-semibold text-foreground">{t('profil.zeiterfassung.overview.categoriesThisWeek')}</h3>
            <button
              onClick={() => onNavigate('categories')}
              className="text-xs text-primary hover:underline flex items-center gap-1"
            >
              {t('profil.zeiterfassung.overview.manage')} <ArrowRight className="h-3 w-3" />
            </button>
          </div>
          {weekCategoryStats.length === 0 ? (
            <p className="text-sm text-muted-foreground py-8 text-center">{t('profil.zeiterfassung.overview.noEntriesThisWeek')}</p>
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
          <h3 className="text-sm font-semibold text-foreground mb-4">{t('profil.zeiterfassung.overview.contract')}</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.overview.weeklyTarget')}</span>
              <span className="font-medium text-foreground">{formatHoursDecimal(weekTarget)}h</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.overview.dailyTarget')}</span>
              <span className="font-medium text-foreground">{formatHoursDecimal(dailyTarget)}h</span>
            </div>
            <div className="border-t border-border pt-3 flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.overview.thisMonthActual')}</span>
              <span className="font-semibold text-foreground">{formatHoursDecimal(weekMinutes)}h</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{t('profil.zeiterfassung.overview.vacationEntitlement')}</span>
              <span className="font-medium text-foreground">{vacationTotal} {t('profil.zeiterfassung.overview.daysPerYear')}</span>
            </div>
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
                      {entryIsToday ? t('profil.zeiterfassung.viewToday') : formatDateLabel(entry.date)}
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

  if (dateStr === yesterday.toISOString().split('T')[0]) return i18next.t('profil.zeiterfassung.overview.yesterday')

  return `${d.getDate().toString().padStart(2, '0')}.${(d.getMonth() + 1).toString().padStart(2, '0')}.`
}
