import { useState, useRef, useEffect, useMemo } from 'react'
import { Play, Pause, Square, Timer, X, ArrowRight, MapPin, FolderKanban, Download } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@/components/ui/tooltip'
import { cn } from '@/lib/cn'
import {
  useTimeTrackingStore, MOCK_PROJECTS, MOCK_TASKS,
  getOvertimeSaldo,
} from '@/stores/timetracking'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, isToday, todayStr } from '@/modules/profil/tabs/zeiterfassung/time-utils'
import ExportDialog from '@/modules/profil/tabs/zeiterfassung/ExportDialog'

export function TimeTrackerWidget() {
  const [isOpen, setIsOpen] = useState(false)
  const [selectedCategory, setSelectedCategory] = useState('')
  const [selectedProject, setSelectedProject] = useState('')
  const [selectedTask, setSelectedTask] = useState('')
  const [description, setDescription] = useState('')
  const [showExportDialog, setShowExportDialog] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const categories = useTimeTrackingStore((s) => s.categories)
  const entries = useTimeTrackingStore((s) => s.entries)
  const targets = useTimeTrackingStore((s) => s.targets)
  const templates = useTimeTrackingStore((s) => s.templates)
  const absences = useTimeTrackingStore((s) => s.absences)
  const gpsEnabled = useTimeTrackingStore((s) => s.gpsEnabled)
  const startTimer = useTimeTrackingStore((s) => s.startTimer)
  const pauseTimer = useTimeTrackingStore((s) => s.pauseTimer)
  const resumeTimer = useTimeTrackingStore((s) => s.resumeTimer)
  const stopTimer = useTimeTrackingStore((s) => s.stopTimer)
  const setGpsEnabled = useTimeTrackingStore((s) => s.setGpsEnabled)

  // Filtered tasks based on selected project (6.6)
  const availableTasks = useMemo(
    () => selectedProject ? MOCK_TASKS.filter((t) => t.projectId === selectedProject) : [],
    [selectedProject],
  )

  // Overtime saldo (6.8)
  const overtimeSaldo = useMemo(() => getOvertimeSaldo(entries, targets), [entries, targets])

  // Absence lock (6.10)
  const todayAbsence = useMemo(() => {
    const today = todayStr()
    return absences.find((a) => {
      if (a.status !== 'approved') return false
      return today >= a.startDate && today <= a.endDate
    })
  }, [absences])
  const isTimerLocked = todayAbsence?.type === 'vacation' || todayAbsence?.type === 'sick'

  const elapsed = useTimerTick()

  // Today's entries and progress
  const todayEntries = entries
    .filter((e) => isToday(e.date))
    .sort((a, b) => a.startTime.localeCompare(b.startTime))
  const todayMinutes = todayEntries.reduce((sum, e) => sum + e.durationMinutes, 0)
  const targetMinutes = targets.dailyHours * 60
  const progressPercent = Math.min(100, Math.round((todayMinutes / targetMinutes) * 100))

  const activeCategory = categories.find((c) => c.id === activeTimer.categoryId)

  // Close on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const handleStart = () => {
    if (isTimerLocked) return
    const catId = selectedCategory || categories[0]?.id
    if (!catId) return
    startTimer(
      catId,
      description,
      selectedProject || null,
      selectedTask || null,
    )
    setDescription('')
    setSelectedProject('')
    setSelectedTask('')
  }

  const handleQuickStart = (categoryId: string, desc: string) => {
    if (isTimerLocked) return
    startTimer(categoryId, desc)
  }

  const handleGoToProfile = () => {
    navigate('/profil')
    setIsOpen(false)
  }

  return (
    <div className="relative" ref={dropdownRef}>
      {/* Trigger Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-2 px-2.5 py-1.5 rounded-lg transition-colors text-sm',
          activeTimer.status !== 'idle'
            ? 'bg-primary/10 hover:bg-primary/15'
            : 'hover:bg-accent',
        )}
      >
        {activeTimer.status === 'idle' ? (
          <>
            <Timer className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground font-mono text-xs tabular-nums">00:00:00</span>
          </>
        ) : (
          <>
            <span className="relative flex h-2.5 w-2.5">
              <span className={cn(
                'absolute inline-flex h-full w-full rounded-full opacity-75',
                activeTimer.status === 'running' ? 'animate-ping bg-emerald-400' : 'animate-pulse bg-amber-400',
              )} />
              <span className={cn(
                'relative inline-flex h-2.5 w-2.5 rounded-full',
                activeTimer.status === 'running' ? 'bg-emerald-500' : 'bg-amber-500',
              )} />
            </span>
            <span className="font-mono text-sm font-semibold text-primary tabular-nums">
              {formatElapsed(elapsed)}
            </span>
            {activeCategory && (
              <span
                className="h-2 w-2 rounded-full"
                style={{ backgroundColor: activeCategory.color }}
              />
            )}
          </>
        )}
      </button>

      {/* Dropdown Panel */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-96 rounded-xl border border-border bg-card shadow-xl z-50 overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <div className="flex items-center gap-2">
              <Timer className="h-4 w-4 text-primary" />
              <h3 className="font-semibold text-sm text-foreground">Zeiterfassung</h3>
            </div>
            <button onClick={() => setIsOpen(false)} className="p-1 rounded hover:bg-accent transition-colors">
              <X className="h-4 w-4 text-muted-foreground" />
            </button>
          </div>

          <div className="max-h-[500px] overflow-y-auto">
            {/* Timer Controls */}
            <div className="px-4 py-3 space-y-3 border-b border-border">
              {activeTimer.status === 'idle' ? (
                <>
                  {/* Absence Lock Banner (6.10) */}
                  {isTimerLocked && (
                    <div className="rounded-md bg-amber-50 dark:bg-amber-950/30 border border-amber-300 dark:border-amber-700 px-3 py-2 text-xs text-amber-800 dark:text-amber-300 flex items-center gap-2">
                      <span>Timer gesperrt — Abwesenheit</span>
                    </div>
                  )}

                  <div className="flex items-center gap-2">
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span>
                            <Button size="sm" onClick={handleStart} disabled={isTimerLocked} className="gap-1.5 shrink-0">
                              <Play className="h-3.5 w-3.5" />
                              Start
                            </Button>
                          </span>
                        </TooltipTrigger>
                        {isTimerLocked && (
                          <TooltipContent>
                            <p className="text-xs">Timer gesperrt — Abwesenheit</p>
                          </TooltipContent>
                        )}
                      </Tooltip>
                    </TooltipProvider>
                    <Select value={selectedCategory} onValueChange={setSelectedCategory}>
                      <SelectTrigger className="h-8 text-xs flex-1">
                        <SelectValue placeholder="Kategorie..." />
                      </SelectTrigger>
                      <SelectContent>
                        {categories.map((cat) => (
                          <SelectItem key={cat.id} value={cat.id}>
                            <span className="flex items-center gap-2">
                              <span className="h-2 w-2 rounded-full" style={{ backgroundColor: cat.color }} />
                              {cat.name}
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  {/* Project / Task Dropdowns (6.6) */}
                  <div className="flex items-center gap-2">
                    <Select value={selectedProject} onValueChange={(val) => { setSelectedProject(val); setSelectedTask('') }}>
                      <SelectTrigger className="h-8 text-xs flex-1">
                        <SelectValue placeholder="Projekt..." />
                      </SelectTrigger>
                      <SelectContent>
                        {MOCK_PROJECTS.map((proj) => (
                          <SelectItem key={proj.id} value={proj.id}>
                            <span className="flex items-center gap-1.5">
                              <FolderKanban className="h-3 w-3 text-muted-foreground" />
                              <span className="font-mono text-[10px] text-muted-foreground">{proj.key}</span>
                              {proj.name}
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {availableTasks.length > 0 && (
                      <Select value={selectedTask} onValueChange={setSelectedTask}>
                        <SelectTrigger className="h-8 text-xs flex-1">
                          <SelectValue placeholder="Aufgabe..." />
                        </SelectTrigger>
                        <SelectContent>
                          {availableTasks.map((task) => (
                            <SelectItem key={task.id} value={task.id}>
                              <span className="flex items-center gap-1.5">
                                <span className="font-mono text-[10px] text-muted-foreground">{task.key}</span>
                                {task.title}
                              </span>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    )}
                  </div>

                  <Input
                    placeholder="Beschreibung..."
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleStart()}
                    className="h-8 text-xs"
                  />

                  {/* Quick Start Templates */}
                  <div className="flex flex-wrap gap-1.5">
                    {templates.slice(0, 4).map((tpl) => {
                      const cat = categories.find((c) => c.id === tpl.categoryId)
                      return (
                        <button
                          key={tpl.id}
                          onClick={() => handleQuickStart(tpl.categoryId, tpl.name)}
                          disabled={isTimerLocked}
                          className={cn(
                            'flex items-center gap-1.5 px-2.5 py-1 rounded-full border border-border bg-secondary/50 text-xs text-muted-foreground transition-colors',
                            isTimerLocked ? 'opacity-50 cursor-not-allowed' : 'hover:bg-accent hover:text-foreground',
                          )}
                        >
                          <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: cat?.color }} />
                          {tpl.name}
                        </button>
                      )
                    })}
                  </div>
                </>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-center gap-3">
                    {activeTimer.status === 'running' ? (
                      <Button size="sm" variant="outline" onClick={pauseTimer} className="gap-1.5">
                        <Pause className="h-3.5 w-3.5" />
                        Pause
                      </Button>
                    ) : (
                      <Button size="sm" onClick={resumeTimer} className="gap-1.5">
                        <Play className="h-3.5 w-3.5" />
                        Weiter
                      </Button>
                    )}
                    <Button size="sm" variant="destructive" onClick={stopTimer} className="gap-1.5">
                      <Square className="h-3.5 w-3.5" />
                      Stop
                    </Button>
                    <div className="flex-1 text-right">
                      <span className="text-lg font-mono font-bold text-primary tabular-nums">
                        {formatElapsed(elapsed)}
                      </span>
                    </div>
                  </div>
                  {/* GPS active indicator (6.11) */}
                  {gpsEnabled && activeTimer.status === 'running' && (
                    <div className="flex items-center gap-1.5 text-[10px] text-emerald-600 dark:text-emerald-400">
                      <MapPin className="h-3 w-3" />
                      <span>GPS aktiv</span>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Today Progress + Overtime Saldo (6.8) + GPS Toggle (6.11) */}
            <div className="px-4 py-3 border-b border-border space-y-2">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-xs text-muted-foreground">Heute</span>
                <span className="text-xs font-medium text-foreground tabular-nums">
                  {formatMinutes(todayMinutes)} / {formatMinutes(targetMinutes)}
                </span>
              </div>
              <div className="w-full h-2 rounded-full bg-secondary overflow-hidden">
                <div
                  className={cn(
                    'h-full rounded-full transition-all',
                    progressPercent >= 90 ? 'bg-emerald-500' : progressPercent >= 60 ? 'bg-amber-500' : 'bg-primary',
                  )}
                  style={{ width: `${progressPercent}%` }}
                />
              </div>

              {/* Overtime Saldo (6.8) + GPS Toggle (6.11) + Export (6.7) */}
              <div className="flex items-center justify-between pt-1">
                <div className="flex items-center gap-2">
                  <span className="text-[10px] text-muted-foreground uppercase tracking-wider">Saldo:</span>
                  <span className={cn(
                    'text-xs font-semibold tabular-nums',
                    overtimeSaldo >= 0
                      ? 'text-emerald-600 dark:text-emerald-400'
                      : 'text-red-600 dark:text-red-400',
                  )}>
                    {overtimeSaldo >= 0 ? '+' : ''}{formatMinutes(overtimeSaldo)}
                  </span>
                </div>
                <div className="flex items-center gap-3">
                  {/* GPS Toggle (6.11) */}
                  <label className="flex items-center gap-1.5 cursor-pointer">
                    <MapPin className={cn('h-3 w-3', gpsEnabled ? 'text-primary' : 'text-muted-foreground')} />
                    <span className="text-[10px] text-muted-foreground">GPS</span>
                    <Switch
                      checked={gpsEnabled}
                      onCheckedChange={setGpsEnabled}
                      className="scale-75 origin-right"
                    />
                  </label>
                  {/* Export Button (6.7) */}
                  <button
                    onClick={() => setShowExportDialog(true)}
                    className="p-1 rounded hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                    title="Exportieren"
                  >
                    <Download className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </div>

            {/* Today's Entries */}
            <div className="px-4 py-2">
              {todayEntries.length === 0 && activeTimer.status === 'idle' && (
                <p className="text-xs text-muted-foreground text-center py-4">
                  Noch keine Eintraege heute
                </p>
              )}
              {todayEntries.slice(-5).map((entry) => {
                const cat = categories.find((c) => c.id === entry.categoryId)
                return (
                  <div key={entry.id} className="flex items-center gap-2 py-1.5 text-xs">
                    <span className="text-muted-foreground font-mono tabular-nums w-20 shrink-0">
                      {entry.startTime}-{entry.endTime || '...'}
                    </span>
                    <span className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: cat?.color }} />
                    <span className="text-foreground truncate flex-1">{cat?.name}</span>
                    {/* GPS location icon (6.11) */}
                    {entry.location && (
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="text-muted-foreground shrink-0">
                              <MapPin className="h-3 w-3" />
                            </span>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p className="text-xs">{entry.location.address}</p>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    )}
                    <span className="text-muted-foreground tabular-nums shrink-0">
                      {formatMinutes(entry.durationMinutes)}
                    </span>
                  </div>
                )
              })}
            </div>

            {/* Link to Full Page */}
            <div className="px-4 py-3 border-t border-border">
              <button
                onClick={handleGoToProfile}
                className="w-full flex items-center justify-center gap-2 text-xs font-medium text-primary hover:text-primary/80 transition-colors"
              >
                Zur Zeiterfassung
                <ArrowRight className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Export Dialog (6.7) */}
      <ExportDialog open={showExportDialog} onOpenChange={setShowExportDialog} />
    </div>
  )
}
