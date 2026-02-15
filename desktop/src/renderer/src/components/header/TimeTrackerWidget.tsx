import { useState, useRef, useEffect } from 'react'
import { Play, Pause, Square, Timer, X, ArrowRight } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, isToday } from '@/modules/profil/tabs/zeiterfassung/time-utils'

export function TimeTrackerWidget() {
  const [isOpen, setIsOpen] = useState(false)
  const [selectedCategory, setSelectedCategory] = useState('')
  const [description, setDescription] = useState('')
  const dropdownRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const categories = useTimeTrackingStore((s) => s.categories)
  const entries = useTimeTrackingStore((s) => s.entries)
  const targets = useTimeTrackingStore((s) => s.targets)
  const templates = useTimeTrackingStore((s) => s.templates)
  const startTimer = useTimeTrackingStore((s) => s.startTimer)
  const pauseTimer = useTimeTrackingStore((s) => s.pauseTimer)
  const resumeTimer = useTimeTrackingStore((s) => s.resumeTimer)
  const stopTimer = useTimeTrackingStore((s) => s.stopTimer)

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
    const catId = selectedCategory || categories[0]?.id
    if (!catId) return
    startTimer(catId, description)
    setDescription('')
  }

  const handleQuickStart = (categoryId: string, desc: string) => {
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
                  <div className="flex items-center gap-2">
                    <Button size="sm" onClick={handleStart} className="gap-1.5 shrink-0">
                      <Play className="h-3.5 w-3.5" />
                      Start
                    </Button>
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
                          className="flex items-center gap-1.5 px-2.5 py-1 rounded-full border border-border bg-secondary/50 text-xs text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                        >
                          <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: cat?.color }} />
                          {tpl.name}
                        </button>
                      )
                    })}
                  </div>
                </>
              ) : (
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
              )}
            </div>

            {/* Today Progress */}
            <div className="px-4 py-3 border-b border-border">
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
            </div>

            {/* Today's Entries */}
            <div className="px-4 py-2">
              {todayEntries.length === 0 && activeTimer.status === 'idle' && (
                <p className="text-xs text-muted-foreground text-center py-4">
                  Noch keine Einträge heute
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
    </div>
  )
}
