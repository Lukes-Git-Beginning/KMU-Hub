/**
 * Compact header bar displayed above the main content area.
 *
 * Shows current module name, connection status dot, desk maximize toggle,
 * and the notification bell with real-time unread count.
 */
import { useLocation } from 'react-router-dom'
import { Maximize2, Minimize2 } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useUIStore } from '@/stores/ui'
import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

/** Map route paths to display names. */
function getModuleName(pathname: string): string {
  if (pathname === '/') return 'Dashboard'
  if (pathname.startsWith('/crm')) return 'CRM'
  if (pathname.startsWith('/chat')) return 'Chat'
  if (pathname.startsWith('/notifications')) return 'Benachrichtigungen'
  return 'KMU Hub'
}

export function Header() {
  const { pathname } = useLocation()
  const { isOnline } = useOnlineStatus()
  const moduleName = getModuleName(pathname)
  const deskMaximized = useUIStore((s) => s.deskMaximized)
  const toggleDeskMaximized = useUIStore((s) => s.toggleDeskMaximized)

  // Initialize notification WebSocket listener so it's always active
  useNotificationWebSocket()

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-card px-6">
      {/* Left: module name */}
      <h1 className="text-lg font-semibold text-foreground">{moduleName}</h1>

      {/* Right: status indicators */}
      <div className="flex items-center gap-3">
        {/* Connection status dot: green (online), red (offline) */}
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

        {/* Desk maximize toggle */}
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
            {deskMaximized ? 'Schreibtisch anzeigen' : 'Arbeitsflaeche maximieren'}
          </TooltipContent>
        </Tooltip>

        {/* Notification bell with real-time unread count */}
        <NotificationBell />
      </div>
    </header>
  )
}
