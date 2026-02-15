import { Maximize2, Minimize2, Menu } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useUIStore } from '@/stores/ui'
import { DESK_THEMES } from '@/config/desk-themes'
import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { Button } from '@/components/ui/button'
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
  LanguageSwitcher,
  ProfileSwitcher,
  ProfileMenu,
} from '@/components/header'

export function Header() {
  const { isOnline } = useOnlineStatus()
  const deskMaximized = useUIStore((s) => s.deskMaximized)
  const deskThemeId = useUIStore((s) => s.deskThemeId)
  const toggleDeskMaximized = useUIStore((s) => s.toggleDeskMaximized)
  const setSidebarMobileOpen = useUIStore((s) => s.setSidebarMobileOpen)
  const isMinimalTheme = DESK_THEMES[deskThemeId]?.isMinimal ?? false

  // Initialize notification WebSocket listener so it's always active
  useNotificationWebSocket()

  return (
    <header className="flex h-16 items-center border-b border-header-border bg-header-background px-4 md:px-6 glass-surface">
      {/* Left: mobile menu + compact search + clock */}
      <div className="flex items-center gap-3 md:gap-4">
        <button
          onClick={() => setSidebarMobileOpen(true)}
          className="shrink-0 rounded-lg p-2 hover:bg-accent lg:hidden"
        >
          <Menu className="h-5 w-5 text-muted-foreground" />
        </button>

        <SearchBar />

        <div className="hidden sm:block">
          <HeaderClock />
        </div>
      </div>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Right: controls */}
      <div className="flex shrink-0 items-center gap-1 md:gap-2">
        {/* Daily Planner */}
        <div className="hidden sm:block">
          <DailyPlannerWidget />
        </div>

        {/* Time Tracker */}
        <div className="hidden sm:block">
          <TimeTrackerWidget />
        </div>

        {/* Language Switcher */}
        <div className="hidden md:block">
          <LanguageSwitcher />
        </div>

        {/* Notification bell (existing, real API) */}
        <NotificationBell />

        {/* Connection status dot */}
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={`inline-block h-2.5 w-2.5 rounded-full ${
                isOnline ? 'bg-emerald-500' : 'bg-destructive'
              }`}
              aria-label={isOnline ? 'Online' : 'Offline'}
            />
          </TooltipTrigger>
          <TooltipContent>
            {isOnline
              ? 'Verbunden mit dem Server'
              : 'Keine Verbindung zum Server'}
          </TooltipContent>
        </Tooltip>

        {/* Desk maximize toggle — hidden for minimal theme */}
        {!isMinimalTheme && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-9 w-9"
                onClick={toggleDeskMaximized}
              >
                {deskMaximized ? (
                  <Minimize2 className="h-4 w-4" />
                ) : (
                  <Maximize2 className="h-4 w-4" />
                )}
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {deskMaximized
                ? 'Schreibtisch anzeigen'
                : 'Arbeitsflaeche maximieren'}
            </TooltipContent>
          </Tooltip>
        )}

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
