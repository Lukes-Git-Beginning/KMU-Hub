/**
 * Enhanced SLA indicator badge with color coding and animations.
 *
 * Green (>4h remaining), Yellow (<4h), Red (overdue with pulse).
 */
import { Clock, AlertTriangle } from 'lucide-react'
import { cn } from '@/lib/cn'

interface SLABadgeProps {
  overdue: boolean
  remaining: string
  dueAt?: string
  compact?: boolean
}

export function SLABadge({ overdue, remaining, dueAt, compact }: SLABadgeProps) {
  const isWarning = !overdue && remaining.includes('h') && !remaining.includes('d')
  const hours = parseInt(remaining, 10)
  const isYellow = isWarning && !isNaN(hours) && hours < 4

  let colorClass = 'text-success'
  let bgClass = 'bg-success-light'
  let Icon = Clock

  if (overdue) {
    colorClass = 'text-destructive'
    bgClass = 'bg-error-light'
    Icon = AlertTriangle
  } else if (isYellow) {
    colorClass = 'text-warning-foreground'
    bgClass = 'bg-warning-light'
  }

  if (compact) {
    return (
      <span className={cn('flex items-center gap-1 text-xs font-medium', colorClass)}>
        <Icon className={cn('h-3 w-3', overdue && 'animate-pulse')} />
        {remaining}
      </span>
    )
  }

  return (
    <div
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium transition-all',
        bgClass,
        colorClass,
        overdue && 'animate-pulse'
      )}
      title={dueAt ? `Faellig: ${new Date(dueAt).toLocaleString('de-CH')}` : undefined}
    >
      <Icon className="h-3.5 w-3.5" />
      <span>{remaining}</span>
    </div>
  )
}

export function SLABreachBanner({ remaining }: { remaining: string }) {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-error-light px-3 py-2 text-xs font-medium text-destructive">
      <AlertTriangle className="h-4 w-4 animate-pulse shrink-0" />
      <span>SLA-Verletzung: {remaining} — Bitte umgehend bearbeiten!</span>
    </div>
  )
}
