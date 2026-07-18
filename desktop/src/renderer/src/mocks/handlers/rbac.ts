/**
 * RBAC handlers — effective permissions, role CRUD (R-2 builder), per-role
 * grants and user↔role assignment. Stateful against ../data/rbac.ts; the
 * behaviour (incl. guardrails) mirrors what Luke's API must do server-side
 * (backend-gaps §RBAC).
 *
 * Error contract: mutations reject with `{ error: <code> }` where code is one
 * of preset_immutable | role_limit_reached | role_name_exists |
 * role_has_members | last_admin | not_found — the FE maps codes to i18n.
 */
import { http, HttpResponse } from 'msw'
import { API_BASE_URL } from '@/lib/constants'
import type {
  AssignRoleInput,
  CreateRoleInput,
  EffectivePermissionsResponse,
  RolePermissionsResponse,
  RolesResponse,
  UpdateRoleInput,
  UpdateRolePermissionsInput,
} from '@/api/rbac-types'
import type { CapabilityScope } from '@/api/rbac-types'
import {
  CUSTOM_ROLE_LIMIT,
  createCustomRole,
  customRoleCount,
  deleteCustomRole,
  fullAccessHolderCount,
  getCustomRole,
  getDemoSessionUserId,
  getRoleGrants,
  isPresetRole,
  listRoles,
  membersOfRole,
  removeRoleFromUser,
  resolveCapabilities,
  roleNameExists,
  rolesForUser,
  roleSummary,
  setCustomRoleGrants,
  assignRoleToUser,
  updateCustomRole,
} from '../data/rbac'

const API = API_BASE_URL

const error = (code: string, status: number) =>
  HttpResponse.json({ error: code }, { status })

function effectivePermissionsBody(userId: string): EffectivePermissionsResponse {
  const roleIds = rolesForUser(userId)
  return {
    permissions: {
      roles: roleIds.map(roleSummary),
      capabilities: resolveCapabilities(roleIds),
    },
  }
}

function roleFromList(roleId: string) {
  return listRoles().find((r) => r.id === roleId)
}

export const rbacHandlers = [
  // Effective permissions of the authenticated account (multi-role union).
  http.get(`${API}/api/v1/auth/me/permissions`, () =>
    HttpResponse.json(effectivePermissionsBody(getDemoSessionUserId())),
  ),

  // Effective permissions of ANY account (admin/HR view in team + user detail).
  http.get(`${API}/api/v1/admin/users/:userId/permissions`, ({ params }) =>
    HttpResponse.json(effectivePermissionsBody(String(params.userId))),
  ),

  // Role list (presets + tenant custom roles).
  http.get(`${API}/api/v1/admin/roles`, () => {
    const body: RolesResponse = { roles: listRoles() }
    return HttpResponse.json(body)
  }),

  // Grant map of one role.
  http.get(`${API}/api/v1/admin/roles/:roleId/permissions`, ({ params }) => {
    const roleId = String(params.roleId)
    const grants = getRoleGrants(roleId)
    if (!grants) return error('not_found', 404)
    const body: RolePermissionsResponse = {
      roleId,
      grants: Object.fromEntries(
        Object.entries(grants).map(([key, scope]) => [key, { scope }]),
      ),
    }
    return HttpResponse.json(body)
  }),

  // Create custom role (always a clone of basedOn).
  http.post(`${API}/api/v1/admin/roles`, async ({ request }) => {
    const input = (await request.json()) as CreateRoleInput
    if (!input?.name?.trim() || !input?.basedOn) return error('not_found', 422)
    if (!getRoleGrants(input.basedOn)) return error('not_found', 404)
    if (customRoleCount() >= CUSTOM_ROLE_LIMIT) return error('role_limit_reached', 409)
    if (roleNameExists(input.name)) return error('role_name_exists', 409)
    const def = createCustomRole(input)
    return HttpResponse.json({ role: roleFromList(def.id) }, { status: 201 })
  }),

  // Update custom role meta (presets are immutable).
  http.patch(`${API}/api/v1/admin/roles/:roleId`, async ({ params, request }) => {
    const roleId = String(params.roleId)
    if (isPresetRole(roleId)) return error('preset_immutable', 403)
    if (!getCustomRole(roleId)) return error('not_found', 404)
    const input = (await request.json()) as UpdateRoleInput
    if (input.name !== undefined && roleNameExists(input.name, roleId)) {
      return error('role_name_exists', 409)
    }
    updateCustomRole(roleId, input)
    return HttpResponse.json({ role: roleFromList(roleId) })
  }),

  // Delete custom role (presets immutable; roles with members blocked).
  http.delete(`${API}/api/v1/admin/roles/:roleId`, ({ params }) => {
    const roleId = String(params.roleId)
    if (isPresetRole(roleId)) return error('preset_immutable', 403)
    if (!getCustomRole(roleId)) return error('not_found', 404)
    if (membersOfRole(roleId).length > 0) return error('role_has_members', 409)
    deleteCustomRole(roleId)
    return new HttpResponse(null, { status: 204 })
  }),

  // Replace the grant map of a custom role.
  http.put(`${API}/api/v1/admin/roles/:roleId/permissions`, async ({ params, request }) => {
    const roleId = String(params.roleId)
    if (isPresetRole(roleId)) return error('preset_immutable', 403)
    if (!getCustomRole(roleId)) return error('not_found', 404)
    const input = (await request.json()) as UpdateRolePermissionsInput
    const grants: Record<string, CapabilityScope> = Object.fromEntries(
      Object.entries(input?.grants ?? {}).map(([key, g]) => [key, g.scope]),
    )
    setCustomRoleGrants(roleId, grants)
    const body: RolePermissionsResponse = { roleId, grants: input?.grants ?? {} }
    return HttpResponse.json(body)
  }),

  // Assign role to account (n:m, existing roles stay).
  http.post(`${API}/api/v1/users/:userId/roles`, async ({ params, request }) => {
    const userId = String(params.userId)
    const input = (await request.json()) as AssignRoleInput
    if (!input?.roleId || !getRoleGrants(input.roleId)) return error('not_found', 404)
    const roles = assignRoleToUser(userId, input.roleId)
    return HttpResponse.json({ roles })
  }),

  // Remove role from account — last-admin guardrail.
  http.delete(`${API}/api/v1/users/:userId/roles/:roleId`, ({ params }) => {
    const userId = String(params.userId)
    const roleId = String(params.roleId)
    if (roleId === 'admin' && rolesForUser(userId).includes('admin') && fullAccessHolderCount() <= 1) {
      return error('last_admin', 409)
    }
    const roles = removeRoleFromUser(userId, roleId)
    return HttpResponse.json({ roles })
  }),
]
