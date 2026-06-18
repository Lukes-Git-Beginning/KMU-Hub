import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Plus, Trash2, Clock, MapPin, FolderKanban,
  Palmtree, ThermometerSun, Home, BookOpen, AlertTriangle,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip'
import { cn } from '@/lib/cn'
import {
  useWorkTimeEntries, useTimeCategories, useDeleteTimeEntry, useTimeProjects,
  useWorkTimeStatus, useAbsenceCalendar, useDailySummary,
} from '@/api/hooks/hr-hooks'
import { adaptWorkTimeEntryToFE } from '@/api/hr-client'
import { formatMinutes, todayStr } from './time-utils'
import ManualEntryForm from './ManualEntryForm'

const ABSENCE_ICONS: Record<string, typeof Palmtree> = {
  vacation: Palmtree,
  urlaub: Palmtree,
  sick: ThermometerSun,
  krankheit: ThermometerSun,
  homeoffice: Home,
  education: BookOpen,
  other: AlertTriangle,
}

export default function TodayView() {
  const { t } = useTranslation()
  const [showManualForm, setShowManualForm] = useState(false)

  const ABSENCE_LABELS: Record<string, string> = {
    vacation: t('profil.zeiterfassung.absenceType.vacation'),
    urlaub: t('profil.zeiterfassung.absenceType.vacation'),
    sick: t('profil.zeiterfassung.absenceType.sick'),
    krankheit: t('profil.zeiterfassung.absenceType.sick'),
    homeoffice: t('profil.zeiterfassung.absenceType.homeoffice'),
    education: t('profil.zeiterfassung.absenceType.education'),
    other: t('profil.zeiterfassung.absenceType.other'),
  }

  const { data: entriesRaw, isLoading } = useWorkTimeEntries({ start_date: todayStr(), end_date: todayStr() })
  const { data: categories = [] } = useTimeCategories()
  const { data: projects = [] } = useTimeProjects()
  const { data: status } = useWorkTimeStatus()
  const { data: dailySummary } = useDailySummary(todayStr())
  const { data: absences = [] } = useAbsenceCalendar({ start_date: todayStr(), end_date: todayStr() })
  const deleteEntry = useDeleteTimeEntry()

  const todayEntries = useMemo(
    () =>
      (entriesRaw?.entries ?? [])
        .map(adaptWorkTimeEntryToFE)
        .sort((a, b) => a.startTime.localeCompare(b.startTime)),
    [entriesRaw],
  )

  const totalMinutes = dailySummary?.netWorkMinutes ?? todayEntries.reduce((sum, e) => sum + e.durationMinutes, 0)
  const targetMinutes = 8.4 * 60
  const progressPercent = Math.min(100, Math.round((totalMinutes / targetMinutes) * 100))
  const dailyOvertime = totalMinutes - targetMinutes

  const todayAbsence = useMemo(() => {
    const today = todayStr()
    return absences.find((a) => today >= a.startDate && today <= a.endDate)
  }, [absences])

  const getCategoryInfo = (catId: string | null) =>
    catId ? categories.find((c) => c.id === catId) : undefined

  const getProjectName = (projectId: string | null) =>
    projectId ? projects.find((p) => p.id === projectId)?.name : null

  return (
    <div className="p-6 space-y-4 max-w-3xl mx-auto">
      {/* Absence Banner */}
      {todayAbsence && (() => {
        const key = todayAbsence.leaveTypeKey?.toLowerCase() ?? 'other'
        const AbsIcon = ABSENCE_ICONS[key] || AlertTriangle
        const label = ABSENCE_LABELS[key] || todayAbsence.leaveTypeName
        const isVacation = key === 'vacation' || key === 'urlaub'
        const isSick = key === 'sick' || key === 'krankheit'
        return (
          <div className={cn(
            'rounded-lg border p-3 flex items-center gap-3',
            isVacation ? 'border-warning/30 bg-warning-light' :
            isSick ? 'border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-900/30' :
            'border-blue-300 dark:border-blue-700 bg-blue-50 dark:bg-blue-950/30',
          )}>
            <AbsIcon className={cn(
              'h-5 w-5 shrink-0',
              isVacation ? 'text-warning-foreground' :
              isSick ? 'text-gray-500' :
              'text-blue-500 dark:text-blue-400',
            )} />
            <span className="text-sm font-medium text-foreground">
              Heute: {label}
            </span>
            {(isVacation || isSick) && (
              <span className="text-xs text-muted-foreground ml-auto">
                {t('profil.zeiterfassung.today.timerLocked')}
              </span>
            )}
          </div>
        )
      })()}

      {/* Running Timer Banner */}
      {status?.isClockedIn && !status.isOnBreak && (
        <div className="rounded-xl border-2 border-primary/50 bg-primary/5 p-4 animate-in fade-in">
          <div className="flex items-center gap-3">
            <div className="relative">
              <span className="h-3 w-3 rounded-full inline-block bg-primary" />
              <span className="absolute inset-0 h-3 w-3 rounded-full animate-ping opacity-40 bg-primary" />
            </div>
            <p className="text-sm font-medium text-foreground flex-1">
              {t('profil.zeiterfassung.today.running')}
            </p>
          </div>
        </div>
      )}
      {status?.isOnBreak && (
        <div className="rounded-xl border-2 border-warning/50 bg-warning-light p-3 text-sm text-warning-foreground">
          {t('profil.zeiterfassung.break')}
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

        {isLoading && (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-12 rounded-lg border border-border bg-card animate-pulse" />
            ))}
          </div>
        )}

        {!isLoading && todayEntries.length === 0 && !status?.isClockedIn && (
          <div className="text-center py-12 text-muted-foreground">
            <Clock className="h-10 w-10 mx-auto mb-3 opacity-50" />
            <p className="font-medium">{t('profil.zeiterfassung.noEntriesToday')}</p>
            <p className="text-sm">{t('profil.zeiterfassung.today.startTimerOrManual')}</p>
          </div>
        )}

        {!isLoading && todayEntries.map((entry) => {
          const cat = getCategoryInfo(entry.categoryId)
          const projectName = getProjectName(entry.projectId)
          return (
            <div
              key={entry.id}
              className="flex items-center gap-3 p-3 rounded-lg border border-border bg-card hover:border-primary/30 transition-colors group"
            >
              {/* Time */}
              <div className="text-sm font-mono text-muted-foreground w-28 shrink-0 tabular-nums">
                {entry.startTime} - {entry.endTime || '...'}
              </div>

              {/* Category */}
              <span
                className="h-2.5 w-2.5 rounded-full shrink-0"
                style={{ backgroundColor: cat?.color || '#6b7280' }}
              />
              <span className="text-sm font-medium text-foreground w-32 truncate shrink-0">
                {cat?.name || t('profil.zeiterfassung.unknown')}
              </span>

              {/* Description + Project */}
              <div className="flex-1 min-w-0">
                <span className="text-sm text-muted-foreground truncate block">
                  {entry.description}
                </span>
                {projectName && (
                  <span className="flex items-center gap-1.5 text-[11px] text-primary/70 mt-0.5">
                    <FolderKanban className="h-3 w-3 shrink-0" />
                    {projectName}
                  </span>
                )}
              </div>

              {/* GPS Location */}
              {entry.location !== null && (
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="shrink-0 text-muted-foreground hover:text-primary transition-colors cursor-default">
                        <MapPin className="h-3.5 w-3.5" />
                      </span>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p className="text-xs">{entry.location.address}</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
              )}

              {/* Duration */}
              <span className="text-sm font-medium text-foreground w-16 text-right shrink-0 tabular-nums">
                {formatMinutes(entry.durationMinutes)}
              </span>

              {/* Actions */}
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                {entry.isManual && (
                  <span className="text-[10px] text-muted-foreground bg-secondary px-1.5 py-0.5 rounded">
                    manuell
                  </span>
                )}
                <button
                  onClick={() => deleteEntry.mutate(entry.id)}
                  className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          )
        })}
      </div>

      {/* Soll/Ist Footer */}
      <div className="rounded-xl border border-border bg-card p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-muted-foreground">{t('profil.zeiterfassung.overview.target')} / {t('profil.zeiterfassung.overview.actual')}</span>
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
            {t('profil.zeiterfassung.overview.remaining')} {formatMinutes(targetMinutes - totalMinutes)} {t('profil.zeiterfassung.today.untilTarget')}
          </p>
        )}
      </div>

      {/* Manual Entry Form Dialog */}
      <ManualEntryForm open={showManualForm} onOpenChange={setShowManualForm} />
    </div>
  )
}
