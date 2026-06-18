import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { useWorkTimeEntries } from '@/api/hooks/hr-hooks'
import {
  getMonthDates, dateToStr, getMonthLabel, formatMinutes, isToday,
} from './time-utils'
import type { WorkTimeEntry } from '@/api/hr-types'

export default function MonthView() {
  const { t } = useTranslation()
  const [monthOffset, setMonthOffset] = useState(0)

  const monthDates = useMemo(() => getMonthDates(monthOffset), [monthOffset])
  const dailyTarget = 480 // 8h default; real target from HRSettings (follow-up)

  // Build date range for API query
  const startDate = useMemo(() => dateToStr(monthDates[0]), [monthDates])
  const endDate = useMemo(() => dateToStr(monthDates[monthDates.length - 1]), [monthDates])

  const { data: entriesData } = useWorkTimeEntries({
    start_date: startDate,
    end_date: endDate,
  })
  const allEntries = entriesData?.entries ?? []

  // Minutes per day
  const dayData = useMemo(() => {
    const data: Record<string, { total: number }> = {}
    for (const d of monthDates) {
      data[dateToStr(d)] = { total: 0 }
    }
    for (const entry of allEntries as WorkTimeEntry[]) {
      const date = entry.clockIn?.slice(0, 10) ?? ''
      if (data[date]) {
        data[date].total += entry.netWorkMinutes ?? 0
      }
    }
    return data
  }, [allEntries, monthDates])

  const daysWithEntries = Object.values(dayData).filter((d) => d.total > 0).length
  const totalMinutes = Object.values(dayData).reduce((s, d) => s + d.total, 0)
  const avgPerDay = daysWithEntries > 0 ? Math.round(totalMinutes / daysWithEntries) : 0
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
            {t('profil.zeiterfassung.month.current')}
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
            const total = dayData[ds]?.total ?? 0
            const isWeekend = date.getDay() === 0 || date.getDay() === 6
            const barHeight = total > 0 ? Math.max(4, (total / maxMinutesInDay) * 180) : 0
            const targetLine = (dailyTarget / maxMinutesInDay) * 180

            return (
              <div
                key={ds}
                className="flex-1 relative group"
                style={{ height: '100%' }}
              >
                {!isWeekend && (
                  <div
                    className="absolute left-0 right-0 border-t border-dashed border-muted-foreground/30"
                    style={{ bottom: targetLine }}
                  />
                )}
                <div
                  className={cn(
                    'absolute bottom-0 left-[1px] right-[1px] rounded-t-sm transition-all',
                    total >= dailyTarget && !isWeekend ? 'bg-success' : total > 0 ? 'bg-primary/60' : 'bg-transparent',
                  )}
                  style={{ height: barHeight }}
                />
                <div className={cn(
                  'absolute -bottom-5 left-0 right-0 text-center text-[9px]',
                  isToday(ds) ? 'text-primary font-bold' : isWeekend ? 'text-muted-foreground/40' : 'text-muted-foreground',
                )}>
                  {date.getDate()}
                </div>

                {/* Tooltip */}
                <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block z-10 pointer-events-none">
                  <div className="bg-popover border border-border text-popover-foreground text-xs rounded-lg px-3 py-2 shadow-lg whitespace-nowrap">
                    <p className="font-medium">{date.getDate()}. {getMonthLabel(monthOffset).split(' ')[0]}</p>
                    <p>{total > 0 ? formatMinutes(total) : t('profil.zeiterfassung.noEntries')}</p>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Month Summary */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">{t('profil.zeiterfassung.total')}</p>
          <p className="text-xl font-bold text-foreground mt-1">{formatMinutes(totalMinutes)}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">{t('profil.zeiterfassung.month.avgPerDay')}</p>
          <p className="text-xl font-bold text-foreground mt-1">{formatMinutes(avgPerDay)}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">{t('profil.zeiterfassung.month.targetMonth')}</p>
          <p className="text-xl font-bold text-foreground mt-1">{formatMinutes(monthTarget)}</p>
        </div>
        <div className="rounded-xl border border-border bg-card p-4">
          <p className="text-xs text-muted-foreground">{t('profil.zeiterfassung.balance')}</p>
          <p className={cn(
            'text-xl font-bold mt-1',
            totalMinutes >= monthTarget ? 'text-success' : 'text-warning-foreground',
          )}>
            {totalMinutes >= monthTarget ? '+' : ''}{formatMinutes(totalMinutes - monthTarget)}
          </p>
        </div>
      </div>
    </div>
  )
}
