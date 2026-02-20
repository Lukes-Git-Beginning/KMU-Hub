import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { useAuthStore } from '@/stores/auth'
import { canSeeNavItem } from '@/config/roles'
import { isModuleAllowedForProfile } from '@/config/business-profiles'
import { useProfileStore } from '@/stores/profile'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { SidebarBadge } from './SidebarBadge'
import type { NavItemConfig } from './nav-items'

interface SidebarNavProps {
  items: NavItemConfig[]
  collapsed: boolean
  onItemClick?: () => void
}

export function SidebarNav({ items, collapsed, onItemClick }: SidebarNavProps) {
  const user = useAuthStore((s) => s.user)
  const businessProfileId = useProfileStore((s) => s.businessProfileId)
  const devShowAll = useProfileStore((s) => s.devShowAllModules)
  const enabledOptionals = useProfileStore((s) => s.enabledOptionalModules)

  const visibleItems = items.filter((item) => {
    // Layer 1: Role-based filtering (RBAC)
    if (!canSeeNavItem(user, item.id)) return false
    // Layer 2: Business profile filtering (skip if dev override)
    if (devShowAll) return true
    return isModuleAllowedForProfile(item.id, businessProfileId, enabledOptionals)
  })

  return (
    <nav className="flex-1 overflow-y-auto p-3">
      <ul className="space-y-1">
        {visibleItems.map((item) => (
          <li key={item.id}>
            <NavItem item={item} collapsed={collapsed} onItemClick={onItemClick} />
          </li>
        ))}
      </ul>
    </nav>
  )
}

function NavItem({
  item,
  collapsed,
  onItemClick,
}: {
  item: NavItemConfig
  collapsed: boolean
  onItemClick?: () => void
}) {
  const Icon = item.icon

  const sharedClasses = cn(
    'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-all duration-150',
    collapsed && 'justify-center px-2'
  )

  if (!item.enabled) {
    return (
      <Tooltip delayDuration={100}>
        <TooltipTrigger asChild>
          <div
            className={cn(
              sharedClasses,
              'cursor-not-allowed text-sidebar-muted/50'
            )}
          >
            <div className="relative shrink-0">
              <Icon className="h-5 w-5" />
            </div>
            {!collapsed && (
              <span className="flex-1">{item.label}</span>
            )}
          </div>
        </TooltipTrigger>
        <TooltipContent side={collapsed ? 'right' : 'top'}>
          {item.label} — Bald verfügbar
        </TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Tooltip delayDuration={collapsed ? 100 : 999999}>
      <TooltipTrigger asChild>
        <NavLink
          to={item.to}
          end={item.to === '/'}
          onClick={onItemClick}
          className={({ isActive }) =>
            cn(
              sharedClasses,
              isActive
                ? 'bg-sidebar-active text-sidebar-primary font-semibold shadow-sm border border-sidebar-border/60'
                : 'text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-foreground border border-transparent',
            )
          }
        >
          <div className="relative shrink-0">
            <Icon className="h-5 w-5" />
            {collapsed && item.badge && (
              <SidebarBadge badge={item.badge} collapsed />
            )}
          </div>
          {!collapsed && (
            <>
              <span className="flex-1">{item.label}</span>
              {item.badge && <SidebarBadge badge={item.badge} collapsed={false} />}
            </>
          )}
        </NavLink>
      </TooltipTrigger>
      {collapsed && (
        <TooltipContent side="right">{item.label}</TooltipContent>
      )}
    </Tooltip>
  )
}
