/**
 * Colored dot indicator showing a user's presence status.
 *
 * Reads from the Zustand presence store and renders a rounded dot
 * with a color matching the user's current presence level. Supports
 * three sizes for different contexts (avatar overlay, inline, etc.).
 *
 * Per user decision: only used in chat participant lists, DMs, and
 * team overview -- NOT in CRM or calendar views.
 */
import { cn } from '@/lib/cn'
import { usePresenceStore } from '@/stores/presence'
import type { PresenceLevel } from '@/api/video-types'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface PresenceIndicatorProps {
  userId: string
  size?: 'sm' | 'md' | 'lg'
  /** Additional CSS classes for positioning (e.g. absolute bottom-0 right-0). */
  className?: string
}

/** Map presence level to Tailwind background color class. */
const presenceColors: Record<PresenceLevel, string> = {
  online: 'bg-green-500',
  away: 'bg-yellow-500',
  dnd: 'bg-destructive',
  in_call: 'bg-purple-500',
  offline: 'bg-gray-400',
}

/** Map presence level to German display label. */
const presenceLabels: Record<PresenceLevel, string> = {
  online: 'Online',
  away: 'Abwesend',
  dnd: 'Nicht stoeren',
  in_call: 'Im Anruf',
  offline: 'Offline',
}

/** Size variants: width + height classes. */
const sizeClasses: Record<'sm' | 'md' | 'lg', string> = {
  sm: 'h-2 w-2',
  md: 'h-3 w-3',
  lg: 'h-4 w-4',
}

export function PresenceIndicator({
  userId,
  size = 'md',
  className,
}: PresenceIndicatorProps) {
  const status = usePresenceStore((s) => s.presenceMap[userId]) ?? 'offline'

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className={cn(
            'inline-block shrink-0 rounded-full ring-2 ring-white dark:ring-gray-900',
            sizeClasses[size],
            presenceColors[status],
            className,
          )}
          aria-label={presenceLabels[status]}
        />
      </TooltipTrigger>
      <TooltipContent side="top" className="text-xs">
        {presenceLabels[status]}
      </TooltipContent>
    </Tooltip>
  )
}
