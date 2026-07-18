/**
 * RBAC client (R-1) — effective permissions + role list.
 *
 * Endpoints are app-owned contract (rbac-types.ts); MSW serves them in demo
 * mode, Luke's gateway will serve them for real (backend-gaps §RBAC:
 * `GET /api/v1/auth/me/permissions`, `GET /api/v1/admin/roles`).
 */
import { authenticatedRequest } from '@/api/utils/authenticatedFetch'
import type {
  EffectivePermissions,
  EffectivePermissionsResponse,
  Role,
  RolesResponse,
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

/** All roles of the tenant (system presets + custom roles). Throws on miss. */
export async function fetchRoles(): Promise<Role[]> {
  const resp = await authenticatedRequest<RolesResponse>({
    method: 'GET',
    path: '/api/v1/admin/roles',
  })
  return resp?.roles ?? []
}
