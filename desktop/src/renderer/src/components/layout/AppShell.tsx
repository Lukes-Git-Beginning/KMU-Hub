/**
 * Main application shell layout.
 *
 * Renders the sidebar, header, and content area. The content area
 * uses React Router's Outlet for nested route rendering, wrapped
 * in Suspense for lazy-loaded modules.
 */
import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { useUIStore } from '@/stores/ui'
import { useWebSocket } from '@/hooks/useWebSocket'
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
    <div className="flex h-full bg-background overflow-hidden">
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
    </div>
  )
}
