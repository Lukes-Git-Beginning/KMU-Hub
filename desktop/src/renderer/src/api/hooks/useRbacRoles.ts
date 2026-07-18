/**
 * RBAC role-builder hooks (R-2) — react-query over the rbac-client CRUD.
 *
 * After any mutation that can affect the signed-in account (grants of a role
 * it holds, its assignments), callers must refresh the permissions store so
 * gating updates without re-login: `usePermissionsStore.getState().refresh()`.
 * The hooks here handle cache invalidation only.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  assignUserRole,
  createRole,
  deleteRole,
  fetchRolePermissions,
  fetchRoles,
  fetchUserEffectivePermissions,
  putRolePermissions,
  removeUserRole,
  updateRole,
} from '@/api/rbac-client'
import type { CreateRoleInput, UpdateRoleInput, UpdateRolePermissionsInput } from '@/api/rbac-types'

const ROLES_KEY = ['rbac', 'roles'] as const
const roleGrantsKey = (roleId: string) => ['rbac', 'roles', roleId, 'permissions'] as const
const userPermissionsKey = (userId: string) => ['rbac', 'users', userId, 'permissions'] as const

/** All tenant roles (7 presets + custom). */
export function useRoles() {
  return useQuery({ queryKey: ROLES_KEY, queryFn: fetchRoles })
}

/** Grant map of one role (editor + compare + diff-vs-preset). */
export function useRoleGrants(roleId: string | null) {
  return useQuery({
    queryKey: roleGrantsKey(roleId ?? 'none'),
    queryFn: () => fetchRolePermissions(roleId as string),
    enabled: roleId !== null,
  })
}

/** Effective permissions of any account (team/user-detail audit view). */
export function useUserEffectivePermissions(userId: string | null) {
  return useQuery({
    queryKey: userPermissionsKey(userId ?? 'none'),
    queryFn: () => fetchUserEffectivePermissions(userId as string),
    enabled: userId !== null,
  })
}

export function useCreateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateRoleInput) => createRole(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ROLES_KEY }),
  })
}

export function useUpdateRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ roleId, input }: { roleId: string; input: UpdateRoleInput }) =>
      updateRole(roleId, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ROLES_KEY }),
  })
}

export function useDeleteRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (roleId: string) => deleteRole(roleId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ROLES_KEY }),
  })
}

export function useSetRoleGrants() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ roleId, input }: { roleId: string; input: UpdateRolePermissionsInput }) =>
      putRolePermissions(roleId, input),
    onSuccess: (_grants, { roleId }) => {
      void qc.invalidateQueries({ queryKey: roleGrantsKey(roleId) })
      // capabilityCount in the list + every effective view derived from it
      void qc.invalidateQueries({ queryKey: ROLES_KEY })
      void qc.invalidateQueries({ queryKey: ['rbac', 'users'] })
    },
  })
}

export function useAssignUserRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, roleId }: { userId: string; roleId: string }) =>
      assignUserRole(userId, roleId),
    onSuccess: (_roles, { userId }) => {
      void qc.invalidateQueries({ queryKey: userPermissionsKey(userId) })
      void qc.invalidateQueries({ queryKey: ROLES_KEY })
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

export function useRemoveUserRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, roleId }: { userId: string; roleId: string }) =>
      removeUserRole(userId, roleId),
    onSuccess: (_roles, { userId }) => {
      void qc.invalidateQueries({ queryKey: userPermissionsKey(userId) })
      void qc.invalidateQueries({ queryKey: ROLES_KEY })
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}
