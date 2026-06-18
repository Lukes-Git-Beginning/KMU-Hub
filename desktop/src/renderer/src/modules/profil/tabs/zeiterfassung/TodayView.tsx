import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Clock, FolderKanban, Palmtree, ThermometerSun, Home, BookOpen, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import {
  useWorkTimeEntries,
  useDailySummary,
  useActiveShift,
  useWorkTimeStatus,
  useTimeProjects,
  useTimeCategories,
} from '@/api/hooks/hr-hooks'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, isToday, todayStr } from './time-utils'
import ManualEntryForm from './ManualEntryForm'
import type { WorkTimeEntry } from '@/api/hr-types'

const ABSENCE_ICONS: Record<string, typeof Palmtree> = {
  vacation: Palmtree,
  sick: ThermometerSun,
  homeoffice: Home,
  education: BookOpen,
  other: AlertTriangle,
}

export default function TodayView() {
  const { t } = useTranslation()
  const [showManualForm, setShowManualForm] = useState(false)

  // API hooks
  const { data: entriesData } = useWorkTimeEntries()
  const allEntries = entriesData?.entries ?? []
  const { data: status } = useWorkTimeStatus()
  const { data: activeShift } = useActiveShift()
  const { data: dailySummary } = useDailySummary(todayStr())
  const { data: projects = [] } = useTimeProjects()
  const { data: categories = [] } = useTimeCategories()

  const elapsed = useTimerTick()

  // Filter today's entries (Wire-Shape: clockIn is ISO timestamp)
  const todayEntries = useMemo(
    () =>
      (allEntries as WorkTimeEntry[])
        .filter((e) => {
          const date = e.clockIn?.slice(0, 10) ?? ''
          return isToday(date)
        })
        .sort((a, b) => (a.clockIn ?? '').localeCompare(b.clockIn ?? '')),
    [allEntries],
  )

  const totalMinutes = dailySummary?.netWorkMinutes ?? todayEntries.reduce((s, e) => s + (e.netWorkMinutes ?? 0), 0)
  // 480 = 8h default; real target comes from HRSettings (separate follow-up)
  const targetMinutes = 480
  const progressPercent = Math.min(100, Math.round((totalMinutes / targetMinutes) * 100))
  const dailyOvertime = totalMinutes - targetMinutes

  const getProjectName = (projectId: string | null | undefined) =>
    projectId ? projects.find((p) => p.id === projectId)?.name : null

  const isClockedIn = status?.isClockedIn ?? false
  const isOnBreak = status?.isOnBreak ?? false
  const activeColor = categories[0]?.color ?? '#6366f1'

  return (
    <div className="p-6 space-y-4 max-w-3xl mx-auto">
      {/* Running Shift Banner */}
      {isClockedIn && (
        <div className="rounded-xl border-2 border-primary/50 bg-primary/5 p-4 animate-in fade-in">
          <div className="flex items-center gap-3">
            <div className="relative">
              <span
                className="h-3 w-3 rounded-full inline-block"
                style={{ backgroundColor: activeColor }}
              />
              {!isOnBreak && (
                <span
                  className="absolute inset-0 h-3 w-3 rounded-full animate-ping opacity-40"
                  style={{ backgroundColor: activeColor }}
                />
              )}
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-foreground">
                {t('profil.zeiterfassung.today.activeShift')}
                {activeShift?.shift?.total_break_minutes ? (
                  <span className="text-muted-foreground ml-2">
                    — {t('profil.zeiterfassung.today.breakMinutes', { count: activeShift.shift.total_break_minutes })}
                  </span>
                ) : null}
              </p>
            </div>
            <span className="text-xl font-mono font-bold text-primary tabular-nums">
              {formatElapsed(elapsed)}
            </span>
            <span className={cn(
              'px-2 py-0.5 rounded-full text-[10px] font-semibold uppercase tracking-wider',
              isOnBreak
                ? 'bg-warning-light text-warning-foreground'
                : 'bg-success-light text-success',
            )}>
              {isOnBreak ? t('profil.zeiterfassung.today.paused') : t('profil.zeiterfassung.today.running')}
            </span>
          </div>
        </div>
      )}

      {/* Entry List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-foreground">
            {t('profil.zeiterfassung.today.entriesToday', { count: todayEntries.length })}
          </h3>
          <Button size="sm" variant="outline" onClick={() => setShowManualForm(true)} className="gap-2">
            <Plus className="h-3.5 w-3.5" />
            {t('profil.zeiterfassung.today.manualEntry')}
          </Button>
        </div>

        {todayEntries.length === 0 && !isClockedIn && (
          <div className="text-center py-12 text-muted-foreground">
            <Clock className="h-10 w-10 mx-auto mb-3 opacity-50" />
            <p className="font-medium">{t('profil.zeiterfassung.noEntriesToday')}</p>
            <p className="text-sm">{t('profil.zeiterfassung.today.startTimerOrManual')}</p>
          </div>
        )}

        {todayEntries.map((entry) => {
          const projectName = getProjectName(entry.projectId)
          const clockInTime = entry.clockIn?.slice(11, 16) ?? ''
          const clockOutTime = entry.clockOut?.slice(11, 16) ?? '...'
          const netMins = entry.netWorkMinutes ?? 0
          return (
            <div
              key={entry.id}
              className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card hover:border-primary/30 transition-colors group"
            >
              {/* Time */}
              <div className="text-sm font-mono text-muted-foreground w-28 shrink-0 tabular-nums">
                {clockInTime} - {clockOutTime}
              </div>

              {/* Activity / Note */}
              <div className="flex-1 min-w-0">
                <span className="text-sm text-muted-foreground truncate block">
                  {entry.activity ?? entry.note ?? t('profil.zeiterfassung.unknown')}
                </span>
                {projectName && (
                  <span className="flex items-center gap-1.5 text-[11px] text-primary/70 mt-0.5">
                    <FolderKanban className="h-3 w-3 shrink-0" />
                    {entry.customerName && (
                      <span className="text-muted-foreground">{entry.customerName} /</span>
                    )}
                    {projectName}
                  </span>
                )}
              </div>

              {/* Billable */}
              {entry.billable && (
                <span className="text-[10px] text-muted-foreground bg-secondary px-1.5 py-0.5 rounded shrink-0">
                  {t('profil.zeiterfassung.today.billable')}
                </span>
              )}

              {/* Duration */}
              <span className="text-sm font-medium text-foreground w-16 text-right shrink-0 tabular-nums">
                {formatMinutes(netMins)}
              </span>

              {/* Manual marker */}
              {entry.isManual && (
                <span className="text-[10px] text-muted-foreground bg-secondary px-1.5 py-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity">
                  {t('profil.zeiterfassung.today.manual')}
                </span>
              )}
            </div>
          )
        })}
      </div>

      {/* Soll/Ist Footer */}
      <div className="rounded-xl border border-border bg-card p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-muted-foreground">
            {t('profil.zeiterfassung.overview.target')} / {t('profil.zeiterfassung.overview.actual')}
          </span>
          <div className="flex items-center gap-3">
            <span className="text-sm font-medium text-foreground">
              {formatMinutes(totalMinutes)} / {formatMinutes(targetMinutes)}
            </span>
            <span className={cn(
              'text-xs font-semibold px-2 py-0.5 rounded-full',
              dailyOvertime >= 0
                ? 'bg-success-light text-success'
                : 'bg-warning-light text-warning-foreground',
            )}>
              {dailyOvertime >= 0 ? '+' : ''}{formatMinutes(dailyOvertime)}
            </span>
          </div>
        </div>
        <div className="w-full h-3 rounded-full bg-secondary overflow-hidden">
          <div
            className={cn(
              'h-full rounded-full transition-all duration-500',
              progressPercent >= 90 ? 'bg-success' : progressPercent >= 60 ? 'bg-warning' : 'bg-destructive',
            )}
            style={{ width: `${progressPercent}%` }}
          />
        </div>
        {totalMinutes > targetMinutes && (
          <p className="text-xs text-success mt-1">
            +{formatMinutes(totalMinutes - targetMinutes)} {t('profil.zeiterfassung.overtime')}
          </p>
        )}
        {totalMinutes < targetMinutes && totalMinutes > 0 && (
          <p className="text-xs text-muted-foreground mt-1">
            {t('profil.zeiterfassung.overview.remaining')} {formatMinutes(targetMinutes - totalMinutes)}{' '}
            {t('profil.zeiterfassung.today.untilTarget')}
          </p>
        )}
      </div>

      <ManualEntryForm open={showManualForm} onOpenChange={setShowManualForm} />
    </div>
  )
}
