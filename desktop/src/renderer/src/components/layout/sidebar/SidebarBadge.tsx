import { cn } from '@/lib/cn'
import type { NavBadge } from './nav-items'

interface SidebarBadgeProps {
  badge: NavBadge
  collapsed: boolean
}

export function SidebarBadge({ badge, collapsed }: SidebarBadgeProps) {
  if (collapsed) {
    return (
      <span
        className={cn(
          'absolute -top-1 -right-1 h-2 w-2 rounded-full',
          badge.type === 'live' ? 'bg-error animate-pulse-live' : 'bg-sidebar-primary'
        )}
      />
    )
  }

  if (badge.type === 'live') {
    return (
      <span className="inline-flex items-center gap-1 rounded-full bg-error/10 px-2 py-0.5 text-xs font-medium text-error">
        <span className="h-1.5 w-1.5 rounded-full bg-error animate-pulse-live" />
        Live
      </span>
    )
  }

  return (
    <span className="rounded-full bg-sidebar-primary/10 px-2 py-0.5 text-xs font-medium text-sidebar-primary">
      {badge.value}
    </span>
  )
}
