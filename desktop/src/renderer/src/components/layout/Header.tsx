/**
 * Compact header bar displayed above the main content area.
 *
 * Shows current module name, online/offline indicator, and the
 * notification bell with real-time unread count.
 */
import { useLocation } from 'react-router-dom'
import { Wifi, WifiOff } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { Badge } from '@/components/ui/badge'
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

  // Initialize notification WebSocket listener so it's always active
  useNotificationWebSocket()

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-card px-6">
      {/* Left: module name */}
      <h1 className="text-lg font-semibold text-foreground">{moduleName}</h1>

      {/* Right: status indicators */}
      <div className="flex items-center gap-3">
        {/* Online / Offline indicator */}
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex items-center gap-1.5">
              {isOnline ? (
                <Wifi className="h-4 w-4 text-emerald-500" />
              ) : (
                <WifiOff className="h-4 w-4 text-destructive" />
              )}
              <Badge
                variant={isOnline ? 'secondary' : 'destructive'}
                className="text-xs"
              >
                {isOnline ? 'Online' : 'Offline'}
              </Badge>
            </div>
          </TooltipTrigger>
          <TooltipContent>
            {isOnline
              ? 'Verbunden mit dem Server'
              : 'Keine Verbindung zum Server'}
          </TooltipContent>
        </Tooltip>

        {/* Notification bell with real-time unread count */}
        <NotificationBell />
      </div>
    </header>
  )
}
