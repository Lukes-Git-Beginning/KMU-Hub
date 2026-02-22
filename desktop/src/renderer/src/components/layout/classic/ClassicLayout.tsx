import { Suspense } from 'react'
import { Outlet } from 'react-router-dom'
import { Header } from '../Header'
import { OfflineBanner } from '../OfflineBanner'
import { ModuleLoadingFallback, ModuleErrorBoundary } from '../ModuleShell'
import { ClassicSidebar } from './ClassicSidebar'

export function ClassicLayout() {
  return (
    <div className="flex h-full bg-background overflow-hidden glass-surface">
      <ClassicSidebar />
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
