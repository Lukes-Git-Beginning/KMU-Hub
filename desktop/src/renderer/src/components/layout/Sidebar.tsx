/**
 * App sidebar with navigation links and user info.
 *
 * Supports collapsed mode (icons only) and expanded mode (icons + text).
 * Navigation items use NavLink for active state highlighting.
 */
import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  Users,
  MessageSquare,
  Bell,
  PanelLeftClose,
  PanelLeftOpen,
  LogOut,
  Maximize2,
  Minimize2,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/crm', icon: Users, label: 'CRM' },
  { to: '/chat', icon: MessageSquare, label: 'Chat' },
  { to: '/notifications', icon: Bell, label: 'Benachrichtigungen' },
] as const

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const deskMaximized = useUIStore((s) => s.deskMaximized)
  const toggleDeskMaximized = useUIStore((s) => s.toggleDeskMaximized)

  const initials = user
    ? `${user.firstName.charAt(0)}${user.lastName.charAt(0)}`.toUpperCase()
    : '??'

  const displayName = user
    ? `${user.firstName} ${user.lastName}`
    : 'Benutzer'

  return (
    <aside
      className={cn(
        'flex flex-col border-r transition-all duration-200',
        'bg-[var(--desk-sidebar-bg)] border-[var(--desk-sidebar-border)]',
        collapsed ? 'w-16' : 'w-64'
      )}
    >
      {/* Logo / Brand */}
      <div className="flex h-14 items-center border-b border-border px-4">
        {!collapsed && (
          <span className="text-lg font-bold tracking-tight text-foreground">
            KMU Hub
          </span>
        )}
        {collapsed && (
          <span className="mx-auto text-lg font-bold text-foreground">K</span>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => (
          <Tooltip key={item.to} delayDuration={collapsed ? 100 : 999999}>
            <TooltipTrigger asChild>
              <NavLink
                to={item.to}
                end={item.to === '/'}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-secondary text-secondary-foreground'
                      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
                    collapsed && 'justify-center px-2'
                  )
                }
              >
                <item.icon className="h-5 w-5 shrink-0" />
                {!collapsed && <span>{item.label}</span>}
              </NavLink>
            </TooltipTrigger>
            {collapsed && (
              <TooltipContent side="right">
                {item.label}
              </TooltipContent>
            )}
          </Tooltip>
        ))}
      </nav>

      {/* Bottom section: user + collapse toggle */}
      <div className="border-t border-border p-2">
        {/* User info */}
        <div
          className={cn(
            'flex items-center gap-3 rounded-md px-3 py-2',
            collapsed && 'justify-center px-2'
          )}
        >
          <Avatar className="h-8 w-8 shrink-0">
            <AvatarFallback className="text-xs">{initials}</AvatarFallback>
          </Avatar>
          {!collapsed && (
            <div className="flex-1 min-w-0">
              <p className="truncate text-sm font-medium text-foreground">
                {displayName}
              </p>
              <p className="truncate text-xs text-muted-foreground">
                {user?.email}
              </p>
            </div>
          )}
          {!collapsed && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 shrink-0"
                  onClick={() => logout()}
                >
                  <LogOut className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Abmelden</TooltipContent>
            </Tooltip>
          )}
        </div>

        {/* Maximize + Collapse toggles */}
        <div className={cn('mt-1 flex gap-1', collapsed ? 'flex-col' : 'flex-row')}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className={cn(collapsed ? 'w-full px-2' : 'px-2')}
                onClick={toggleDeskMaximized}
              >
                {deskMaximized ? (
                  <Minimize2 className="h-4 w-4" />
                ) : (
                  <Maximize2 className="h-4 w-4" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">
              {deskMaximized ? 'Schreibtisch anzeigen' : 'Arbeitsflaeche maximieren'}
            </TooltipContent>
          </Tooltip>

          <Button
            variant="ghost"
            size="sm"
            className={cn(
              collapsed ? 'w-full px-2' : 'flex-1'
            )}
            onClick={onToggle}
          >
            {collapsed ? (
              <PanelLeftOpen className="h-4 w-4" />
            ) : (
              <>
                <PanelLeftClose className="h-4 w-4" />
                <span className="ml-2">Einklappen</span>
              </>
            )}
          </Button>
        </div>
      </div>
    </aside>
  )
}
