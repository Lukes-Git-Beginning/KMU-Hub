import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { OfflineBanner } from '../OfflineBanner'
import { ModuleLoadingFallback, ModuleErrorBoundary } from '../ModuleShell'
import { TopNavBar } from './TopNavBar'

export function TopNavLayout() {
  return (
    <div className="flex h-full flex-col bg-background overflow-hidden glass-surface">
      <TopNavBar />
      <main className="flex-1 overflow-auto">
        <OfflineBanner />
        <ModuleErrorBoundary>
          <Suspense fallback={<ModuleLoadingFallback />}>
            <Outlet />
          </Suspense>
        </ModuleErrorBoundary>
      </main>
    </div>
  )
}
