import { useState, useRef, useEffect, useMemo } from 'react'
import {
  Play, Pause, Square, Timer, ArrowRight, Coffee, ArrowRightLeft,
} from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '@/lib/cn'
import {
  useTimeTrackingStore, MOCK_PROJECTS, MOCK_TASKS,
  getOvertimeSaldo,
} from '@/stores/timetracking'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, isToday } from '@/modules/profil/tabs/zeiterfassung/time-utils'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function TimeTrackerWidget() {
  const [isOpen, setIsOpen] = useState(false)
  const [selectedCategory, setSelectedCategory] = useState('')
  const [selectedProject, setSelectedProject] = useState('')
  const [selectedTask, setSelectedTask] = useState('')
  const [description, setDescription] = useState('')
  const [showTaskPicker, setShowTaskPicker] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  // Store
  const isClockedIn = useTimeTrackingStore((s) => s.isClockedIn)
  const clockedInAt = useTimeTrackingStore((s) => s.clockedInAt)
  const isOnBreak = useTimeTrackingStore((s) => s.isOnBreak)
  const breakStartedAt = useTimeTrackingStore((s) => s.breakStartedAt)
  const totalBreakMs = useTimeTrackingStore((s) => s.totalBreakMs)
  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const categories = useTimeTrackingStore((s) => s.categories)
  const entries = useTimeTrackingStore((s) => s.entries)
  const targets = useTimeTrackingStore((s) => s.targets)
  const templates = useTimeTrackingStore((s) => s.templates)

  const clockIn = useTimeTrackingStore((s) => s.clockIn)
  const clockOut = useTimeTrackingStore((s) => s.clockOut)
  const startBreak = useTimeTrackingStore((s) => s.startBreak)
  const endBreak = useTimeTrackingStore((s) => s.endBreak)
  const startTimer = useTimeTrackingStore((s) => s.startTimer)
  const pauseTimer = useTimeTrackingStore((s) => s.pauseTimer)
  const resumeTimer = useTimeTrackingStore((s) => s.resumeTimer)
  const stopTimer = useTimeTrackingStore((s) => s.stopTimer)
  const switchTask = useTimeTrackingStore((s) => s.switchTask)

  // Task timer (current task elapsed)
  const taskElapsed = useTimerTick()

  // Total work time ticker
  const [workElapsed, setWorkElapsed] = useState(0)
   
  useEffect(() => {
    if (!isClockedIn || !clockedInAt) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- timer tick update
      setWorkElapsed(0)
      return
    }
    const tick = () => {
      const currentBreak = isOnBreak && breakStartedAt ? Date.now() - breakStartedAt : 0
      setWorkElapsed(Math.max(0, Date.now() - clockedInAt - totalBreakMs - currentBreak))
    }
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [isClockedIn, clockedInAt, isOnBreak, breakStartedAt, totalBreakMs])

  // Break timer
  const [breakElapsed, setBreakElapsed] = useState(0)
   
  useEffect(() => {
    if (!isOnBreak || !breakStartedAt) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- timer tick update
      setBreakElapsed(0)
      return
    }
    const tick = () => setBreakElapsed(Date.now() - breakStartedAt)
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [isOnBreak, breakStartedAt])

  // Filtered tasks based on project
  const availableTasks = useMemo(
    () => selectedProject ? MOCK_TASKS.filter((t) => t.projectId === selectedProject) : [],
    [selectedProject],
  )

  // Overtime saldo
  const overtimeSaldo = useMemo(() => getOvertimeSaldo(entries, targets), [entries, targets])

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
        setShowTaskPicker(false)
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isOpen])

  const handleClockIn = () => {
    clockIn()
  }

  const handleClockOut = () => {
    clockOut()
    setIsOpen(false)
    setShowTaskPicker(false)
  }

  const handleStartTask = () => {
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
    setShowTaskPicker(false)
  }

  const handleSwitchTask = () => {
    const catId = selectedCategory || categories[0]?.id
    if (!catId) return
    switchTask(
      catId,
      description,
      selectedProject || null,
      selectedTask || null,
    )
    setDescription('')
    setSelectedProject('')
    setSelectedTask('')
    setShowTaskPicker(false)
  }

  const handleQuickStart = (categoryId: string, desc: string) => {
    if (activeTimer.status !== 'idle') {
      switchTask(categoryId, desc)
    } else {
      startTimer(categoryId, desc)
    }
  }

  const handleGoToProfile = () => {
    navigate('/profil')
    setIsOpen(false)
  }

  // ─── Render ──────────────────────────────────────────────────

  return (
    <div className="relative" ref={dropdownRef} data-tour="time-tracker">
      {/* ── Trigger Button ── */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={cn(
          'flex items-center gap-2 px-2.5 py-1.5 rounded-lg transition-colors text-sm',
          isClockedIn
            ? isOnBreak
              ? 'bg-amber-500/10 hover:bg-amber-500/15'
              : 'bg-success/10 hover:bg-success/15'
            : 'hover:bg-accent',
        )}
        title={isClockedIn ? (isOnBreak ? 'Pause' : 'Eingestempelt') : 'Einstempeln'}
      >
        {!isClockedIn ? (
          <>
            <Timer className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground text-xs hidden sm:inline">Einstempeln</span>
          </>
        ) : isOnBreak ? (
          <>
            <Coffee className="h-4 w-4 text-amber-500" />
            <span className="text-xs font-medium text-amber-600 dark:text-amber-400">Pause</span>
          </>
        ) : (
          <>
            <span className="relative flex h-2.5 w-2.5">
              <span className={cn(
                'absolute inline-flex h-full w-full rounded-full opacity-75',
                activeTimer.status === 'running' ? 'animate-ping bg-emerald-400' : 'bg-emerald-400',
              )} />
              <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-500" />
            </span>
            <span className="font-mono text-sm font-semibold text-primary tabular-nums">
              {activeTimer.status !== 'idle'
                ? formatElapsed(taskElapsed)
                : formatElapsed(workElapsed)
              }
            </span>
            {activeCategory && (
              <>
                <span className="text-muted-foreground text-xs">·</span>
                <span className="text-xs text-foreground/80 max-w-[80px] truncate">
                  {activeCategory.name}
                </span>
              </>
            )}
          </>
        )}
      </button>

      {/* ── Dropdown ── */}
      {isOpen && (
        <div className="absolute right-0 top-full mt-2 w-80 rounded-xl border border-border bg-card shadow-xl z-50 overflow-hidden">
          {/* ── Not Clocked In ── */}
          {!isClockedIn ? (
            <div className="p-4 space-y-3">
              <div className="text-center space-y-1">
                <Timer className="h-6 w-6 text-muted-foreground mx-auto" />
                <p className="text-sm font-medium text-foreground">Arbeitstag starten</p>
                <p className="text-xs text-muted-foreground">Stempel dich ein um die Zeiterfassung zu beginnen</p>
              </div>
              {overtimeSaldo !== 0 && (
                <div className="text-center">
                  <span className={cn(
                    'text-xs font-medium tabular-nums',
                    overtimeSaldo >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400',
                  )}>
                    Saldo: {overtimeSaldo >= 0 ? '+' : ''}{formatMinutes(overtimeSaldo)}
                  </span>
                </div>
              )}
              <button
                onClick={handleClockIn}
                className="w-full flex items-center justify-center gap-2 rounded-lg bg-primary px-3 py-2.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Play className="h-4 w-4" />
                Einstempeln
              </button>
              <button
                onClick={handleGoToProfile}
                className="w-full flex items-center justify-center gap-1.5 text-xs text-primary hover:text-primary/80 transition-colors"
              >
                Zur Zeiterfassung
                <ArrowRight className="h-3 w-3" />
              </button>
            </div>
          ) : isOnBreak ? (
            <div className="p-4 space-y-3">
              <div className="text-center space-y-1">
                <Coffee className="h-6 w-6 text-amber-500 mx-auto" />
                <p className="text-sm font-medium text-foreground">Pause</p>
                <span className="text-2xl font-mono font-bold text-amber-500 tabular-nums">
                  {formatElapsed(breakElapsed)}
                </span>
              </div>
              <div className="text-center text-xs text-muted-foreground">
                Arbeitszeit: {formatElapsed(workElapsed)}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => endBreak()}
                  className="flex-1 flex items-center justify-center gap-2 rounded-lg bg-primary px-3 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                >
                  <Play className="h-3.5 w-3.5" />
                  Pause beenden
                </button>
                <button
                  onClick={handleClockOut}
                  className="flex items-center justify-center gap-2 rounded-lg bg-destructive px-3 py-2 text-xs font-medium text-destructive-foreground hover:bg-destructive/90 transition-colors"
                >
                  <Square className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          ) : (
            <div className="max-h-[480px] overflow-y-auto">
              {/* ── Work Time Header ── */}
              <div className="px-4 py-3 border-b border-border">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-muted-foreground">Arbeitszeit</span>
                  <span className="font-mono text-sm font-semibold text-foreground tabular-nums">
                    {formatElapsed(workElapsed)}
                  </span>
                </div>
                <div className="w-full h-1.5 rounded-full bg-secondary overflow-hidden">
                  <div
                    className={cn(
                      'h-full rounded-full transition-all',
                      progressPercent >= 90 ? 'bg-emerald-500' : progressPercent >= 60 ? 'bg-amber-500' : 'bg-primary',
                    )}
                    style={{ width: `${progressPercent}%` }}
                  />
                </div>
                <div className="flex items-center justify-between mt-1">
                  <span className="text-[10px] text-muted-foreground tabular-nums">
                    {formatMinutes(todayMinutes)} / {formatMinutes(targetMinutes)}
                  </span>
                  <span className={cn(
                    'text-[10px] font-medium tabular-nums',
                    overtimeSaldo >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400',
                  )}>
                    Saldo: {overtimeSaldo >= 0 ? '+' : ''}{formatMinutes(overtimeSaldo)}
                  </span>
                </div>
              </div>

              {/* ── Active Task ── */}
              {activeTimer.status !== 'idle' && (
                <div className="px-4 py-3 border-b border-border">
                  <div className="flex items-center gap-2 mb-1.5">
                    {activeCategory && (
                      <span className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: activeCategory.color }} />
                    )}
                    <span className="text-sm font-medium text-foreground truncate">
                      {activeCategory?.name ?? 'Aufgabe'}
                    </span>
                    <div className="flex-1" />
                    <span className={cn(
                      'font-mono text-sm font-bold tabular-nums',
                      activeTimer.status === 'running' ? 'text-primary' : 'text-amber-500',
                    )}>
                      {formatElapsed(taskElapsed)}
                    </span>
                  </div>
                  {activeTimer.description && (
                    <p className="text-xs text-muted-foreground mb-2 truncate">{activeTimer.description}</p>
                  )}
                  <div className="flex items-center gap-1.5">
                    {activeTimer.status === 'running' ? (
                      <button
                        onClick={pauseTimer}
                        className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1 text-xs text-foreground hover:bg-secondary transition-colors"
                      >
                        <Pause className="h-3 w-3" />
                        Pause
                      </button>
                    ) : (
                      <button
                        onClick={resumeTimer}
                        className="flex items-center gap-1.5 rounded-md bg-primary px-2.5 py-1 text-xs text-primary-foreground hover:bg-primary/90 transition-colors"
                      >
                        <Play className="h-3 w-3" />
                        Weiter
                      </button>
                    )}
                    <button
                      onClick={() => { stopTimer(); setShowTaskPicker(true) }}
                      className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1 text-xs text-foreground hover:bg-secondary transition-colors"
                    >
                      <ArrowRightLeft className="h-3 w-3" />
                      Wechseln
                    </button>
                    <button
                      onClick={stopTimer}
                      className="flex items-center gap-1.5 rounded-md bg-destructive/10 px-2.5 py-1 text-xs text-destructive hover:bg-destructive/20 transition-colors"
                    >
                      <Square className="h-3 w-3" />
                      Stop
                    </button>
                  </div>
                </div>
              )}

              {/* ── Task Picker ── */}
              {(activeTimer.status === 'idle' || showTaskPicker) && (
                <div className="px-4 py-3 border-b border-border space-y-2">
                  <span className="text-xs font-medium text-muted-foreground">
                    {activeTimer.status !== 'idle' ? 'Aufgabe wechseln' : 'Aufgabe starten'}
                  </span>

                  {/* Category chips */}
                  <div className="flex flex-wrap gap-1">
                    {categories.filter(c => c.id !== 'cat-break').map((cat) => (
                      <button
                        key={cat.id}
                        onClick={() => setSelectedCategory(cat.id)}
                        className={cn(
                          'flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] transition-colors border',
                          selectedCategory === cat.id
                            ? 'border-primary/50 bg-primary/10 text-foreground'
                            : 'border-border bg-secondary/50 text-muted-foreground hover:bg-accent',
                        )}
                      >
                        <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: cat.color }} />
                        {cat.name}
                      </button>
                    ))}
                  </div>

                  {/* Project selector */}
                  <select
                    value={selectedProject}
                    onChange={(e) => { setSelectedProject(e.target.value); setSelectedTask('') }}
                    className="h-7 w-full rounded-md border border-border bg-transparent px-2 text-xs outline-none focus:border-primary"
                  >
                    <option value="">Projekt (optional)...</option>
                    {MOCK_PROJECTS.map((p) => (
                      <option key={p.id} value={p.id}>{p.key} — {p.name}</option>
                    ))}
                  </select>

                  {availableTasks.length > 0 && (
                    <select
                      value={selectedTask}
                      onChange={(e) => setSelectedTask(e.target.value)}
                      className="h-7 w-full rounded-md border border-border bg-transparent px-2 text-xs outline-none focus:border-primary"
                    >
                      <option value="">Aufgabe...</option>
                      {availableTasks.map((t) => (
                        <option key={t.id} value={t.id}>{t.key} — {t.title}</option>
                      ))}
                    </select>
                  )}

                  {/* Description */}
                  <input
                    type="text"
                    placeholder="Beschreibung..."
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        if (activeTimer.status !== 'idle') { handleSwitchTask() } else { handleStartTask() }
                      }
                    }}
                    className="h-7 w-full rounded-md border border-border bg-transparent px-2 text-xs outline-none placeholder:text-muted-foreground focus:border-primary"
                  />

                  {/* Start/Switch button */}
                  <button
                    onClick={activeTimer.status !== 'idle' ? handleSwitchTask : handleStartTask}
                    className="w-full flex items-center justify-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                  >
                    {activeTimer.status !== 'idle' ? (
                      <><ArrowRightLeft className="h-3 w-3" />Wechseln</>
                    ) : (
                      <><Play className="h-3 w-3" />Start</>
                    )}
                  </button>

                  {/* Quick templates */}
                  {templates.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {templates.slice(0, 4).map((tpl) => {
                        const cat = categories.find((c) => c.id === tpl.categoryId)
                        return (
                          <button
                            key={tpl.id}
                            onClick={() => handleQuickStart(tpl.categoryId, tpl.name)}
                            className="flex items-center gap-1 px-2 py-0.5 rounded-full border border-border bg-secondary/50 text-[10px] text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                          >
                            <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: cat?.color }} />
                            {tpl.name}
                          </button>
                        )
                      })}
                    </div>
                  )}
                </div>
              )}

              {/* ── Today's Entries ── */}
              {todayEntries.length > 0 && (
                <div className="px-4 py-2 border-b border-border">
                  <span className="text-[10px] text-muted-foreground uppercase tracking-wider">Heute</span>
                  {todayEntries.slice(-4).map((entry) => {
                    const cat = categories.find((c) => c.id === entry.categoryId)
                    return (
                      <div key={entry.id} className="flex items-center gap-2 py-1 text-xs">
                        <span className="text-muted-foreground font-mono tabular-nums w-[72px] shrink-0">
                          {entry.startTime}–{entry.endTime || '...'}
                        </span>
                        <span className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: cat?.color }} />
                        <span className="text-foreground truncate flex-1">
                          {entry.description || cat?.name}
                        </span>
                        <span className="text-muted-foreground tabular-nums shrink-0">
                          {formatMinutes(entry.durationMinutes)}
                        </span>
                      </div>
                    )
                  })}
                </div>
              )}

              {/* ── Footer: Break + Clock Out ── */}
              <div className="px-4 py-3 space-y-2">
                <div className="flex gap-2">
                  <button
                    onClick={() => { startBreak() }}
                    className="flex-1 flex items-center justify-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary transition-colors"
                  >
                    <Coffee className="h-3.5 w-3.5" />
                    Pause
                  </button>
                  <button
                    onClick={handleClockOut}
                    className="flex-1 flex items-center justify-center gap-1.5 rounded-md bg-destructive px-3 py-1.5 text-xs font-medium text-destructive-foreground hover:bg-destructive/90 transition-colors"
                  >
                    <Square className="h-3.5 w-3.5" />
                    Ausstempeln
                  </button>
                </div>
                <button
                  onClick={handleGoToProfile}
                  className="w-full flex items-center justify-center gap-1.5 text-xs text-primary hover:text-primary/80 transition-colors"
                >
                  Zur Zeiterfassung
                  <ArrowRight className="h-3 w-3" />
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
