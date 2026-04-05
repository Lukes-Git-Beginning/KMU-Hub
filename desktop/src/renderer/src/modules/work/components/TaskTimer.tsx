/**
 * Task timer component with start/stop controls and elapsed time display.
 *
 * Features:
 * - Real-time HH:MM:SS elapsed counter using requestAnimationFrame
 * - Start/stop toggle button
 * - Auto-stop previous timer when starting on different task
 * - Total time summary display
 * - Timer state persists in Zustand store across page navigation
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Play, Square, Clock, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  useActiveTimer,
  useStartTimer,
  useStopTimer,
  useTaskTimeSummary,
} from '@/api/hooks/useTimeEntries'
import { useWorkStore } from '@/stores/work'

interface TaskTimerProps {
  taskId: string
}

/** Format seconds into HH:MM:SS */
function formatElapsed(totalSeconds: number): string {
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = Math.floor(totalSeconds % 60)
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

/** Format seconds into human-readable duration like "2h 15min" */
function formatDuration(totalSeconds: number): string {
  if (totalSeconds === 0) return '0min'
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  if (hours > 0 && minutes > 0) return `${hours}h ${minutes}min`
  if (hours > 0) return `${hours}h`
  return `${minutes}min`
}

export default function TaskTimer({ taskId }: TaskTimerProps) {
  const { t } = useTranslation()
  const { data: activeTimerData } = useActiveTimer()
  const { data: summaryData } = useTaskTimeSummary(taskId)
  const startTimer = useStartTimer()
  const stopTimer = useStopTimer()

  const { setActiveTimer, clearActiveTimer } = useWorkStore()

  // The active timer for this specific task
  const activeTimer = activeTimerData?.timer
  const isTimerRunningOnThisTask = activeTimer?.task_id === taskId
  const isTimerRunningElsewhere =
    !!activeTimer?.task_id && activeTimer.task_id !== taskId

  // Elapsed time counter
  const [elapsed, setElapsed] = useState(0)
  const rafRef = useRef<number | null>(null)
  const startTimeRef = useRef<number | null>(null)

  // Sync timer state from API response to Zustand store
   
  useEffect(() => {
    if (activeTimer) {
      setActiveTimer(
        activeTimer.task_id ?? '',
        activeTimer.started_at ?? '',
        activeTimer.id ?? ''
      )
    }
  }, [activeTimer, setActiveTimer])

  // Run the elapsed counter
   
  useEffect(() => {
    if (!isTimerRunningOnThisTask || !activeTimer?.started_at) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- timer tick update
      setElapsed(0)
      startTimeRef.current = null
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current)
        rafRef.current = null
      }
      return
    }

    startTimeRef.current = new Date(activeTimer.started_at).getTime()

    function tick() {
      if (startTimeRef.current) {
        const now = Date.now()
        setElapsed(Math.floor((now - startTimeRef.current) / 1000))
      }
      rafRef.current = requestAnimationFrame(tick)
    }

    tick()

    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current)
        rafRef.current = null
      }
    }
  }, [isTimerRunningOnThisTask, activeTimer?.started_at])

  const handleStartStop = useCallback(async () => {
    if (isTimerRunningOnThisTask) {
      try {
        await stopTimer.mutateAsync()
        clearActiveTimer()
      } catch {
        // Error handled by TanStack Query
      }
    } else {
      try {
        const result = await startTimer.mutateAsync(taskId)
        if (result?.entry) {
          setActiveTimer(
            taskId,
            result.entry.started_at ?? new Date().toISOString(),
            result.entry.id ?? ''
          )
        }
      } catch {
        // Error handled by TanStack Query
      }
    }
  }, [
    isTimerRunningOnThisTask,
    taskId,
    startTimer,
    stopTimer,
    setActiveTimer,
    clearActiveTimer,
  ])

  const summary = summaryData?.summary
  const totalSeconds = summary?.total_duration_seconds ?? 0

  return (
    <div className="space-y-3">
      {/* Timer controls */}
      <div className="flex items-center gap-3">
        <Button
          variant={isTimerRunningOnThisTask ? 'destructive' : 'default'}
          size="sm"
          className="gap-1.5"
          onClick={handleStartStop}
          disabled={startTimer.isPending || stopTimer.isPending}
        >
          {isTimerRunningOnThisTask ? (
            <>
              <Square className="h-3.5 w-3.5" />
              {t('work.timer.stop')}
            </>
          ) : (
            <>
              <Play className="h-3.5 w-3.5" />
              {t('work.timer.start')}
            </>
          )}
        </Button>

        {isTimerRunningOnThisTask && (
          <span className="font-mono text-lg font-semibold text-green-600 dark:text-green-400 tabular-nums">
            {formatElapsed(elapsed)}
          </span>
        )}
      </div>

      {/* Switch timer hint */}
      {isTimerRunningElsewhere && (
        <div className="flex items-start gap-2 rounded-md border border-warning/30 bg-warning-light px-3 py-2 text-xs text-warning-foreground">
          <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <div>
            <span className="font-medium">{t('work.timer.runningElsewhere')}</span>{' '}
            {t('work.timer.switchHint')}
          </div>
        </div>
      )}

      {/* Total time summary */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Clock className="h-3 w-3" />
        <span>{t('work.timer.total')}: {formatDuration(totalSeconds)}</span>
        {summary && summary.entry_count > 0 && (
          <span>({summary.entry_count} Einträge)</span>
        )}
      </div>
    </div>
  )
}
