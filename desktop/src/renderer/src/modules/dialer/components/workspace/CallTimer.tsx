import { useState, useEffect } from 'react'
import { cn } from '@/lib/cn'

interface CallTimerProps {
  startedAt: number | null
  className?: string
  warnAfterSeconds?: number
  static?: boolean
  staticSeconds?: number
}

export default function CallTimer({
  startedAt,
  className,
  warnAfterSeconds = 180,
  static: isStatic,
  staticSeconds,
}: CallTimerProps) {
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    if (isStatic || !startedAt) return

    const tick = () => setElapsed(Math.floor((Date.now() - startedAt) / 1000))
    tick()
    const interval = setInterval(tick, 1000)
    return () => clearInterval(interval)
  }, [startedAt, isStatic])

  const seconds = isStatic ? (staticSeconds ?? 0) : elapsed
  const minutes = Math.floor(seconds / 60)
  const secs = seconds % 60
  const isWarning = seconds >= warnAfterSeconds

  return (
    <span
      className={cn(
        'font-mono text-3xl font-bold tabular-nums transition-colors duration-300',
        isWarning ? 'text-warning' : 'text-foreground',
        className,
      )}
    >
      {String(minutes).padStart(2, '0')}:{String(secs).padStart(2, '0')}
    </span>
  )
}
