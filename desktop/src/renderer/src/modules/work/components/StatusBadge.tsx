/**
 * Colored badge displaying a task status with optional closed indicator.
 *
 * Renders as a small pill with the status color as background (with opacity)
 * and text. When is_closed, adds a checkmark icon to indicate completion.
 */
import { Check } from 'lucide-react'
import { cn } from '@/lib'

interface StatusBadgeProps {
  name: string
  color?: string
  isClosed?: boolean
  className?: string
}

export default function StatusBadge({
  name,
  color = '#6b7280',
  isClosed,
  className,
}: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium border',
        className
      )}
      style={{
        backgroundColor: `${color}15`,
        color: color,
        borderColor: `${color}40`,
      }}
    >
      {isClosed && <Check className="h-3 w-3" />}
      {name}
    </span>
  )
}
