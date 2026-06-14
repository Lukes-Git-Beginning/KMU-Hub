/**
 * Header bar displayed above the main content area.
 *
 * Shows search, clock, daily planner, time tracker, language switcher,
 * presence status, notifications, connection status, desk controls,
 * and profile menu.
 */
import { Menu } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useUIStore } from '@/stores/ui'
import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { PresenceStatusPicker } from '@/features/presence'
import { DEMO_MODE } from '@/mocks/demo-mode'
import {
  SearchBar,
  HeaderClock,
  WorkClockWidget,
  ProfileSwitcher,
  ProfileMenu,
  HeaderWidgetSlots,
  ConnectionStatusIndicator,
} from '@/components/header'

export function Header() {
  const { t } = useTranslation()
  const navLayout = useUIStore((s) => s.navLayout)
  const setSidebarMobileOpen = useUIStore((s) => s.setSidebarMobileOpen)

  // Initialize notification WebSocket listener so it's always active
  useNotificationWebSocket()

  return (
    <header data-tour="header" className="relative z-20 flex h-[72px] items-center border-b border-header-border bg-header-background px-[16px] md:px-[24px] glass-surface">
      {/* Left: demo badge + mobile menu + search + clock */}
      <div className="flex shrink-0 items-center gap-[12px] md:gap-[16px]">
        {DEMO_MODE && (
          <span className="shrink-0 rounded-md bg-amber-500/15 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-amber-600">
            Demo
          </span>
        )}
        {navLayout === 'sidebar' && (
          <button
            onClick={() => setSidebarMobileOpen(true)}
            className="shrink-0 rounded-lg p-2 hover:bg-accent lg:hidden"
            aria-label={t('components.layout.openNav')}
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
      <div className="flex-1 px-4">
        <HeaderWidgetSlots />
      </div>

      {/* Right: controls */}
      <div className="flex shrink-0 min-w-0 items-center gap-[8px] md:gap-[12px]">
        {/* Unified work clock (HR-API backed: clock-in/out, break, saldo) */}
        <div className="hidden lg:block">
          <WorkClockWidget />
        </div>

        {/* User presence status picker */}
        <PresenceStatusPicker />

        {/* Notification bell (existing, real API) */}
        <NotificationBell />

        {/* Connection status dot (browser + WebSocket state) */}
        <ConnectionStatusIndicator />

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
