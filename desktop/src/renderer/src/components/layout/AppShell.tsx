/**
 * Main application shell layout.
 *
 * Renders the sidebar, header, and content area. The content area
 * uses React Router's Outlet for nested route rendering, wrapped
 * in Suspense for lazy-loaded modules.
 *
 * Global overlays: PresenceProvider (heartbeat + WS presence),
 * FloatingCallBar (in-call status), IncomingCallOverlay (call notifications).
 */
import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { useUIStore } from '@/stores/ui'
import { useWebSocket } from '@/hooks/useWebSocket'
import { PresenceProvider } from '@/features/presence'
import { FloatingCallBar } from '@/features/video/FloatingCallBar'
import { IncomingCallOverlay } from '@/features/video/IncomingCallOverlay'
import { Sidebar } from './Sidebar'
import { Header } from './Header'
import { OfflineBanner } from './OfflineBanner'
import { ModuleLoadingFallback, ModuleErrorBoundary } from './ModuleShell'

export function AppShell() {
  const sidebarCollapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  // Manage WebSocket lifecycle based on auth state
  useWebSocket()

  return (
    <PresenceProvider>
      <div className="flex h-screen bg-background">
        <Sidebar collapsed={sidebarCollapsed} onToggle={toggleSidebar} />

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

        {/* Global video call overlays */}
        <FloatingCallBar />
        <IncomingCallOverlay />
      </div>
    </PresenceProvider>
  )
}
