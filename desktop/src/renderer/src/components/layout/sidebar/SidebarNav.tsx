import { useLocation, NavLink } from 'react-router-dom'
import { cn } from '@/lib/cn'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { NavItemConfig } from './nav-items'

interface SidebarNavProps {
  items: NavItemConfig[]
  collapsed: boolean
  onItemClick?: () => void
}

export function SidebarNav({ items, collapsed, onItemClick }: SidebarNavProps) {
  return (
    <nav className="flex-1 overflow-y-auto px-2 py-1.5">
      <div className="flex flex-col gap-0.5">
        {items.map((item) => (
          <NavItem
            key={item.id}
            item={item}
            collapsed={collapsed}
            onItemClick={onItemClick}
          />
        ))}
      </div>
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
  const location = useLocation()

  const isActive =
    item.to === '/'
      ? location.pathname === '/'
      : location.pathname.startsWith(item.to)

  const iconStyle = {
    '--_nav-h': item.color.h,
    '--_nav-s': `${item.color.s}%`,
  } as React.CSSProperties

  return (
    <Tooltip delayDuration={collapsed ? 100 : 999999}>
      <TooltipTrigger asChild>
        <NavLink
          to={item.to}
          end={item.to === '/'}
          onClick={onItemClick}
          className={cn(
            'group relative flex items-center rounded-lg transition-all duration-150',
            collapsed ? 'justify-center p-2' : 'gap-2.5 px-2.5 py-1.5',
            isActive
              ? 'bg-sidebar-active text-sidebar-foreground'
              : 'text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground'
          )}
        >
          {/* Icon */}
          <div
            className={cn('nav-icon-box nav-icon-box-sm', isActive && 'active')}
            style={iconStyle}
          >
            <Icon className="h-4 w-4 relative z-[2]" />
          </div>

          {/* Label */}
          {!collapsed && (
            <span
              className={cn(
                'text-sm leading-tight truncate',
                isActive ? 'font-semibold' : 'font-medium'
              )}
            >
              {item.label}
            </span>
          )}

          {/* Badge dot */}
          {item.badge && (
            <div className={cn(
              'absolute z-10',
              collapsed ? '-top-0.5 -right-0.5' : 'top-1/2 -translate-y-1/2 right-2'
            )}>
              {item.badge.type === 'live' ? (
                <span className="flex h-2 w-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-destructive/75 opacity-75" />
                  <span className="relative inline-flex h-2 w-2 rounded-full bg-destructive" />
                </span>
              ) : item.badge.value ? (
                <span className="flex h-4.5 min-w-4.5 items-center justify-center rounded-full bg-primary/15 px-1 text-[10px] font-semibold text-primary">
                  {item.badge.value}
                </span>
              ) : null}
            </div>
          )}
        </NavLink>
      </TooltipTrigger>
      {collapsed && (
        <TooltipContent side="right">{item.label}</TooltipContent>
      )}
    </Tooltip>
  )
}
