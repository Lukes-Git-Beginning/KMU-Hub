/**
 * RBAC permission hooks (A-2). react-query over `/api/v1/admin/permissions`.
 * The matrix is mock-persisted (MSW) today; the contract is swap-ready for the
 * gateway RBAC service (Luke's track 🔒). Toggles update optimistically.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'
import type { PermissionsResponse, SetPermissionInput } from '@/api/admin-types'

const PERMISSIONS_KEY = ['admin', 'permissions'] as const

export function useRolePermissions() {
  return useQuery<PermissionsResponse>({
    queryKey: PERMISSIONS_KEY,
    queryFn: () =>
      authenticatedRequest<PermissionsResponse>({
        method: 'GET',
        path: '/api/v1/admin/permissions',
      }),
  })
}

export function useSetPermission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: SetPermissionInput) =>
      authenticatedRequest<{ matrix: PermissionsResponse['matrix'] }>({
        method: 'PATCH',
        path: '/api/v1/admin/permissions',
        body: input,
      }),
    // Optimistic toggle so the checkbox feels instant.
    onMutate: async ({ capabilityId, role, granted }) => {
      await qc.cancelQueries({ queryKey: PERMISSIONS_KEY })
      const prev = qc.getQueryData<PermissionsResponse>(PERMISSIONS_KEY)
      if (prev) {
        const row = { ...(prev.matrix[capabilityId] ?? { admin: true }) }
        if (granted) row[role] = true
        else delete row[role]
        qc.setQueryData<PermissionsResponse>(PERMISSIONS_KEY, {
          ...prev,
          matrix: { ...prev.matrix, [capabilityId]: row },
        })
      }
      return { prev }
    },
    onError: (_err, _input, ctx) => {
      if (ctx?.prev) qc.setQueryData(PERMISSIONS_KEY, ctx.prev)
    },
    onSettled: () => qc.invalidateQueries({ queryKey: PERMISSIONS_KEY }),
  })
}
