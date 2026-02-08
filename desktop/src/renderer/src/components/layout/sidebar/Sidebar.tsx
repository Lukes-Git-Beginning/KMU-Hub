/**
 * Redesigned sidebar with navigation, badges, branding, and responsive behavior.
 *
 * Features:
 * - 10 navigation items (enabled routes + disabled "coming soon")
 * - Badge system (text counters, live indicator)
 * - Branding header with collapse toggle
 * - User profile with online status
 * - Mobile drawer (fixed overlay) + tablet auto-collapse
 * - Figma color tokens (sidebar-*)
 */
import { useEffect, useRef } from 'react'
import { cn } from '@/lib/cn'
import { useAuthStore } from '@/stores/auth'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import { navItems } from './nav-items'
import { SidebarBranding } from './SidebarBranding'
import { SidebarNav } from './SidebarNav'
import { SidebarUser } from './SidebarUser'
import { NavLink } from 'react-router-dom'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  isMobileOpen?: boolean
  onMobileClose?: () => void
}

export function Sidebar({ collapsed, onToggle, isMobileOpen = false, onMobileClose }: SidebarProps) {
  const user = useAuthStore((s) => s.user)
  const isTablet = useMediaQuery('(min-width: 768px) and (max-width: 1199px)')
  const didAutoCollapse = useRef(false)

  // Auto-collapse on tablet
  useEffect(() => {
    if (isTablet && !collapsed && !didAutoCollapse.current) {
      didAutoCollapse.current = true
      onToggle()
    }
    if (!isTablet) {
      didAutoCollapse.current = false
    }
  }, [isTablet, collapsed, onToggle])

  const mainItems = navItems.filter((item) => item.section === 'main')
  const bottomItems = navItems.filter((item) => item.section === 'bottom').filter((item) => {
    if (!item.roles) return true
    return item.roles.some((r) => user?.roles?.includes(r))
  })

  return (
    <aside
      className={cn(
        'flex flex-col bg-sidebar border-r border-sidebar-border',
        'fixed lg:static inset-y-0 left-0 z-50',
        'transform transition-all duration-300 ease-in-out',
        isMobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        collapsed ? 'lg:w-16' : 'lg:w-64',
        'w-64'
      )}
    >
      <SidebarBranding collapsed={collapsed} onToggle={onToggle} />

      <SidebarNav items={mainItems} collapsed={collapsed} onItemClick={onMobileClose} />

      {/* Bottom: Settings + User */}
      <div className="mt-auto border-t border-sidebar-border">
        {bottomItems.map((item) => {
          const Icon = item.icon
          return (
            <div key={item.id} className="p-3 pb-0">
              <Tooltip delayDuration={collapsed ? 100 : 999999}>
                <TooltipTrigger asChild>
                  <NavLink
                    to={item.to}
                    onClick={onMobileClose}
                    className={({ isActive }) =>
                      cn(
                        'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                        isActive
                          ? 'border-l-2 border-sidebar-active-border bg-sidebar-active text-sidebar-primary'
                          : 'text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
                        collapsed && 'justify-center px-2',
                        collapsed && isActive && 'border-l-0'
                      )
                    }
                  >
                    <Icon className="h-5 w-5 shrink-0" />
                    {!collapsed && <span>{item.label}</span>}
                  </NavLink>
                </TooltipTrigger>
                {collapsed && (
                  <TooltipContent side="right">{item.label}</TooltipContent>
                )}
              </Tooltip>
            </div>
          )
        })}

        <SidebarUser collapsed={collapsed} />
      </div>
    </aside>
  )
}
