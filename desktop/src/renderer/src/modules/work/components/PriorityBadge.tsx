/**
 * Color-coded priority indicator badge with icon.
 *
 * Supports full mode (icon + text) and compact mode (icon only) for
 * space-constrained contexts like Kanban cards.
 */
import { useTranslation } from 'react-i18next'
import { AlertTriangle, ArrowUp, Minus, ArrowDown } from 'lucide-react'
import { cn } from '@/lib'

type Priority = 'urgent' | 'high' | 'medium' | 'low'

const priorityConfig: Record<
  Priority,
  {
    color: string
    icon: React.ComponentType<{ className?: string }>
    labelKey: string
  }
> = {
  urgent: {
    color: 'text-destructive bg-error-light border-destructive/30',
    icon: AlertTriangle,
    labelKey: 'work.priority.urgent',
  },
  high: {
    color: 'text-orange-600 bg-orange-50 border-orange-200 dark:text-orange-400 dark:bg-orange-950 dark:border-orange-800',
    icon: ArrowUp,
    labelKey: 'work.priority.high',
  },
  medium: {
    color: 'text-blue-600 bg-blue-50 border-blue-200 dark:text-blue-400 dark:bg-blue-950 dark:border-blue-800',
    icon: Minus,
    labelKey: 'work.priority.normal',
  },
  low: {
    color: 'text-gray-500 bg-gray-50 border-gray-200 dark:text-gray-400 dark:bg-gray-800 dark:border-gray-700',
    icon: ArrowDown,
    labelKey: 'work.priority.low',
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
  const { t } = useTranslation()
  const config = priorityConfig[priority] ?? priorityConfig.medium
  const Icon = config.icon
  const label = t(config.labelKey)

  if (compact) {
    return (
      <span
        className={cn(
          'inline-flex items-center justify-center rounded border p-0.5',
          config.color,
          className
        )}
        title={label}
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
      {label}
    </span>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export { priorityConfig }
export type { Priority }
