/**
 * Effective-permissions store (RBAC R-1).
 *
 * Holds the resolved capability map of the signed-in account (multi-role
 * union, server-resolved). Default-deny: a capability that is not in the map
 * is denied; while nothing valid is loaded, every check denies (pessimistic,
 * ra-rbac pattern).
 *
 * Persistence follows the settings-store pattern: localStorage (`persist`) is
 * the optimistic cache so the nav doesn't flash empty on app start; the cache
 * only applies while `forUserId` matches the authenticated user.
 *
 * Server-first with client fallback: until Luke ships
 * `GET /api/v1/auth/me/permissions`, a real backend 404s — we then resolve the
 * preset grants locally from the user's role names (ROLE_DEFS). Enforcement is
 * FE-only in R-1 either way; the server stays the source of truth once live.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { CapabilityGrant, EffectivePermissions } from '@/api/rbac-types'
import { fetchEffectivePermissions } from '@/api/rbac-client'
import type { RoleId } from '@/config/roles'
import { ROLE_DEFS, resolveCapabilities } from '@/mocks/data/rbac'
import { useAuthStore } from '@/stores/auth'

type PermissionsStatus = 'idle' | 'loading' | 'ready' | 'error'

/** Runtime-only overlay preview ("Als Rolle anzeigen", R-2 editor). */
export interface PermissionsPreview {
  /** Display label shown in the preview banner (role name). */
  label: string
  capabilities: Record<string, CapabilityGrant>
}

interface PermissionsState {
  /** Account the loaded permission set belongs to (guards the persisted cache). */
  forUserId: string | null
  roles: EffectivePermissions['roles']
  capabilities: Record<string, CapabilityGrant>
  status: PermissionsStatus
  /** Session guard — initFromServer() runs once, refresh() forces. */
  serverInitialized: boolean
  /**
   * Overlay preview: while set, capability checks resolve against this map
   * instead of the account's own — the user stays signed in (never persisted).
   */
  preview: PermissionsPreview | null
  /** Hydrate once per session (central hydrator). */
  initFromServer: () => Promise<void>
  /** Force reload (profile switch / role change). */
  refresh: () => Promise<void>
  startPreview: (preview: PermissionsPreview) => void
  endPreview: () => void
}

/** Map backend role-name strings onto known preset ids; unknown → least-privilege member. */
function toKnownRoleIds(roleNames: string[]): RoleId[] {
  const known = roleNames.filter((r): r is RoleId => r in ROLE_DEFS)
  return known.length > 0 ? known : ['member']
}

/** Local fallback resolution from the auth user's role names (pre-Luke backends). */
function fallbackPermissions(roleNames: string[]): EffectivePermissions {
  const roleIds = toKnownRoleIds(roleNames)
  return {
    roles: roleIds.map((id) => ({
      id,
      name: ROLE_DEFS[id].name,
      isSystem: true,
      color: ROLE_DEFS[id].color,
    })),
    capabilities: resolveCapabilities(roleIds),
  }
}

export const usePermissionsStore = create<PermissionsState>()(
  persist(
    (set, get) => ({
      forUserId: null,
      roles: [],
      capabilities: {},
      status: 'idle',
      serverInitialized: false,
      preview: null,

      initFromServer: async () => {
        if (get().serverInitialized) return
        await get().refresh()
      },

      startPreview: (preview) => set({ preview }),
      endPreview: () => set({ preview: null }),

      refresh: async () => {
        const user = useAuthStore.getState().user
        if (!user) return
        set({ status: 'loading' })
        try {
          const permissions = await fetchEffectivePermissions()
          set({
            forUserId: user.id,
            roles: permissions.roles,
            capabilities: permissions.capabilities,
            status: 'ready',
            serverInitialized: true,
          })
        } catch {
          // Endpoint missing (real backend pre-Luke) or offline — resolve
          // the preset grants locally so the app stays usable.
          const fallback = fallbackPermissions(user.roles)
          set({
            forUserId: user.id,
            roles: fallback.roles,
            capabilities: fallback.capabilities,
            status: 'ready',
            serverInitialized: true,
          })
        }
      },
    }),
    {
      name: 'cosmi-permissions',
      partialize: (s) => ({
        forUserId: s.forUserId,
        roles: s.roles,
        capabilities: s.capabilities,
      }),
    },
  ),
)

// Reload whenever the authenticated account changes (login, dev bypass,
// profile switcher). This also covers the startup race where the central
// hydrator fires before the auth store has a user — refresh() no-ops on
// null and this subscription catches the user arriving later. The persisted
// cache renders instantly; the fetch revalidates it.
useAuthStore.subscribe((state, prev) => {
  const id = state.user?.id
  if (id && id !== prev.user?.id) {
    // A different account never inherits an active overlay preview.
    usePermissionsStore.getState().endPreview()
    void usePermissionsStore.getState().refresh()
  }
})
