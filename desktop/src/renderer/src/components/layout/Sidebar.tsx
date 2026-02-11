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
  FolderKanban,
  CalendarDays,
  Video,
  Users2,
  Bell,
  Cog,
  PanelLeftClose,
  PanelLeftOpen,
  LogOut,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useAuthStore } from '@/stores/auth'
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

interface NavItem {
  to: string
  icon: typeof LayoutDashboard
  label: string
  /** If set, only users with one of these roles see this item. */
  roles?: string[]
}

const navItems: NavItem[] = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/crm', icon: Users, label: 'CRM' },
  { to: '/chat', icon: MessageSquare, label: 'Chat' },
  { to: '/work', icon: FolderKanban, label: 'Aufgaben' },
  { to: '/calendar', icon: CalendarDays, label: 'Kalender' },
  { to: '/video', icon: Video, label: 'Video & Anrufe' },
  { to: '/meetings', icon: Users2, label: 'Meetings' },
  { to: '/notifications', icon: Bell, label: 'Benachrichtigungen' },
  { to: '/settings/dashboard', icon: Cog, label: 'Einstellungen', roles: ['admin'] },
]

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  const initials = user
    ? `${user.firstName.charAt(0)}${user.lastName.charAt(0)}`.toUpperCase()
    : '??'

  const displayName = user
    ? `${user.firstName} ${user.lastName}`
    : 'Benutzer'

  return (
    <aside
      className={cn(
        'flex flex-col border-r border-border bg-card transition-all duration-200',
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
        {navItems.filter((item) => {
          if (!item.roles) return true
          return item.roles.some((r) => user?.roles.includes(r))
        }).map((item) => (
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

        {/* Collapse toggle */}
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            'mt-1 w-full',
            collapsed && 'px-2'
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
    </aside>
  )
}
