/**
 * useModuleAssignments.ts — TanStack Query hooks for the Module-Assignment Tab.
 *
 * Backed by the real /api/v1/tenant/module-grants endpoints (R3-E6). Mutations
 * apply an optimistic cache update for snappy UX, roll back on error, and
 * invalidate on settle so the server stays the source of truth.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'
import type { UserModuleGrant, ModuleId } from '@/lib/pricing'

// ───────────────────────── Query Keys ─────────────────────────

const MA_KEYS = {
  all: ['moduleAssignments', 'grants'] as const,
  byUser: (userId: string) => ['moduleAssignments', 'grants', 'user', userId] as const,
}

const GRANTS_PATH = '/api/v1/tenant/module-grants'

// Backend grant shape (gateway HandleListModuleGrants). moduleId is widened to
// ModuleId on the way in.
interface RawGrant {
  userId: string
  userName: string
  moduleId: string
  grantedAt: string
  lastActiveAt: string | null
}

function toGrant(g: RawGrant): UserModuleGrant {
  return {
    userId: g.userId,
    userName: g.userName,
    moduleId: g.moduleId as ModuleId,
    grantedAt: g.grantedAt,
    lastActiveAt: g.lastActiveAt,
  }
}

// ───────────────────────── Hooks ─────────────────────────

/** All UserModuleGrant entries for the tenant. */
export function useModuleAssignments() {
  return useQuery<UserModuleGrant[]>({
    queryKey: MA_KEYS.all,
    queryFn: async () => {
      const data = await authenticatedRequest<{ grants: RawGrant[] }>({
        method: 'GET',
        path: GRANTS_PATH,
      })
      return (data.grants ?? []).map(toGrant)
    },
  })
}

/**
 * Grants filtered to a single user.
 * Derived from the main cache — always consistent.
 */
export function useUserModules(userId: string) {
  const { data: allGrants = [], ...rest } = useModuleAssignments()
  const userGrants = allGrants.filter((g) => g.userId === userId)
  return { data: userGrants, ...rest }
}

// ───────────────────────── Mutations ─────────────────────────

export interface GrantModuleArgs {
  userId: string
  userName: string
  moduleId: ModuleId
}

/** Add a module grant for a user (optimistic, server-persisted). */
export function useGrantModule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (args: GrantModuleArgs) => {
      await authenticatedRequest({
        method: 'PUT',
        path: `${GRANTS_PATH}/${args.userId}/${args.moduleId}`,
      })
    },
    onMutate: async (args) => {
      await queryClient.cancelQueries({ queryKey: MA_KEYS.all })
      const previous = queryClient.getQueryData<UserModuleGrant[]>(MA_KEYS.all)

      const newGrant: UserModuleGrant = {
        userId: args.userId,
        userName: args.userName,
        moduleId: args.moduleId,
        grantedAt: new Date().toISOString(),
        lastActiveAt: null,
      }
      const base = previous ?? []
      // Avoid a duplicate row if the grant already exists.
      const exists = base.some((g) => g.userId === args.userId && g.moduleId === args.moduleId)
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, exists ? base : [...base, newGrant])

      return { previous }
    },
    onError: (_err, _args, context) => {
      if (context?.previous) {
        queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, context.previous)
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: MA_KEYS.all })
    },
  })
}

export interface RevokeModuleArgs {
  userId: string
  moduleId: ModuleId
  /**
   * Optional: true wenn die Revocation aus einer Billing-Recommendation stammt.
   * Nur dann soll der Cost-Savings-Counter aktualisiert werden (kein Doppel-Counting
   * bei manuellen Admin-Entzügen).
   */
  fromRecommendation?: boolean
}

/** Remove a module grant for a user (optimistic, server-persisted). */
export function useRevokeModule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (args: RevokeModuleArgs) => {
      await authenticatedRequest({
        method: 'DELETE',
        path: `${GRANTS_PATH}/${args.userId}/${args.moduleId}`,
      })
    },
    onMutate: async (args) => {
      await queryClient.cancelQueries({ queryKey: MA_KEYS.all })
      const previous = queryClient.getQueryData<UserModuleGrant[]>(MA_KEYS.all) ?? []

      const next = previous.filter(
        (g) => !(g.userId === args.userId && g.moduleId === args.moduleId),
      )
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, next)

      return { previous }
    },
    onError: (_err, _args, context) => {
      if (context?.previous) {
        queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, context.previous)
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: MA_KEYS.all })
    },
  })
}

export interface BulkRevokeArgs {
  pairs: { userId: string; moduleId: ModuleId }[]
  /**
   * Optional: true wenn der Bulk-Revoke aus einer Billing-Recommendation stammt.
   * Nur dann soll der Cost-Savings-Counter aktualisiert werden.
   */
  fromRecommendation?: boolean
}

/** Revoke multiple grants at once (optimistic, server-persisted). */
export function useBulkRevokeInactive() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (args: BulkRevokeArgs) => {
      if (args.pairs.length === 0) return
      await authenticatedRequest({
        method: 'POST',
        path: `${GRANTS_PATH}/bulk-revoke`,
        body: { pairs: args.pairs },
      })
    },
    onMutate: async (args) => {
      await queryClient.cancelQueries({ queryKey: MA_KEYS.all })
      const previous = queryClient.getQueryData<UserModuleGrant[]>(MA_KEYS.all) ?? []

      const pairSet = new Set(args.pairs.map((p) => `${p.userId}:${p.moduleId}`))
      const next = previous.filter((g) => !pairSet.has(`${g.userId}:${g.moduleId}`))
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, next)

      return { previous }
    },
    onError: (_err, _args, context) => {
      if (context?.previous) {
        queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, context.previous)
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: MA_KEYS.all })
    },
  })
}

/** Restore a set of grants (used by undo-toast logic) — re-grants each. */
export function useRestoreGrants() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (grants: UserModuleGrant[]) => {
      await Promise.all(
        grants.map((g) =>
          authenticatedRequest({
            method: 'PUT',
            path: `${GRANTS_PATH}/${g.userId}/${g.moduleId}`,
          }),
        ),
      )
    },
    onMutate: async (grants) => {
      await queryClient.cancelQueries({ queryKey: MA_KEYS.all })
      const current = queryClient.getQueryData<UserModuleGrant[]>(MA_KEYS.all) ?? []

      // Add back (avoid duplicates)
      const existingKeys = new Set(current.map((g) => `${g.userId}:${g.moduleId}`))
      const toAdd = grants.filter((g) => !existingKeys.has(`${g.userId}:${g.moduleId}`))
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, [...current, ...toAdd])

      return { previous: current }
    },
    onError: (_err, _grants, context) => {
      if (context?.previous) {
        queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, context.previous)
      }
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: MA_KEYS.all })
    },
  })
}
