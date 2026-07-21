/**
 * RBAC client (R-1) — effective permissions + role list.
 *
 * Endpoints are app-owned contract (rbac-types.ts); MSW serves them in demo
 * mode, Luke's gateway will serve them for real (backend-gaps §RBAC:
 * `GET /api/v1/auth/me/permissions`, `GET /api/v1/admin/roles`).
 */
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'
import type {
  AssignRoleInput,
  CreateRoleInput,
  EffectivePermissions,
  EffectivePermissionsResponse,
  Role,
  RoleGrants,
  RolePermissionsResponse,
  RoleResponse,
  RolesResponse,
  UpdateRoleInput,
  UpdateRolePermissionsInput,
  UpdateUserOverridesInput,
  UserOverrides,
  UserOverridesResponse,
  UserRolesResponse,
} from '@/api/rbac-types'

/** Resolved effective permissions of the authenticated account. Throws on miss. */
export async function fetchEffectivePermissions(): Promise<EffectivePermissions> {
  const resp = await authenticatedRequest<EffectivePermissionsResponse>({
    method: 'GET',
    path: '/api/v1/auth/me/permissions',
  })
  if (!resp?.permissions) throw new Error('empty permissions response')
  return resp.permissions
}

/**
 * Resolved effective permissions of ANY account (admin/HR audit view).
 * `base: true` returns the pure role union without per-user overrides (the
 * grayed "inherited" baseline in the R-6 override editor).
 */
export async function fetchUserEffectivePermissions(
  userId: string,
  opts?: { base?: boolean },
): Promise<EffectivePermissions> {
  const resp = await authenticatedRequest<EffectivePermissionsResponse>({
    method: 'GET',
    path: `/api/v1/admin/users/${userId}/permissions${opts?.base ? '?base=1' : ''}`,
  })
  if (!resp?.permissions) throw new Error('empty permissions response')
  return resp.permissions
}

/** All roles of the tenant (system presets + custom roles). Throws on miss. */
export async function fetchRoles(): Promise<Role[]> {
  const resp = await authenticatedRequest<RolesResponse>({
    method: 'GET',
    path: '/api/v1/admin/roles',
  })
  return resp?.roles ?? []
}

// ── R-2 role builder (CRUD + grants + assignment) ───────────────────────────

/** Grant map of one role (capability key → scope). */
export async function fetchRolePermissions(roleId: string): Promise<RoleGrants> {
  const resp = await authenticatedRequest<RolePermissionsResponse>({
    method: 'GET',
    path: `/api/v1/admin/roles/${roleId}/permissions`,
  })
  return resp?.grants ?? {}
}

/** Create a custom role as a clone of `basedOn`. */
export async function createRole(input: CreateRoleInput): Promise<Role> {
  const resp = await authenticatedRequest<RoleResponse>({
    method: 'POST',
    path: '/api/v1/admin/roles',
    body: input,
  })
  return resp.role
}

/** Update name/description/color of a custom role. */
export async function updateRole(roleId: string, input: UpdateRoleInput): Promise<Role> {
  const resp = await authenticatedRequest<RoleResponse>({
    method: 'PATCH',
    path: `/api/v1/admin/roles/${roleId}`,
    body: input,
  })
  return resp.role
}

/** Delete a custom role (server rejects presets and roles with members). */
export async function deleteRole(roleId: string): Promise<void> {
  await authenticatedRequest<void>({
    method: 'DELETE',
    path: `/api/v1/admin/roles/${roleId}`,
  })
}

/** Replace the full grant map of a custom role. */
export async function putRolePermissions(
  roleId: string,
  input: UpdateRolePermissionsInput,
): Promise<RoleGrants> {
  const resp = await authenticatedRequest<RolePermissionsResponse>({
    method: 'PUT',
    path: `/api/v1/admin/roles/${roleId}/permissions`,
    body: input,
  })
  return resp?.grants ?? {}
}

/** Assign a role to an account (n:m — existing roles stay). */
export async function assignUserRole(userId: string, roleId: string): Promise<string[]> {
  const body: AssignRoleInput = { roleId }
  const resp = await authenticatedRequest<UserRolesResponse>({
    method: 'POST',
    path: `/api/v1/users/${userId}/roles`,
    body,
  })
  return resp?.roles ?? []
}

/** Remove a role from an account. */
export async function removeUserRole(userId: string, roleId: string): Promise<string[]> {
  const resp = await authenticatedRequest<UserRolesResponse>({
    method: 'DELETE',
    path: `/api/v1/users/${userId}/roles/${roleId}`,
  })
  return resp?.roles ?? []
}

// ── R-6 per-user overrides ──────────────────────────────────────────────────

/** Current override map of an account (capability key → allow/deny + scope). */
export async function fetchUserOverrides(userId: string): Promise<UserOverrides> {
  const resp = await authenticatedRequest<UserOverridesResponse>({
    method: 'GET',
    path: `/api/v1/admin/users/${userId}/overrides`,
  })
  return resp?.overrides ?? {}
}

/** Replace the whole override map of an account (empty map clears all). */
export async function putUserOverrides(
  userId: string,
  overrides: UserOverrides,
): Promise<UserOverrides> {
  const body: UpdateUserOverridesInput = { overrides }
  const resp = await authenticatedRequest<UserOverridesResponse>({
    method: 'PUT',
    path: `/api/v1/admin/users/${userId}/overrides`,
    body,
  })
  return resp?.overrides ?? {}
}
