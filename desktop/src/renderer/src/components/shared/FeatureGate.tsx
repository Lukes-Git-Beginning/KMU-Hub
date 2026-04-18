import type { ReactNode } from 'react'
import { useFeatureFlags } from '@/api/hooks/useFeatureFlags'

interface FeatureGateProps {
  /** Feature flag key, e.g. "modules.wiki" */
  flag: string
  /** Rendered while the flag is loading or when the flag is disabled. Defaults to null. */
  fallback?: ReactNode
  children: ReactNode
}

/**
 * Conditionally renders children based on a feature flag.
 *
 * While flags are loading: renders fallback (avoids flash of gated content).
 * When flag is disabled: renders fallback.
 * When flag is enabled: renders children.
 *
 * @example
 * <FeatureGate flag="modules.wiki">
 *   <WikiSidebarLink />
 * </FeatureGate>
 */
export function FeatureGate({ flag, fallback = null, children }: FeatureGateProps) {
  const { isLoading, isEnabled } = useFeatureFlags()

  if (isLoading) return <>{fallback}</>
  if (!isEnabled(flag)) return <>{fallback}</>

  return <>{children}</>
}
