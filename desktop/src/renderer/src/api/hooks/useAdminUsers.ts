/**
 * Admin account hooks (A-1 Benutzerverwaltung).
 *
 * Thin react-query wrappers over the `/api/v1/admin/users` endpoints. The MSW
 * mock backs these today; the contract (see `@/api/admin-types`) is swap-ready
 * for the real account-provisioning service (Luke's track 🔒).
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'
import type { AdminUser, AdminUserStatus, InviteUserInput } from '@/api/admin-types'

const ADMIN_USERS_KEY = ['admin', 'users'] as const

/** All tenant accounts (active / invited / deactivated). */
export function useAdminUsers() {
  return useQuery<AdminUser[]>({
    queryKey: ADMIN_USERS_KEY,
    queryFn: async () => {
      const data = await authenticatedRequest<{ users: AdminUser[] }>({
        method: 'GET',
        path: '/api/v1/admin/users',
      })
      return data.users ?? []
    },
  })
}

/** Invite a new user — creates a pending account (status `invited`). */
export function useInviteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (input: InviteUserInput) => {
      const data = await authenticatedRequest<{ user: AdminUser }>({
        method: 'POST',
        path: '/api/v1/admin/users/invite',
        body: input,
      })
      return data.user
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ADMIN_USERS_KEY }),
  })
}

/** Change a user's roles and/or account status. */
export function useUpdateAdminUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (input: { id: string; roles?: string[]; status?: AdminUserStatus }) => {
      const { id, ...patch } = input
      const data = await authenticatedRequest<{ user: AdminUser }>({
        method: 'PATCH',
        path: `/api/v1/admin/users/${id}`,
        body: patch,
      })
      return data.user
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ADMIN_USERS_KEY }),
  })
}

/** Re-send a pending invite (refreshes the invite timestamp). */
export function useResendInvite() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const data = await authenticatedRequest<{ user: AdminUser }>({
        method: 'POST',
        path: `/api/v1/admin/users/${id}/resend-invite`,
      })
      return data.user
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ADMIN_USERS_KEY }),
  })
}
