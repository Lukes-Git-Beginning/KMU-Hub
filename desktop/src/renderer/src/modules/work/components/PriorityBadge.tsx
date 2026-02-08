/**
 * Color-coded priority indicator badge with icon.
 *
 * Supports full mode (icon + text) and compact mode (icon only) for
 * space-constrained contexts like Kanban cards.
 */
import { AlertTriangle, ArrowUp, Minus, ArrowDown } from 'lucide-react'
import { cn } from '@/lib'

type Priority = 'urgent' | 'high' | 'medium' | 'low'

const priorityConfig: Record<
  Priority,
  {
    color: string
    icon: React.ComponentType<{ className?: string }>
    label: string
  }
> = {
  urgent: {
    color: 'text-red-600 bg-red-50 border-red-200',
    icon: AlertTriangle,
    label: 'Dringend',
  },
  high: {
    color: 'text-orange-600 bg-orange-50 border-orange-200',
    icon: ArrowUp,
    label: 'Hoch',
  },
  medium: {
    color: 'text-blue-600 bg-blue-50 border-blue-200',
    icon: Minus,
    label: 'Normal',
  },
  low: {
    color: 'text-gray-500 bg-gray-50 border-gray-200',
    icon: ArrowDown,
    label: 'Niedrig',
  },
}

interface PriorityBadgeProps {
  priority: Priority
  compact?: boolean
  className?: string
}

export default function PriorityBadge({
  priority,
  compact = false,
  className,
}: PriorityBadgeProps) {
  const config = priorityConfig[priority] ?? priorityConfig.medium
  const Icon = config.icon

  if (compact) {
    return (
      <span
        className={cn(
          'inline-flex items-center justify-center rounded border p-0.5',
          config.color,
          className
        )}
        title={config.label}
      >
        <Icon className="h-3 w-3" />
      </span>
    )
  }

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium',
        config.color,
        className
      )}
    >
      <Icon className="h-3 w-3" />
      {config.label}
    </span>
  )
}

export { priorityConfig }
export type { Priority }
