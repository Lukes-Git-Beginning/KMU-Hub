/**
 * Header clock-in/out toggle button.
 *
 * Compact widget that displays current work time status and allows
 * quick clock-in/out, break start/end from the header bar.
 *
 * Uses requestAnimationFrame for smooth timer display (same pattern
 * as Phase 6 task timer). ArbZG toast warnings are triggered
 * automatically via mutation onSuccess callbacks in hr-hooks.ts.
 */
import { useState, useEffect, useRef, useCallback } from 'react'
import { Square, Coffee, Timer } from 'lucide-react'
import { cn } from '@/lib/cn'
import {
  useWorkTimeStatus,
  useClockIn,
  useClockOut,
  useStartBreak,
  useEndBreak,
} from '@/api/hooks/hr-hooks'

export function ClockInButton() {
  const { data: status } = useWorkTimeStatus()
  const clockInMutation = useClockIn()
  const clockOutMutation = useClockOut()
  const startBreakMutation = useStartBreak()
  const endBreakMutation = useEndBreak()

  const [elapsedDisplay, setElapsedDisplay] = useState('00:00:00')
  const [isExpanded, setIsExpanded] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const rafRef = useRef<number>(0)

  const isMutating =
    clockInMutation.isPending ||
    clockOutMutation.isPending ||
    startBreakMutation.isPending ||
    endBreakMutation.isPending

  // Live timer using requestAnimationFrame
  const updateTimer = useCallback(() => {
    if (status?.isClockedIn && status.currentShiftStart && !status.isOnBreak) {
      const start = new Date(status.currentShiftStart).getTime()
      const now = Date.now()
      const totalSeconds = Math.floor((now - start) / 1000)
      const hours = Math.floor(totalSeconds / 3600)
      const minutes = Math.floor((totalSeconds % 3600) / 60)
      const seconds = totalSeconds % 60
      setElapsedDisplay(
        `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`,
      )
    }
    // eslint-disable-next-line react-hooks/immutability -- ref assigned in useCallback for rAF loop
    rafRef.current = requestAnimationFrame(updateTimer)
  }, [status?.isClockedIn, status?.currentShiftStart, status?.isOnBreak])


  useEffect(() => {
    if (status?.isClockedIn) {
      rafRef.current = requestAnimationFrame(updateTimer)
    } else {
      setElapsedDisplay('00:00:00')
    }
    return () => {
      if (rafRef.current) cancelAnimationFrame(rafRef.current)
    }
  }, [status?.isClockedIn, updateTimer])

  // Close on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsExpanded(false)
      }
    }
    if (isExpanded) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [isExpanded])

  const handleClockIn = () => {
    clockInMutation.mutate()
    setIsExpanded(false)
  }

  const handleClockOut = () => {
    clockOutMutation.mutate()
    setIsExpanded(false)
  }

  return (
    <div className="relative" ref={containerRef}>
      {/* Trigger button */}
      <button
        onClick={() => {
          if (!status?.isClockedIn) {
            // Quick clock-in on click when idle
            handleClockIn()
          } else {
            setIsExpanded(!isExpanded)
          }
        }}
        disabled={isMutating}
        className={cn(
          'flex items-center gap-2 px-2.5 py-1.5 rounded-lg transition-colors text-sm',
          status?.isClockedIn
            ? 'bg-success/10 hover:bg-success/15'
            : 'hover:bg-accent',
        )}
        title={status?.isClockedIn ? 'Eingestempelt' : 'Einstempeln'}
      >
        {!status?.isClockedIn ? (
          <>
            <Timer className="h-4 w-4 text-muted-foreground" />
            <span className="text-muted-foreground text-xs hidden sm:inline">Einstempeln</span>
          </>
        ) : status.isOnBreak ? (
          <>
            <Coffee className="h-4 w-4 text-amber-500" />
            <span className="text-xs font-medium text-amber-600 dark:text-amber-400">Pause</span>
          </>
        ) : (
          <>
            <span className="relative flex h-2.5 w-2.5">
              <span className="absolute inline-flex h-full w-full rounded-full opacity-75 animate-ping bg-emerald-400" />
              <span className="relative inline-flex h-2.5 w-2.5 rounded-full bg-emerald-500" />
            </span>
            <span className="font-mono text-sm font-semibold text-primary tabular-nums">
              {elapsedDisplay}
            </span>
          </>
        )}
      </button>

      {/* Expanded controls dropdown */}
      {isExpanded && status?.isClockedIn && (
        <div className="absolute right-0 top-full mt-2 w-48 rounded-xl border border-border bg-card shadow-xl z-50 overflow-hidden">
          <div className="p-3 space-y-2">
            <div className="text-center mb-2">
              <span className="text-lg font-mono font-bold text-primary tabular-nums">
                {status.isOnBreak ? 'Pause' : elapsedDisplay}
              </span>
            </div>

            {status.isOnBreak ? (
              <button
                onClick={() => { endBreakMutation.mutate(); setIsExpanded(false) }}
                disabled={isMutating}
                className="w-full flex items-center justify-center gap-2 rounded-lg bg-primary px-3 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <Coffee className="h-3.5 w-3.5" />
                Pause beenden
              </button>
            ) : (
              <>
                <button
                  onClick={() => { startBreakMutation.mutate(); setIsExpanded(false) }}
                  disabled={isMutating}
                  className="w-full flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-xs font-medium text-foreground hover:bg-secondary transition-colors"
                >
                  <Coffee className="h-3.5 w-3.5" />
                  Pause starten
                </button>
                <button
                  onClick={handleClockOut}
                  disabled={isMutating}
                  className="w-full flex items-center justify-center gap-2 rounded-lg bg-destructive px-3 py-2 text-xs font-medium text-destructive-foreground hover:bg-destructive/90 transition-colors"
                >
                  <Square className="h-3.5 w-3.5" />
                  Ausstempeln
                </button>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
