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
import type { NavLayout } from '@/stores/ui'
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
  DailyPlannerWidget,
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
      {/* Left: mobile menu + compact search + clock */}
      <div className="flex items-center gap-[12px] md:gap-[16px]">
        {navLayout === 'sidebar' && (
          <button
            onClick={() => setSidebarMobileOpen(true)}
            className="shrink-0 rounded-lg p-2 hover:bg-accent lg:hidden"
            aria-label="Navigation öffnen"
          >
            <Menu className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
          </button>
        )}

        <SearchBar />

        <div className="hidden sm:block">
          <HeaderClock />
        </div>

        {/* Configurable header widget slots */}
        <HeaderWidgetSlots />
      </div>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Right: controls */}
      <div className="flex min-w-0 items-center gap-[4px] md:gap-[8px]">
        {/* Daily Planner */}
        <div className="hidden sm:block">
          <DailyPlannerWidget />
        </div>

        {/* Unified Time Tracker (clock-in + task tracking) */}
        <div className="hidden sm:block">
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
