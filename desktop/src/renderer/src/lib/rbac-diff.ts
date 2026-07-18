/**
 * Grant-map diffing (R-2) — powers the "X Abweichungen vom Standard" badge,
 * the editor's deviation markers and the role compare view.
 */
import type { RoleGrants } from '@/api/rbac-types'

export interface GrantsDiff {
  /** Keys granted here but not in the base. */
  added: string[]
  /** Keys the base grants but this map doesn't. */
  removed: string[]
  /** Keys granted in both, with different scopes. */
  scopeChanged: Array<{ key: string; from: string; to: string }>
}

/** Diff `grants` against `base` (base = preset for deviations, server state for drafts). */
export function diffGrants(grants: RoleGrants, base: RoleGrants): GrantsDiff {
  const added: string[] = []
  const removed: string[] = []
  const scopeChanged: GrantsDiff['scopeChanged'] = []
  for (const [key, grant] of Object.entries(grants)) {
    const baseGrant = base[key]
    if (!baseGrant) added.push(key)
    else if (baseGrant.scope !== grant.scope) {
      scopeChanged.push({ key, from: baseGrant.scope, to: grant.scope })
    }
  }
  for (const key of Object.keys(base)) {
    if (!grants[key]) removed.push(key)
  }
  return { added, removed, scopeChanged }
}

export function diffCount(diff: GrantsDiff): number {
  return diff.added.length + diff.removed.length + diff.scopeChanged.length
}

/** Deviating keys as a set (editor row markers). */
export function diffKeySet(diff: GrantsDiff): Set<string> {
  return new Set([...diff.added, ...diff.removed, ...diff.scopeChanged.map((c) => c.key)])
}
