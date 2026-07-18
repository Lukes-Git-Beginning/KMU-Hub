/**
 * useCapability (RBAC R-1) — the one way UI code asks "may the signed-in
 * account do X?".
 *
 * Default-deny: a key missing from the effective map is denied; while no valid
 * permission set is loaded for the current user, every check denies
 * (pessimistic). Convention for gated UI (CAPABILITY-KATALOG §Arbeitsanweisung):
 * security-relevant → hide, workflow-relevant → disable with tooltip.
 *
 * Key format: `resource:action` (backend composition), level-1 visibility via
 * `<module>:module:view` — see config/capabilities.ts.
 */
import { useMemo } from 'react'
import { useAuthStore } from '@/stores/auth'
import { usePermissionsStore } from '@/stores/permissions'
import type { CapabilityGrant, CapabilityScope } from '@/api/rbac-types'
import { moduleViewKey, type ModuleKey } from '@/config/capabilities'

export interface CapabilityResult {
  allowed: boolean
  /** Data scope of the grant (null while denied). */
  scope: CapabilityScope | null
  /** Role ids the grant derives from (provenance; empty while denied). */
  sources: string[]
  /** True only while the first load for this user is still in flight. */
  isLoading: boolean
}

/** Full capability check with scope + provenance. */
export function useCapability(key: string): CapabilityResult {
  const userId = useAuthStore((s) => s.user?.id ?? null)
  const forUserId = usePermissionsStore((s) => s.forUserId)
  const status = usePermissionsStore((s) => s.status)
  const grant: CapabilityGrant | undefined = usePermissionsStore((s) => s.capabilities[key])

  const valid = userId !== null && forUserId === userId
  const allowed = valid && grant !== undefined
  return {
    allowed,
    scope: allowed && grant ? grant.scope : null,
    sources: allowed && grant ? grant.sources : [],
    isLoading: !valid && status === 'loading',
  }
}

/** Boolean shorthand. */
export function useHasCapability(key: string): boolean {
  return useCapability(key).allowed
}

/** Level-1 module visibility (`<module>:module:view`). */
export function useModuleVisible(module: ModuleKey): boolean {
  return useHasCapability(moduleViewKey(module))
}

/**
 * Capability set for filter loops (nav items, tab lists) — one subscription,
 * many checks. `ready` is false until a permission set valid for the current
 * user is present (initial load on a fresh profile).
 */
export function useCapabilitySet(): { has: (key: string) => boolean; ready: boolean } {
  const userId = useAuthStore((s) => s.user?.id ?? null)
  const forUserId = usePermissionsStore((s) => s.forUserId)
  const capabilities = usePermissionsStore((s) => s.capabilities)

  return useMemo(() => {
    const valid = userId !== null && forUserId === userId
    return {
      has: (key: string) => valid && key in capabilities,
      ready: valid,
    }
  }, [userId, forUserId, capabilities])
}
