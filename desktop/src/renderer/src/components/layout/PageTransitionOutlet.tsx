/**
 * Shared outlet wrapper with page transition animation.
 *
 * Wraps react-router's <Outlet /> with a fade+slide-up entrance
 * animation that re-triggers on every route change.
 * Used by all 4 navigation layouts.
 */
import { Suspense } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { usePageTransition } from '@/hooks/usePageTransition'
import { ModuleLoadingFallback } from './ModuleShell'

export function PageTransitionOutlet() {
  const location = useLocation()
  const transitionClass = usePageTransition(location.pathname)

  return (
    <div className={`h-full ${transitionClass}`}>
      <Suspense fallback={<ModuleLoadingFallback />}>
        <Outlet />
      </Suspense>
    </div>
  )
}
