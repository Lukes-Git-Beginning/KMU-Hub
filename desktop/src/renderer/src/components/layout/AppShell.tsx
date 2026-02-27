/**
 * Main application shell — stable layout multiplexer.
 *
 * Reads `navLayout` from the UI store and swaps only the navigation
 * chrome (sidebar, header, dock, topnav) while keeping the content
 * area (`PageTransitionOutlet`) mounted at a stable tree position.
 *
 * This prevents page components from losing local state when the user
 * switches layouts (e.g. from Settings > Darstellung).
 *
 * Layouts:
 *   - sidebar (default): left sidebar + full header
 *   - dock: mini top bar + floating bottom dock
 *   - topnav: horizontal scrollable tab bar
 *   - classic: permanent icon-only sidebar (always collapsed)
 */
import { useUIStore } from '@/stores/ui'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { useOnlineStatus } from '@/hooks/useOnlineStatus'
import { useNotificationWebSocket } from '@/api/hooks/useNotifications'
import { PresenceProvider } from '@/features/presence'
import { PresenceStatusPicker } from '@/features/presence'
import { FloatingCallBar } from '@/features/video/FloatingCallBar'
import { IncomingCallOverlay } from '@/features/video/IncomingCallOverlay'
import { NotificationBell } from '@/modules/notifications/NotificationBell'
import { SearchBar, ProfileMenu } from '@/components/header'
import { Sidebar } from './sidebar'
import { Header } from './Header'
import { OfflineBanner } from './OfflineBanner'
import { ModuleErrorBoundary } from './ModuleShell'
import { PageTransitionOutlet } from './PageTransitionOutlet'
import { OnboardingWizard } from '@/components/onboarding/OnboardingWizard'
import { TourOverlay } from '@/components/shared/TourOverlay'
import { PasswordExpiryDialog } from '@/components/shared/PasswordExpiryDialog'
import { DockBar } from './dock/DockBar'
import { TopNavBar } from './topnav/TopNavBar'
import { ClassicSidebar } from './classic/ClassicSidebar'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

export function AppShell() {
  const navLayout = useUIStore((s) => s.navLayout)
  const onboardingCompleted = useUIStore((s) => s.onboardingCompleted)
  const sidebarCollapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)
  const sidebarMobileOpen = useUIStore((s) => s.sidebarMobileOpen)
  const setSidebarMobileOpen = useUIStore((s) => s.setSidebarMobileOpen)

  // Global lifecycle hooks (shared across all layouts)
  useWebSocket()
  useKeyboardShortcuts()

  const isSidebar = navLayout === 'sidebar'
  const isClassic = navLayout === 'classic'
  const isDock = navLayout === 'dock'
  const isTopNav = navLayout === 'topnav'

  return (
    <PresenceProvider>
      {/* Skip-to-content link for keyboard navigation */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[200] focus:rounded-lg focus:bg-primary focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-primary-foreground focus:shadow-lg"
      >
        Zum Hauptinhalt springen
      </a>

      <div className="flex h-full bg-background overflow-hidden glass-surface">
        {/* Mobile sidebar overlay */}
        {isSidebar && sidebarMobileOpen && (
          <div
            className="fixed inset-0 z-40 bg-black/50 lg:hidden"
            onClick={() => setSidebarMobileOpen(false)}
          />
        )}

        {/* Left navigation (sidebar variants) */}
        {isSidebar && (
          <Sidebar
            collapsed={sidebarCollapsed}
            onToggle={toggleSidebar}
            isMobileOpen={sidebarMobileOpen}
            onMobileClose={() => setSidebarMobileOpen(false)}
          />
        )}
        {isClassic && <ClassicSidebar />}

        {/* Main column — content area stays mounted across layout switches */}
        <main className="flex flex-1 flex-col overflow-hidden">
          {/* Top chrome */}
          <OfflineBanner />
          {isDock && <DockHeader />}
          {(isSidebar || isClassic || isTopNav) && <Header />}
          {isTopNav && <TopNavBar />}

          {/* Content — stable position, never unmounts */}
          <div id="main-content" className="flex flex-1 flex-col min-h-0 overflow-auto">
            <ModuleErrorBoundary>
              <PageTransitionOutlet />
            </ModuleErrorBoundary>
          </div>

          {/* Bottom dock */}
          {isDock && <DockBar />}
        </main>
      </div>

      {/* Global overlays — shared across all layouts */}
      <FloatingCallBar />
      <IncomingCallOverlay />
      <TourOverlay />
      <PasswordExpiryDialog />
      {!onboardingCompleted && <OnboardingWizard />}
    </PresenceProvider>
  )
}

/* ── Dock mini-header (extracted from DockLayout) ── */
function DockHeader() {
  const { isOnline } = useOnlineStatus()
  useNotificationWebSocket()

  return (
    <header className="relative z-20 flex h-12 shrink-0 items-center border-b border-header-border bg-header-background px-4 glass-surface">
      <SearchBar />
      <div className="flex-1" />
      <div className="flex items-center gap-2">
        <PresenceStatusPicker />
        <NotificationBell />
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={`inline-block h-2 w-2 rounded-full ${isOnline ? 'bg-emerald-500' : 'bg-destructive'}`}
              role="status"
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
  )
}
