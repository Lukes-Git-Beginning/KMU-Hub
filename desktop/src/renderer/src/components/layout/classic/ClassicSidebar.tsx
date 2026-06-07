import { useLocation, NavLink } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { useAuthStore } from '@/stores/auth'
import { useFilteredNavItems } from '@/hooks/useFilteredNavItems'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type { NavItemConfig } from '../sidebar/nav-items'

const MOCK_USER = { firstName: 'Darien', lastName: 'Müller' }

export function ClassicSidebar() {
  const { mainItems, bottomItems } = useFilteredNavItems()
  const storeUser = useAuthStore((s) => s.user)
  const user = storeUser ?? MOCK_USER
  const initials = `${user.firstName.charAt(0)}${user.lastName.charAt(0)}`

  return (
    <aside className="flex w-[72px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar glass-surface">
      {/* Branding */}
      <div className="flex h-14 items-center justify-center border-b border-sidebar-border">
        <span className="text-lg font-bold text-sidebar-primary">K</span>
      </div>

      {/* Main nav */}
      <nav className="flex-1 overflow-y-auto p-1.5">
        <ul className="flex flex-col items-center gap-1">
          {mainItems
            .filter((i) => i.enabled)
            .map((item) => (
              <li key={item.id}>
                <ClassicNavItem item={item} />
              </li>
            ))}
        </ul>
      </nav>

      {/* Bottom: Settings + User */}
      <div className="border-t border-sidebar-border p-1.5 space-y-1">
        <div className="flex flex-col items-center gap-1">
          {bottomItems
            .filter((i) => i.enabled)
            .map((item) => (
              <ClassicNavItem key={item.id} item={item} />
            ))}
        </div>

        <Tooltip delayDuration={100}>
          <TooltipTrigger asChild>
            <div className="flex justify-center py-2">
              <div className="relative">
                <Avatar className="h-8 w-8">
                  <AvatarFallback className="bg-sidebar-primary text-sidebar-primary-foreground text-xs">
                    {initials}
                  </AvatarFallback>
                </Avatar>
                <span className="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-sidebar bg-success" />
              </div>
            </div>
          </TooltipTrigger>
          <TooltipContent side="right">
            {user.firstName} {user.lastName}
          </TooltipContent>
        </Tooltip>
      </div>
    </aside>
  )
}

function ClassicNavItem({ item }: { item: NavItemConfig }) {
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
    <Tooltip delayDuration={100}>
      <TooltipTrigger asChild>
        <NavLink
          to={item.to}
          end={item.to === '/'}
          onClick={(e) => {
            if (item.onActivate) {
              e.preventDefault()
              item.onActivate()
            }
          }}
          className="relative flex items-center justify-center rounded-xl p-1.5 transition-all duration-150"
        >
          <div
            className={cn('nav-icon-box nav-icon-box-sm', isActive && 'active')}
            style={iconStyle}
          >
            <Icon className="h-[16px] w-[16px] relative z-[2]" />
          </div>

          {/* Badge dot */}
          {item.badge && (
            <span
              className={cn(
                'absolute top-0.5 right-0.5 h-2 w-2 rounded-full z-10',
                item.badge.type === 'live'
                  ? 'bg-destructive animate-pulse'
                  : 'bg-sidebar-primary'
              )}
            />
          )}
        </NavLink>
      </TooltipTrigger>
      <TooltipContent side="right">{item.label}</TooltipContent>
    </Tooltip>
  )
}
