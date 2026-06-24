/**
 * Tenant module-activation hooks (A-3). react-query over `/api/v1/admin/license`.
 * Mock-persisted today; swap-ready for the real licensing service (Luke 🔒).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'
import type { LicenseResponse, TenantModule, ToggleModuleInput } from '@/api/admin-types'

const LICENSE_KEY = ['admin', 'license'] as const

export function useTenantModules() {
  return useQuery<TenantModule[]>({
    queryKey: LICENSE_KEY,
    queryFn: async () => {
      const data = await authenticatedRequest<LicenseResponse>({
        method: 'GET',
        path: '/api/v1/admin/license',
      })
      return data.modules ?? []
    },
  })
}

export function useToggleModule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ToggleModuleInput) =>
      authenticatedRequest<{ module: TenantModule }>({
        method: 'PATCH',
        path: '/api/v1/admin/license',
        body: input,
      }),
    onMutate: async ({ moduleId, active }) => {
      await qc.cancelQueries({ queryKey: LICENSE_KEY })
      const prev = qc.getQueryData<TenantModule[]>(LICENSE_KEY)
      if (prev) {
        qc.setQueryData<TenantModule[]>(
          LICENSE_KEY,
          prev.map((m) =>
            m.moduleId === moduleId
              ? { ...m, active, assignedSeats: active ? m.assignedSeats : 0 }
              : m,
          ),
        )
      }
      return { prev }
    },
    onError: (_e, _i, ctx) => {
      if (ctx?.prev) qc.setQueryData(LICENSE_KEY, ctx.prev)
    },
    onSettled: () => qc.invalidateQueries({ queryKey: LICENSE_KEY }),
  })
}
