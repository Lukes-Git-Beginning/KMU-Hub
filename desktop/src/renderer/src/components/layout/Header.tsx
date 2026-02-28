/**
 * Header bar displayed above the main content area.
 *
 * Shows search, clock, daily planner, time tracker, language switcher,
 * presence status, notifications, connection status, desk controls,
 * and profile menu.
 */
import { Menu } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useUIStore } from '@/stores/ui'

import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { PresenceStatusPicker } from '@/features/presence'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  SearchBar,
  HeaderClock,
  TimeTrackerWidget,
  ProfileSwitcher,
  ProfileMenu,
  HeaderWidgetSlots,
} from '@/components/header'

export function Header() {
  const { isOnline } = useOnlineStatus()
  const navLayout = useUIStore((s) => s.navLayout)
  const setSidebarMobileOpen = useUIStore((s) => s.setSidebarMobileOpen)

  // Initialize notification WebSocket listener so it's always active
  useNotificationWebSocket()

  return (
    <header data-tour="header" className="relative z-20 flex h-[64px] items-center border-b border-header-border bg-header-background px-[16px] md:px-[24px] glass-surface">
      {/* Left: mobile menu + search + clock */}
      <div className="flex shrink-0 items-center gap-3">
        {navLayout === 'sidebar' && (
          <button
            onClick={() => setSidebarMobileOpen(true)}
            className="shrink-0 rounded-lg p-2 hover:bg-accent lg:hidden"
            aria-label="Navigation öffnen"
          >
            <Menu className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
          </button>
        )}

        <div className="w-48 lg:w-64 xl:w-80">
          <SearchBar />
        </div>

        <div className="hidden md:block shrink-0">
          <HeaderClock />
        </div>
      </div>

      {/* Center: 3 fixed widget slots — fills the gap */}
      <div className="flex flex-1 items-center justify-center px-4">
        <HeaderWidgetSlots />
      </div>

      {/* Right: controls */}
      <div className="flex shrink-0 items-center gap-1 md:gap-2">
        {/* Unified Time Tracker (clock-in + task tracking) */}
        <div className="hidden lg:block">
          <TimeTrackerWidget />
        </div>

        {/* User presence status picker */}
        <PresenceStatusPicker />

        {/* Notification bell (existing, real API) */}
        <NotificationBell />

        {/* Connection status dot */}
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={`inline-block h-2.5 w-2.5 rounded-full ${
                isOnline ? 'bg-emerald-500' : 'bg-destructive'
              }`}
              role="status"
              aria-label={isOnline ? 'Online' : 'Offline'}
            />
          </TooltipTrigger>
          <TooltipContent>
            {isOnline
              ? 'Verbunden mit dem Server'
              : 'Keine Verbindung zum Server'}
          </TooltipContent>
        </Tooltip>

        {/* Profile Switcher */}
        <div className="hidden sm:block">
          <ProfileSwitcher />
        </div>

        {/* Profile Menu (avatar) */}
        <ProfileMenu />
      </div>
    </header>
  )
}
