/**
 * Main application shell — layout multiplexer.
 *
 * Reads `navLayout` from the UI store and renders the matching layout:
 *   - sidebar (default): classic left sidebar + header + content
 *   - dock: macOS-style floating bottom bar + mini top bar
 *   - topnav: horizontal scrollable tab bar at the top
 *   - classic: permanent icon-only sidebar (always collapsed)
 *
 * Global overlays (video, help, onboarding) are shared across all layouts.
 */
import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { useUIStore } from '@/stores/ui'
import { useWebSocket } from '@/hooks/useWebSocket'
import { useKeyboardShortcuts } from '@/hooks/useKeyboardShortcuts'
import { PresenceProvider } from '@/features/presence'
import { FloatingCallBar } from '@/features/video/FloatingCallBar'
import { IncomingCallOverlay } from '@/features/video/IncomingCallOverlay'
import { Sidebar } from './sidebar'
import { Header } from './Header'
import { OfflineBanner } from './OfflineBanner'
import { ModuleLoadingFallback, ModuleErrorBoundary } from './ModuleShell'
import { HelpWidget } from '@/components/widgets/HelpWidget'
import { OnboardingWizard } from '@/components/onboarding/OnboardingWizard'
import { DockLayout } from './dock'
import { TopNavLayout } from './topnav'
import { ClassicLayout } from './classic'

export function AppShell() {
  const navLayout = useUIStore((s) => s.navLayout)
  const onboardingCompleted = useUIStore((s) => s.onboardingCompleted)

  // Global lifecycle hooks (shared across all layouts)
  useWebSocket()
  useKeyboardShortcuts()

  return (
    <PresenceProvider>
      {navLayout === 'sidebar' ? (
        <SidebarShell />
      ) : navLayout === 'dock' ? (
        <DockLayout />
      ) : navLayout === 'topnav' ? (
        <TopNavLayout />
      ) : (
        <ClassicLayout />
      )}

      {/* Global overlays — shared across all layouts */}
      <FloatingCallBar />
      <IncomingCallOverlay />
      <HelpWidget />
      {!onboardingCompleted && <OnboardingWizard />}
    </PresenceProvider>
  )
}

/* ── Sidebar Layout (default) ── */
function SidebarShell() {
  const sidebarCollapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)
  const sidebarMobileOpen = useUIStore((s) => s.sidebarMobileOpen)
  const setSidebarMobileOpen = useUIStore((s) => s.setSidebarMobileOpen)

  return (
    <div className="flex h-full bg-background overflow-hidden glass-surface">
      {/* Mobile overlay */}
      {sidebarMobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setSidebarMobileOpen(false)}
        />
      )}

      <Sidebar
        collapsed={sidebarCollapsed}
        onToggle={toggleSidebar}
        isMobileOpen={sidebarMobileOpen}
        onMobileClose={() => setSidebarMobileOpen(false)}
      />

      <main className="flex flex-1 flex-col overflow-hidden">
        <OfflineBanner />
        <Header />

        <div className="flex-1 overflow-auto">
          <ModuleErrorBoundary>
            <Suspense fallback={<ModuleLoadingFallback />}>
              <Outlet />
            </Suspense>
          </ModuleErrorBoundary>
        </div>
      </main>
    </div>
  )
}
