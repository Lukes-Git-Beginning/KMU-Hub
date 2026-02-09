import { useState, useMemo } from 'react'
import {
  Play, Pause, Square, Clock, Calendar, BarChart3, Users, Tag, Plus, ChevronLeft, ChevronRight,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, todayStr, isToday } from './zeiterfassung/time-utils'
import TodayView from './zeiterfassung/TodayView'
import WeekView from './zeiterfassung/WeekView'
import MonthView from './zeiterfassung/MonthView'
import ReportsView from './zeiterfassung/ReportsView'
import TeamView from './zeiterfassung/TeamView'
import CategoriesView from './zeiterfassung/CategoriesView'

type ViewKey = 'today' | 'week' | 'month' | 'reports' | 'team' | 'categories'

const VIEWS: { key: ViewKey; label: string; icon: typeof Clock }[] = [
  { key: 'today', label: 'Heute', icon: Clock },
  { key: 'week', label: 'Woche', icon: Calendar },
  { key: 'month', label: 'Monat', icon: BarChart3 },
  { key: 'reports', label: 'Berichte', icon: BarChart3 },
  { key: 'team', label: 'Team', icon: Users },
  { key: 'categories', label: 'Kategorien', icon: Tag },
]

export default function ZeiterfassungTab() {
  const [activeView, setActiveView] = useState<ViewKey>('today')
  const [timerCategory, setTimerCategory] = useState('')
  const [timerDescription, setTimerDescription] = useState('')

  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const categories = useTimeTrackingStore((s) => s.categories)
  const entries = useTimeTrackingStore((s) => s.entries)
  const targets = useTimeTrackingStore((s) => s.targets)
  const startTimer = useTimeTrackingStore((s) => s.startTimer)
  const pauseTimer = useTimeTrackingStore((s) => s.pauseTimer)
  const resumeTimer = useTimeTrackingStore((s) => s.resumeTimer)
  const stopTimer = useTimeTrackingStore((s) => s.stopTimer)

  const elapsed = useTimerTick()

  const todayEntries = useMemo(
    () => entries.filter((e) => isToday(e.date)),
    [entries],
  )
  const todayMinutes = useMemo(
    () => todayEntries.reduce((sum, e) => sum + e.durationMinutes, 0),
    [todayEntries],
  )
  const targetMinutes = targets.dailyHours * 60
  const progressPercent = Math.min(100, Math.round((todayMinutes / targetMinutes) * 100))

  const handleStart = () => {
    const catId = timerCategory || categories[0]?.id
    if (!catId) return
    startTimer(catId, timerDescription)
    setTimerDescription('')
  }

  return (
    <div className="h-full flex flex-col">
      {/* Timer Toolbar */}
      <div className="border-b border-border bg-card/50 px-6 py-4">
        <div className="flex flex-col sm:flex-row items-start sm:items-center gap-4">
          {/* Timer Controls */}
          <div className="flex items-center gap-2">
            {activeTimer.status === 'idle' ? (
              <Button size="sm" onClick={handleStart} className="gap-2">
                <Play className="h-4 w-4" />
                Start
              </Button>
            ) : activeTimer.status === 'running' ? (
              <>
                <Button size="sm" variant="outline" onClick={pauseTimer} className="gap-2">
                  <Pause className="h-4 w-4" />
                  Pause
                </Button>
                <Button size="sm" variant="destructive" onClick={stopTimer} className="gap-2">
                  <Square className="h-4 w-4" />
                  Stop
                </Button>
              </>
            ) : (
              <>
                <Button size="sm" onClick={resumeTimer} className="gap-2">
                  <Play className="h-4 w-4" />
                  Weiter
                </Button>
                <Button size="sm" variant="destructive" onClick={stopTimer} className="gap-2">
                  <Square className="h-4 w-4" />
                  Stop
                </Button>
              </>
            )}

            {activeTimer.status !== 'idle' && (
              <span className="text-lg font-mono font-bold text-primary tabular-nums">
                {formatElapsed(elapsed)}
              </span>
            )}
          </div>

          {/* Category + Description */}
          {activeTimer.status === 'idle' && (
            <div className="flex items-center gap-2 flex-1">
              <Select value={timerCategory} onValueChange={setTimerCategory}>
                <SelectTrigger className="w-48">
                  <SelectValue placeholder="Kategorie..." />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((cat) => (
                    <SelectItem key={cat.id} value={cat.id}>
                      <span className="flex items-center gap-2">
                        <span
                          className="h-2.5 w-2.5 rounded-full shrink-0"
                          style={{ backgroundColor: cat.color }}
                        />
                        {cat.name}
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input
                placeholder="Beschreibung..."
                value={timerDescription}
                onChange={(e) => setTimerDescription(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleStart()}
                className="flex-1 max-w-xs"
              />
            </div>
          )}

          {/* Active timer info */}
          {activeTimer.status !== 'idle' && activeTimer.categoryId && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span
                className="h-2.5 w-2.5 rounded-full"
                style={{ backgroundColor: categories.find((c) => c.id === activeTimer.categoryId)?.color }}
              />
              <span>{categories.find((c) => c.id === activeTimer.categoryId)?.name}</span>
              {activeTimer.description && (
                <>
                  <span className="text-border">|</span>
                  <span>{activeTimer.description}</span>
                </>
              )}
            </div>
          )}

          {/* Daily Summary */}
          <div className="ml-auto flex items-center gap-3 text-sm">
            <span className="text-muted-foreground">Heute:</span>
            <span className="font-semibold text-foreground">
              {formatMinutes(todayMinutes)}
            </span>
            <span className="text-muted-foreground">/</span>
            <span className="text-muted-foreground">
              {formatMinutes(targetMinutes)}
            </span>
            <div className="w-24 h-2 rounded-full bg-secondary overflow-hidden">
              <div
                className={cn(
                  'h-full rounded-full transition-all',
                  progressPercent >= 90 ? 'bg-emerald-500' : progressPercent >= 60 ? 'bg-amber-500' : 'bg-red-500',
                )}
                style={{ width: `${progressPercent}%` }}
              />
            </div>
            <span className="text-xs text-muted-foreground">{progressPercent}%</span>
          </div>
        </div>
      </div>

      {/* View Switcher */}
      <div className="border-b border-border bg-card/30 px-6">
        <nav className="flex gap-1" role="tablist">
          {VIEWS.map((view) => {
            const Icon = view.icon
            return (
              <button
                key={view.key}
                role="tab"
                aria-selected={activeView === view.key}
                onClick={() => setActiveView(view.key)}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-2.5 text-xs font-medium border-b-2 transition-colors',
                  activeView === view.key
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground',
                )}
              >
                <Icon className="h-3.5 w-3.5" />
                {view.label}
              </button>
            )
          })}
        </nav>
      </div>

      {/* Active View */}
      <div className="flex-1 overflow-auto">
        {activeView === 'today' && <TodayView />}
        {activeView === 'week' && <WeekView />}
        {activeView === 'month' && <MonthView />}
        {activeView === 'reports' && <ReportsView />}
        {activeView === 'team' && <TeamView />}
        {activeView === 'categories' && <CategoriesView />}
      </div>
    </div>
  )
}
