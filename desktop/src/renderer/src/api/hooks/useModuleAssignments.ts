/**
 * useModuleAssignments.ts — TanStack Query hooks for Module-Assignment Tab (Phase 2).
 *
 * No backend calls. Uses MOCK_GRANTS from billing-mock.ts as source of truth.
 * Mutations operate on in-memory QueryClient cache via setQueryData.
 * staleTime: Infinity — data only changes through mutations.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { MOCK_GRANTS } from '@/mocks/billing-mock'
import type { UserModuleGrant, ModuleId } from '@/lib/pricing'

// ───────────────────────── Query Keys ─────────────────────────

const MA_KEYS = {
  all: ['moduleAssignments', 'grants'] as const,
  byUser: (userId: string) => ['moduleAssignments', 'grants', 'user', userId] as const,
}

// ───────────────────────── Seed data ─────────────────────────

// Initialise once; subsequent calls read from QueryClient cache.
let seeded = false
let _cachedGrants: UserModuleGrant[] = [...MOCK_GRANTS]

function getGrants(): UserModuleGrant[] {
  return _cachedGrants
}

// ───────────────────────── Hooks ─────────────────────────

/**
 * All UserModuleGrant entries.
 * Initial data from MOCK_GRANTS; mutated in-memory via queryClient.setQueryData.
 */
export function useModuleAssignments() {
  return useQuery<UserModuleGrant[]>({
    queryKey: MA_KEYS.all,
    queryFn: () => {
      if (!seeded) {
        seeded = true
      }
      return Promise.resolve(getGrants())
    },
    staleTime: Infinity,
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

/** Add a module grant for a user (optimistic). */
export function useGrantModule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (args: GrantModuleArgs): Promise<UserModuleGrant> => {
      const newGrant: UserModuleGrant = {
        userId: args.userId,
        userName: args.userName,
        moduleId: args.moduleId,
        grantedAt: new Date().toISOString(),
        lastActiveAt: null,
      }
      return newGrant
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

      const next = [...(previous ?? _cachedGrants), newGrant]
      _cachedGrants = next
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, next)

      return { previous }
    },
    onError: (_err, _args, context) => {
      if (context?.previous) {
        _cachedGrants = context.previous
        queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, context.previous)
      }
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

/** Remove a module grant for a user (optimistic). */
export function useRevokeModule() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (_args: RevokeModuleArgs) => undefined,
    onMutate: async (args) => {
      await queryClient.cancelQueries({ queryKey: MA_KEYS.all })
      const previous = queryClient.getQueryData<UserModuleGrant[]>(MA_KEYS.all) ?? _cachedGrants

      const next = previous.filter(
        (g) => !(g.userId === args.userId && g.moduleId === args.moduleId),
      )
      _cachedGrants = next
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, next)

      return { previous }
    },
    onError: (_err, _args, context) => {
      if (context?.previous) {
        _cachedGrants = context.previous
        queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, context.previous)
      }
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

/** Revoke multiple grants at once (optimistic). */
export function useBulkRevokeInactive() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (_args: BulkRevokeArgs) => undefined,
    onMutate: async (args) => {
      await queryClient.cancelQueries({ queryKey: MA_KEYS.all })
      const previous = queryClient.getQueryData<UserModuleGrant[]>(MA_KEYS.all) ?? _cachedGrants

      const pairSet = new Set(args.pairs.map((p) => `${p.userId}:${p.moduleId}`))
      const next = previous.filter((g) => !pairSet.has(`${g.userId}:${g.moduleId}`))
      _cachedGrants = next
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, next)

      return { previous }
    },
    onError: (_err, _args, context) => {
      if (context?.previous) {
        _cachedGrants = context.previous
        queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, context.previous)
      }
    },
  })
}

/** Restore a set of grants (used by undo-toast logic). */
export function useRestoreGrants() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (_grants: UserModuleGrant[]) => undefined,
    onMutate: async (grants) => {
      await queryClient.cancelQueries({ queryKey: MA_KEYS.all })
      const current = queryClient.getQueryData<UserModuleGrant[]>(MA_KEYS.all) ?? _cachedGrants

      // Add back (avoid duplicates)
      const existingKeys = new Set(current.map((g) => `${g.userId}:${g.moduleId}`))
      const toAdd = grants.filter((g) => !existingKeys.has(`${g.userId}:${g.moduleId}`))
      const next = [...current, ...toAdd]
      _cachedGrants = next
      queryClient.setQueryData<UserModuleGrant[]>(MA_KEYS.all, next)
    },
  })
}
