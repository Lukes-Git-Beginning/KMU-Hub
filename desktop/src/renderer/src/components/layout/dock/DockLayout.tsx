import { Header } from '../Header'
import { OfflineBanner } from '../OfflineBanner'
import { ModuleErrorBoundary } from '../ModuleShell'
import { PageTransitionOutlet } from '../PageTransitionOutlet'
import { DockBar } from './DockBar'

export function DockLayout() {
  return (
    <div className="flex h-full flex-col bg-background overflow-hidden glass-surface">
      <Header />

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
