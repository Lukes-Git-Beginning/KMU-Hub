import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { PresenceStatusPicker } from '@/features/presence'
import { SearchBar, ProfileMenu } from '@/components/header'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { OfflineBanner } from '../OfflineBanner'
import { ModuleLoadingFallback, ModuleErrorBoundary } from '../ModuleShell'
import { DockBar } from './DockBar'

export function DockLayout() {
  const { isOnline } = useOnlineStatus()
  useNotificationWebSocket()

  return (
    <div className="flex h-full flex-col bg-background overflow-hidden glass-surface">
      {/* Mini top bar */}
      <header className="flex h-12 shrink-0 items-center border-b border-header-border bg-header-background px-4 glass-surface">
        <SearchBar />
        <div className="flex-1" />
        <div className="flex items-center gap-2">
          <PresenceStatusPicker />
          <NotificationBell />
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                className={`inline-block h-2 w-2 rounded-full ${isOnline ? 'bg-emerald-500' : 'bg-destructive'}`}
                aria-label={isOnline ? 'Online' : 'Offline'}
              />
            </TooltipTrigger>
            <TooltipContent>
              {isOnline ? 'Verbunden' : 'Keine Verbindung'}
            </TooltipContent>
          </Tooltip>
          <ProfileMenu />
        </div>
      </header>

      {/* Content */}
      <main className="flex-1 overflow-auto">
        <OfflineBanner />
        <ModuleErrorBoundary>
          <Suspense fallback={<ModuleLoadingFallback />}>
            <Outlet />
          </Suspense>
        </ModuleErrorBoundary>
      </main>

      {/* Floating dock at bottom */}
      <DockBar />
    </div>
  )
}
