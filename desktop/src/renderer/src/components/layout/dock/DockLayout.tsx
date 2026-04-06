import { useTranslation } from 'react-i18next'
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
import { ModuleErrorBoundary } from '../ModuleShell'
import { PageTransitionOutlet } from '../PageTransitionOutlet'
import { DockBar } from './DockBar'

export function DockLayout() {
  const { t } = useTranslation()
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
                className={`inline-block h-2 w-2 rounded-full ${isOnline ? 'bg-success' : 'bg-destructive'}`}
                role="status"
                aria-label={isOnline ? t('layout.dock.statusOnline') : t('layout.dock.statusOffline')}
              />
            </TooltipTrigger>
            <TooltipContent>
              {isOnline ? t('layout.dock.connected') : t('layout.dock.noConnection')}
            </TooltipContent>
          </Tooltip>
          <ProfileMenu />
        </div>
      </header>

      {/* Content */}
      <main id="main-content" className="flex-1 overflow-auto">
        <OfflineBanner />
        <ModuleErrorBoundary>
          <PageTransitionOutlet />
        </ModuleErrorBoundary>
      </main>

      {/* Floating dock at bottom */}
      <DockBar />
    </div>
  )
}
