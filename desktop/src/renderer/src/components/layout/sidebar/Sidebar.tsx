/**
 * Redesigned sidebar — single column, pinned modules, module overview panel.
 *
 * Features:
 * - 1-column layout: Icon + label side-by-side (horizontal)
 * - Narrower: w-56 expanded (224px), w-16 collapsed (64px)
 * - Shows only pinned/favorite modules
 * - Grid button opens all-modules popover panel
 * - Badge dots on icons
 * - Mobile drawer + tablet auto-collapse
 */
import { useState, useEffect, useRef } from 'react'
import { cn } from '@/lib/cn'
import { useUIStore } from '@/stores/ui'
import { useMediaQuery } from '@/hooks/useMediaQuery'
import { useFilteredNavItems } from '@/hooks/useFilteredNavItems'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Shield } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useSettingsStore } from '@/stores/settings'
import { useTenantMode } from '@/hooks/useTenantMode'
import { useAuthStore } from '@/stores/auth'
import { userHasRole } from '@/config/roles'
import { SidebarBranding } from './SidebarBranding'
import { SidebarNav } from './SidebarNav'
import { SidebarModulePanel } from './SidebarModulePanel'
import type { NavItemConfig } from './nav-items'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  isMobileOpen?: boolean
  onMobileClose?: () => void
}

export function Sidebar({ collapsed, onToggle, isMobileOpen = false, onMobileClose }: SidebarProps) {
  const navigate = useNavigate()
  const { isDistributed } = useTenantMode()
  const user = useAuthStore((s) => s.user)
  const canSeeAdminHub = isDistributed && userHasRole(user, ['admin', 'it_support'])
  const security = useSettingsStore((s) => s.security)
  const pwLastChanged = security.passwordLastChanged ? new Date(security.passwordLastChanged) : null
  const pwExpiryDays = security.passwordExpiryDays || 90
  // eslint-disable-next-line react-hooks/purity -- reading current date to compute password expiry status
  const daysSincePwChange = pwLastChanged ? Math.floor((Date.now() - pwLastChanged.getTime()) / (1000 * 60 * 60 * 24)) : 0
  const pwDaysLeft = Math.max(0, pwExpiryDays - daysSincePwChange)
  const showPwWarning = pwExpiryDays > 0 && pwDaysLeft <= 14
  const isTablet = useMediaQuery('(min-width: 768px) and (max-width: 1199px)')
  const didAutoCollapse = useRef(false)
  const [modulePanelOpen, setModulePanelOpen] = useState(false)

  const pinnedModules = useUIStore((s) => s.pinnedModules)
  const { mainItems, bottomItems, allItems } = useFilteredNavItems()

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

  // Filter main items to only pinned ones (preserve pin order)
  const pinnedItems = pinnedModules
    .map((id) => mainItems.find((item) => item.id === id))
    .filter((item): item is NavItemConfig => item != null)

  return (
    <aside
      data-tour="sidebar"
      className={cn(
        'flex flex-col bg-sidebar border-r border-sidebar-border glass-surface',
        'fixed lg:static inset-y-0 left-0 z-50',
        'transform transition-all duration-300 ease-in-out',
        isMobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        collapsed ? 'lg:w-16' : 'lg:w-56',
        'w-56'
      )}
    >
      {/* Branding header with grid button — wraps the popover anchor */}
      <Popover open={modulePanelOpen} onOpenChange={setModulePanelOpen}>
        <PopoverTrigger asChild>
          <div>
            <SidebarBranding
              collapsed={collapsed}
              onToggle={onToggle}
              onOpenModules={() => setModulePanelOpen((o) => !o)}
            />
          </div>
        </PopoverTrigger>
        <PopoverContent
          side="right"
          align="start"
          className="w-80 p-0"
          sideOffset={4}
        >
          <SidebarModulePanel
            items={allItems}
            onSelect={() => {
              setModulePanelOpen(false)
              onMobileClose?.()
            }}
          />
        </PopoverContent>
      </Popover>

      {/* Main nav — only pinned modules */}
      <SidebarNav items={pinnedItems} collapsed={collapsed} onItemClick={onMobileClose} />

      {/* Bottom: Security warning + Benachrichtigungen + Modul-Einstellungen.
          The user/account lives only in the top-right ProfileMenu (avatar) — no
          duplicate user block here. */}
      <div className="mt-auto border-t border-sidebar-border p-2 space-y-0.5">
        {showPwWarning && (
          <button
            onClick={() => navigate('/settings')}
            className={cn(
              'flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-xs transition-colors',
              pwDaysLeft === 0 ? 'bg-error/10 text-error' : 'bg-warning/10 text-warning',
              'hover:opacity-80'
            )}
          >
            <Shield className="h-3.5 w-3.5 shrink-0" />
            {!collapsed && (
              <span className="truncate">
                {pwDaysLeft === 0 ? 'PW abgelaufen' : `PW: ${pwDaysLeft}d`}
              </span>
            )}
          </button>
        )}
        <SidebarNav items={bottomItems} collapsed={collapsed} onItemClick={onMobileClose} />
      </div>
    </aside>
  )
}
