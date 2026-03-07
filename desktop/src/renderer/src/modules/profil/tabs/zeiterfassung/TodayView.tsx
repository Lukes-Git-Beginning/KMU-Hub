import { useState, useMemo } from 'react'
import { Plus, Trash2, Clock, MapPin, FolderKanban, Palmtree, ThermometerSun, Home, BookOpen, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore, MOCK_PROJECTS, MOCK_TASKS, getOvertimeForDay } from '@/stores/timetracking'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, isToday, todayStr } from './time-utils'
import ManualEntryForm from './ManualEntryForm'

const ABSENCE_LABELS: Record<string, string> = {
  vacation: 'Urlaub',
  sick: 'Krank',
  homeoffice: 'Homeoffice',
  education: 'Weiterbildung',
  other: 'Sonstiges',
}

const ABSENCE_ICONS: Record<string, typeof Palmtree> = {
  vacation: Palmtree,
  sick: ThermometerSun,
  homeoffice: Home,
  education: BookOpen,
  other: AlertTriangle,
}

export default function TodayView() {
  const [showManualForm, setShowManualForm] = useState(false)
  const [_editingId, _setEditingId] = useState<string | null>(null)

  const entries = useTimeTrackingStore((s) => s.entries)
  const categories = useTimeTrackingStore((s) => s.categories)
  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const targets = useTimeTrackingStore((s) => s.targets)
  const absences = useTimeTrackingStore((s) => s.absences)
  const deleteEntry = useTimeTrackingStore((s) => s.deleteEntry)
  const elapsed = useTimerTick()

  // Check today's absence (6.10)
  const todayAbsence = useMemo(() => {
    const today = todayStr()
    return absences.find((a) => {
      if (a.status !== 'approved') return false
      return today >= a.startDate && today <= a.endDate
    })
  }, [absences])

  const todayEntries = useMemo(
    () =>
      entries
        .filter((e) => isToday(e.date))
        .sort((a, b) => a.startTime.localeCompare(b.startTime)),
    [entries],
  )

  const totalMinutes = todayEntries.reduce((sum, e) => sum + e.durationMinutes, 0)
  const targetMinutes = targets.dailyHours * 60
  const progressPercent = Math.min(100, Math.round((totalMinutes / targetMinutes) * 100))

  // Daily overtime (6.8)
  const dailyOvertime = getOvertimeForDay(entries, todayStr(), targetMinutes)

  const getCategoryInfo = (catId: string) =>
    categories.find((c) => c.id === catId)

  const getProjectName = (projectId: string | null) =>
    projectId ? MOCK_PROJECTS.find((p) => p.id === projectId)?.name : null

  const getTaskLabel = (taskId: string | null) => {
    if (!taskId) return null
    const task = MOCK_TASKS.find((t) => t.id === taskId)
    return task ? `${task.key}: ${task.title}` : null
  }

  return (
    <div className="p-6 space-y-4 max-w-3xl mx-auto">
      {/* Absence Banner (6.10) */}
      {todayAbsence && (() => {
        const AbsIcon = ABSENCE_ICONS[todayAbsence.type] || AlertTriangle
        return (
          <div className={cn(
            'rounded-lg border p-3 flex items-center gap-3',
            todayAbsence.type === 'vacation' ? 'border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-950/30' :
            todayAbsence.type === 'sick' ? 'border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-900/30' :
            'border-blue-300 dark:border-blue-700 bg-blue-50 dark:bg-blue-950/30',
          )}>
            <AbsIcon className={cn(
              'h-5 w-5 shrink-0',
              todayAbsence.type === 'vacation' ? 'text-amber-600 dark:text-amber-400' :
              todayAbsence.type === 'sick' ? 'text-gray-500' :
              'text-blue-500 dark:text-blue-400',
            )} />
            <span className="text-sm font-medium text-foreground">
              Heute: {ABSENCE_LABELS[todayAbsence.type]}
            </span>
            {(todayAbsence.type === 'vacation' || todayAbsence.type === 'sick') && (
              <span className="text-xs text-muted-foreground ml-auto">
                Timer gesperrt
              </span>
            )}
          </div>
        )
      })()}

      {/* Running Timer */}
      {activeTimer.status !== 'idle' && activeTimer.categoryId && (
        <div className="rounded-xl border-2 border-primary/50 bg-primary/5 p-4 animate-in fade-in">
          <div className="flex items-center gap-3">
            <div className="relative">
              <span
                className="h-3 w-3 rounded-full inline-block"
                style={{ backgroundColor: getCategoryInfo(activeTimer.categoryId)?.color }}
              />
              {activeTimer.status === 'running' && (
                <span className="absolute inset-0 h-3 w-3 rounded-full animate-ping opacity-40"
                  style={{ backgroundColor: getCategoryInfo(activeTimer.categoryId)?.color }}
                />
              )}
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-foreground">
                {getCategoryInfo(activeTimer.categoryId)?.name}
                {activeTimer.description && (
                  <span className="text-muted-foreground ml-2">— {activeTimer.description}</span>
                )}
              </p>
            </div>
            <span className="text-xl font-mono font-bold text-primary tabular-nums">
              {formatElapsed(elapsed)}
            </span>
            <span className={cn(
              'px-2 py-0.5 rounded-full text-[10px] font-semibold uppercase tracking-wider',
              activeTimer.status === 'running'
                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
            )}>
              {activeTimer.status === 'running' ? 'Läuft' : 'Pausiert'}
            </span>
          </div>
        </div>
      )}

      {/* Entry List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-foreground">
            Einträge heute ({todayEntries.length})
          </h3>
          <Button size="sm" variant="outline" onClick={() => setShowManualForm(true)} className="gap-2">
            <Plus className="h-3.5 w-3.5" />
            Manuell eintragen
          </Button>
        </div>

        {todayEntries.length === 0 && activeTimer.status === 'idle' && (
          <div className="text-center py-12 text-muted-foreground">
            <Clock className="h-10 w-10 mx-auto mb-3 opacity-50" />
            <p className="font-medium">Noch keine Einträge heute</p>
            <p className="text-sm">Starte den Timer oder erstelle einen manuellen Eintrag</p>
          </div>
        )}

        {todayEntries.map((entry) => {
          const cat = getCategoryInfo(entry.categoryId)
          const projectName = getProjectName(entry.projectId)
          const taskLabel = getTaskLabel(entry.taskId)
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
                {cat?.name || 'Unbekannt'}
              </span>

              {/* Description + Project/Task (6.6) */}
              <div className="flex-1 min-w-0">
                <span className="text-sm text-muted-foreground truncate block">
                  {entry.description}
                </span>
                {(projectName || taskLabel) && (
                  <span className="flex items-center gap-1.5 text-[11px] text-primary/70 mt-0.5">
                    <FolderKanban className="h-3 w-3 shrink-0" />
                    {projectName}
                    {taskLabel && <span className="text-muted-foreground">/ {taskLabel}</span>}
                  </span>
                )}
              </div>

              {/* GPS Location (6.11) */}
              {entry.location && (
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
                  onClick={() => deleteEntry(entry.id)}
                  className="p-1 rounded hover:bg-destructive/10 text-muted-foreground hover:text-destructive transition-colors"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          )
        })}
      </div>

      {/* Soll/Ist Footer with Overtime (6.8) */}
      <div className="rounded-xl border border-border bg-card p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-muted-foreground">Soll / Ist</span>
          <div className="flex items-center gap-3">
            <span className="text-sm font-medium text-foreground">
              {formatMinutes(totalMinutes)} / {formatMinutes(targetMinutes)}
            </span>
            <span className={cn(
              'text-xs font-semibold px-2 py-0.5 rounded-full',
              dailyOvertime >= 0
                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
            )}>
              {dailyOvertime >= 0 ? '+' : ''}{formatMinutes(dailyOvertime)}
            </span>
          </div>
        </div>
        <div className="w-full h-3 rounded-full bg-secondary overflow-hidden">
          <div
            className={cn(
              'h-full rounded-full transition-all duration-500',
              progressPercent >= 90 ? 'bg-emerald-500' : progressPercent >= 60 ? 'bg-amber-500' : 'bg-red-500',
            )}
            style={{ width: `${progressPercent}%` }}
          />
        </div>
        {totalMinutes > targetMinutes && (
          <p className="text-xs text-emerald-600 dark:text-emerald-400 mt-1">
            +{formatMinutes(totalMinutes - targetMinutes)} Ueberstunden
          </p>
        )}
        {totalMinutes < targetMinutes && totalMinutes > 0 && (
          <p className="text-xs text-muted-foreground mt-1">
            Noch {formatMinutes(targetMinutes - totalMinutes)} bis zum Soll
          </p>
        )}
      </div>

      {/* Manual Entry Form Dialog */}
      <ManualEntryForm open={showManualForm} onOpenChange={setShowManualForm} />
    </div>
  )
}
