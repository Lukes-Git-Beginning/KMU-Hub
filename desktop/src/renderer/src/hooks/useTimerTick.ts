import { useState, useEffect } from 'react'
import { useTimeTrackingStore } from '@/stores/timetracking'

/**
 * Returns the current elapsed milliseconds for the active timer.
 * Re-renders every second when the timer is running.
 */
export function useTimerTick(): number {
  const activeTimer = useTimeTrackingStore((s) => s.activeTimer)
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    if (activeTimer.status === 'idle') {
      setElapsed(0)
      return
    }

    if (activeTimer.status === 'paused') {
      if (activeTimer.pausedAt && activeTimer.startedAt) {
        setElapsed(activeTimer.pausedAt - activeTimer.startedAt)
      }
      return
    }

    // Running — update every second
    const tick = () => {
      if (activeTimer.startedAt) {
        setElapsed(Date.now() - activeTimer.startedAt)
      }
    }
    tick() // immediate update
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [activeTimer.status, activeTimer.startedAt, activeTimer.pausedAt])

  return elapsed
}

/** Format milliseconds as HH:MM:SS */
export function formatElapsed(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}
