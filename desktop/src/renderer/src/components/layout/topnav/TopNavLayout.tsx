import { OfflineBanner } from '../OfflineBanner'
import { ModuleErrorBoundary } from '../ModuleShell'
import { PageTransitionOutlet } from '../PageTransitionOutlet'
import { TopNavBar } from './TopNavBar'

export function TopNavLayout() {
  return (
    <div className="flex h-full flex-col bg-background overflow-hidden glass-surface">
      <TopNavBar />
      <main className="flex-1 overflow-auto">
        <OfflineBanner />
        <ModuleErrorBoundary>
          <PageTransitionOutlet />
        </ModuleErrorBoundary>
      </main>
    </div>
  )
}
