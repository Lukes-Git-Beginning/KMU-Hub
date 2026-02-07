/**
 * Compact header bar displayed above the main content area.
 *
 * Shows current module name, online/offline indicator, and
 * a notification bell placeholder (wired in 05-04).
 */
import { useLocation } from 'react-router-dom'
import { Bell, Wifi, WifiOff } from 'lucide-react'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { Badge } from '@/components/ui/badge'
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

        {/* Notification bell placeholder (wired in 05-04) */}
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" className="relative h-9 w-9">
              <Bell className="h-4 w-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Benachrichtigungen</TooltipContent>
        </Tooltip>
      </div>
    </header>
  )
}
