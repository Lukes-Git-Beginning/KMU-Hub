import { Header } from '../Header'
import { OfflineBanner } from '../OfflineBanner'
import { ModuleErrorBoundary } from '../ModuleShell'
import { PageTransitionOutlet } from '../PageTransitionOutlet'
import { ClassicSidebar } from './ClassicSidebar'

export function ClassicLayout() {
  return (
    <div className="flex h-full bg-background overflow-hidden glass-surface">
      <ClassicSidebar />
      <main className="flex flex-1 flex-col overflow-hidden">
        <OfflineBanner />
        <Header />
        <div id="main-content" className="flex-1 overflow-auto">
          <ModuleErrorBoundary>
            <PageTransitionOutlet />
          </ModuleErrorBoundary>
        </div>
      </main>
    </div>
  )
}
