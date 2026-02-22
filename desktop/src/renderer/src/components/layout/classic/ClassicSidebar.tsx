import { NavLink } from 'react-router-dom'
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

const MOCK_USER = { firstName: 'Darien', lastName: 'Mueller' }

export function ClassicSidebar() {
  const { mainItems, bottomItems } = useFilteredNavItems()
  const storeUser = useAuthStore((s) => s.user)
  const user = storeUser ?? MOCK_USER
  const initials = `${user.firstName.charAt(0)}${user.lastName.charAt(0)}`

  return (
    <aside className="flex w-16 shrink-0 flex-col border-r border-sidebar-border bg-sidebar glass-surface">
      {/* Branding */}
      <div className="flex h-14 items-center justify-center border-b border-sidebar-border">
        <span className="text-lg font-bold text-sidebar-primary">K</span>
      </div>

      {/* Main nav */}
      <nav className="flex-1 overflow-y-auto p-2">
        <ul className="space-y-1">
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
      <div className="border-t border-sidebar-border p-2 space-y-1">
        {bottomItems
          .filter((i) => i.enabled)
          .map((item) => (
            <ClassicNavItem key={item.id} item={item} />
          ))}

        <Tooltip delayDuration={100}>
          <TooltipTrigger asChild>
            <div className="flex justify-center py-2">
              <div className="relative">
                <Avatar className="h-8 w-8">
                  <AvatarFallback className="bg-sidebar-primary text-sidebar-primary-foreground text-xs">
                    {initials}
                  </AvatarFallback>
                </Avatar>
                <span className="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-sidebar bg-emerald-500" />
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

  return (
    <Tooltip delayDuration={100}>
      <TooltipTrigger asChild>
        <NavLink
          to={item.to}
          end={item.to === '/'}
          className={({ isActive }) =>
            cn(
              'flex items-center justify-center rounded-lg p-2 transition-all duration-150',
              isActive
                ? 'bg-sidebar-active text-sidebar-primary shadow-sm'
                : 'text-sidebar-foreground/60 hover:bg-sidebar-accent hover:text-sidebar-foreground'
            )
          }
        >
          <div className="relative">
            <Icon className="h-5 w-5" />
            {item.badge && (
              <span
                className={cn(
                  'absolute -top-1 -right-1 h-2 w-2 rounded-full',
                  item.badge.type === 'live'
                    ? 'bg-red-500 animate-pulse'
                    : 'bg-sidebar-primary'
                )}
              />
            )}
          </div>
        </NavLink>
      </TooltipTrigger>
      <TooltipContent side="right">{item.label}</TooltipContent>
    </Tooltip>
  )
}
