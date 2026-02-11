import { useState, useMemo } from 'react'
import { Plus, Trash2, Clock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/cn'
import { useTimeTrackingStore } from '@/stores/timetracking'
import { useTimerTick, formatElapsed } from '@/hooks/useTimerTick'
import { formatMinutes, isToday } from './time-utils'
import ManualEntryForm from './ManualEntryForm'

export default function TodayView() {
  const [showManualForm, setShowManualForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)

  const entries = useTimeTrackingStore((s) => s.entries)
  const categories = useTimeTrackingStore((s) => s.categories)
  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const targets = useTimeTrackingStore((s) => s.targets)
  const deleteEntry = useTimeTrackingStore((s) => s.deleteEntry)
  const elapsed = useTimerTick()

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

  const getCategoryInfo = (catId: string) =>
    categories.find((c) => c.id === catId)

  return (
    <div className="p-6 space-y-4 max-w-3xl mx-auto">
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
              {activeTimer.status === 'running' ? 'Laeuft' : 'Pausiert'}
            </span>
          </div>
        </div>
      )}

      {/* Entry List */}
      <div className="space-y-2">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-foreground">
            Eintraege heute ({todayEntries.length})
          </h3>
          <Button size="sm" variant="outline" onClick={() => setShowManualForm(true)} className="gap-2">
            <Plus className="h-3.5 w-3.5" />
            Manuell eintragen
          </Button>
        </div>

        {todayEntries.length === 0 && activeTimer.status === 'idle' && (
          <div className="text-center py-12 text-muted-foreground">
            <Clock className="h-10 w-10 mx-auto mb-3 opacity-50" />
            <p className="font-medium">Noch keine Eintraege heute</p>
            <p className="text-sm">Starte den Timer oder erstelle einen manuellen Eintrag</p>
          </div>
        )}

        {todayEntries.map((entry) => {
          const cat = getCategoryInfo(entry.categoryId)
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

              {/* Description */}
              <span className="text-sm text-muted-foreground flex-1 truncate">
                {entry.description}
              </span>

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

      {/* Soll/Ist Footer */}
      <div className="rounded-xl border border-border bg-card p-4">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-muted-foreground">Soll / Ist</span>
          <span className="text-sm font-medium text-foreground">
            {formatMinutes(totalMinutes)} / {formatMinutes(targetMinutes)}
          </span>
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
